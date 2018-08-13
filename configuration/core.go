// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package configuration

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Safing/safing-core/database"
	"github.com/Safing/safing-core/log"
	"github.com/Safing/safing-core/modules"
)

// think about:
// config changes validation (e.g. if on in secure mode, must be on in fortress mode)
// config switches
// small codebase
// nice api
// be static as much as possible

const (
	SecurityLevelOff      int8 = 0
	SecurityLevelDynamic  int8 = 1
	SecurityLevelSecure   int8 = 2
	SecurityLevelFortress int8 = 3

	CompetenceLevelNone      int8 = 0
	CompetenceLevelBasic     int8 = 1
	CompetenceLevelPowerUser int8 = 2
	CompetenceLevelExpert    int8 = 3

	StatusOk      int8 = 0
	StatusWarning int8 = 1
	StatusError   int8 = 2
)

var (
	configurationModule *modules.Module

	lastChange    *int64
	securityLevel *int32

	lock          sync.RWMutex
	status        *SystemStatus
	currentConfig *Configuration
)

func init() {
	configurationModule = modules.Register("Configuration", 128)

	initDefaultConfig()
	initSystemStatusModel()
	initConfigurationModel()

	lastChangeValue := time.Now().Unix()
	lastChange = &lastChangeValue

	var securityLevelValue int32
	securityLevel = &securityLevelValue

	var err error
	config, err := GetConfiguration(configurationInstanceName)
	if err != nil {
		log.Warningf("configuration: could not load configuration: %s", err)
		loadedConfig := defaultConfig
		config = &loadedConfig
		err = config.Create(configurationInstanceName)
		if err != nil {
			log.Warningf("configuration: could not save new configuration: %s", err)
		}
	}

	status, err = GetSystemStatus()
	if err != nil {
		log.Warningf("configuration: could not load status: %s", err)
		status = &SystemStatus{
			CurrentSecurityLevel:  1,
			SelectedSecurityLevel: 1,
		}
		err = status.Create()
		if err != nil {
			log.Warningf("configuration: could not save new status: %s", err)
		}
	}

	log.Infof("configuration: initial security level is [%s]", status.FmtSecurityLevel())
	// atomic.StoreInt32(securityLevel, int32(status.CurrentSecurityLevel))

	updateConfig(config)

	go configChangeListener()
	go statusChangeListener()
}

func configChangeListener() {
	sub := database.NewSubscription()
	sub.Subscribe(fmt.Sprintf("%s/Configuration:%s", database.Me.String(), configurationInstanceName))
	for {
		var receivedModel database.Model

		select {
		case <-configurationModule.Stop:
			configurationModule.StopComplete()
			return
		case receivedModel = <-sub.Updated:
		case receivedModel = <-sub.Created:
		}

		config, ok := database.SilentEnsureModel(receivedModel, configurationModel).(*Configuration)
		if !ok {
			log.Warning("configuration: received config update, but was not of type *Configuration")
			continue
		}

		updateConfig(config)

	}
}

func updateConfig(update *Configuration) {
	new := &Configuration{}

	if update.EnforceCT > 0 && update.EnforceCT < 4 {
		new.EnforceCT = update.EnforceCT
	} else {
		new.EnforceCT = defaultConfig.EnforceCT
	}
	if update.EnforceRevocation > 0 && update.EnforceRevocation < 4 {
		new.EnforceRevocation = update.EnforceRevocation
	} else {
		new.EnforceRevocation = defaultConfig.EnforceRevocation
	}
	if update.DenyInsecureTLS > 0 && update.DenyInsecureTLS < 4 {
		new.DenyInsecureTLS = update.DenyInsecureTLS
	} else {
		new.DenyInsecureTLS = defaultConfig.DenyInsecureTLS
	}
	if update.DenyTLSWithoutSNI > 0 && update.DenyTLSWithoutSNI < 4 {
		new.DenyTLSWithoutSNI = update.DenyTLSWithoutSNI
	} else {
		new.DenyTLSWithoutSNI = defaultConfig.DenyTLSWithoutSNI
	}
	if update.DoNotUseAssignedDNS > 0 && update.DoNotUseAssignedDNS < 4 {
		new.DoNotUseAssignedDNS = update.DoNotUseAssignedDNS
	} else {
		new.DoNotUseAssignedDNS = defaultConfig.DoNotUseAssignedDNS
	}
	if update.DoNotUseMDNS > 0 && update.DoNotUseMDNS < 4 {
		new.DoNotUseMDNS = update.DoNotUseMDNS
	} else {
		new.DoNotUseMDNS = defaultConfig.DoNotUseMDNS
	}
	if update.DoNotForwardSpecialDomains > 0 && update.DoNotForwardSpecialDomains < 4 {
		new.DoNotForwardSpecialDomains = update.DoNotForwardSpecialDomains
	} else {
		new.DoNotForwardSpecialDomains = defaultConfig.DoNotForwardSpecialDomains
	}
	if update.AlwaysPromptAtNewProfile > 0 && update.AlwaysPromptAtNewProfile < 4 {
		new.AlwaysPromptAtNewProfile = update.AlwaysPromptAtNewProfile
	} else {
		new.AlwaysPromptAtNewProfile = defaultConfig.AlwaysPromptAtNewProfile
	}
	if update.DenyNetworkUntilProfileApproved > 0 && update.DenyNetworkUntilProfileApproved < 4 {
		new.DenyNetworkUntilProfileApproved = update.DenyNetworkUntilProfileApproved
	} else {
		new.DenyNetworkUntilProfileApproved = defaultConfig.DenyNetworkUntilProfileApproved
	}

	// generic configuration
	if update.CompetenceLevel >= 0 && update.CompetenceLevel <= 3 {
		new.CompetenceLevel = update.CompetenceLevel
	} else {
		new.CompetenceLevel = 3
		// TODO: maybe notify user?
	}

	if len(update.DNSServers) != 0 {
		new.DNSServers = update.DNSServers
	} else {
		new.DNSServers = defaultConfig.DNSServers
	}

	if update.DNSServerRetryRate != 0 {
		new.DNSServerRetryRate = update.DNSServerRetryRate
	} else {
		new.DNSServerRetryRate = defaultConfig.DNSServerRetryRate
	}
	if len(update.CountryBlacklist) != 0 {
		new.CountryBlacklist = update.CountryBlacklist
	} else {
		new.CountryBlacklist = defaultConfig.CountryBlacklist
	}
	if len(update.ASBlacklist) != 0 {
		new.ASBlacklist = update.ASBlacklist
	} else {
		new.ASBlacklist = defaultConfig.ASBlacklist
	}

	lock.Lock()
	defer lock.Unlock()

	// set new config and update timestamp
	currentConfig = new
	atomic.StoreInt64(lastChange, time.Now().UnixNano())

	// update status with new values
	// status.CurrentSecurityLevel = currentConfig.SecurityLevel
	// status.Save()

	// update atomic securityLevel
	// atomic.StoreInt32(securityLevel, int32(currentConfig.SecurityLevel))
}

func statusChangeListener() {
	sub := database.NewSubscription()
	sub.Subscribe(fmt.Sprintf("%s/SystemStatus:%s", database.Me.String(), systemStatusInstanceName))
	for {
		var receivedModel database.Model

		select {
		case <-configurationModule.Stop:
			configurationModule.StopComplete()
			return
		case receivedModel = <-sub.Updated:
		case receivedModel = <-sub.Created:
		}

		status, ok := database.SilentEnsureModel(receivedModel, systemStatusModel).(*SystemStatus)
		if !ok {
			log.Warning("configuration: received system status update, but was not of type *SystemStatus")
			continue
		}

		atomic.StoreInt32(securityLevel, int32(status.CurrentSecurityLevel))
	}
}
