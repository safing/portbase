// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package configuration

import (
	"sync/atomic"

	"github.com/Safing/safing-core/database"

	datastore "github.com/ipfs/go-datastore"
)

type SecurityLevelBoolean int8

func (slb SecurityLevelBoolean) IsSet() bool {
	return int8(atomic.LoadInt32(securityLevel)) >= int8(slb)
}

func (slb SecurityLevelBoolean) IsSetWithLevel(customSecurityLevel int8) bool {
	return customSecurityLevel >= int8(slb) || int8(atomic.LoadInt32(securityLevel)) >= int8(slb)
}

func (slb SecurityLevelBoolean) Level() int8 {
	return int8(slb)
}

type Configuration struct {
	database.Base

	// Security Config
	EnforceCT                       SecurityLevelBoolean `json:",omitempty bson:",omitempty"` // Hardfail on Certificate Transparency
	EnforceRevocation               SecurityLevelBoolean `json:",omitempty bson:",omitempty"` // Hardfail on Certificate Revokation
	DenyInsecureTLS                 SecurityLevelBoolean `json:",omitempty bson:",omitempty"` // Block TLS connections, that use insecure TLS versions, cipher suites, ...
	DenyTLSWithoutSNI               SecurityLevelBoolean `json:",omitempty bson:",omitempty"` // Block TLS connections that do not use SNI, connections without SNI cannot be verified as well as connections with SNI.
	DoNotUseAssignedDNS             SecurityLevelBoolean `json:",omitempty bson:",omitempty"` // Do not use DNS Servers assigned by DHCP
	DoNotUseMDNS                    SecurityLevelBoolean `json:",omitempty bson:",omitempty"` // Do not use mDNS
	DoNotForwardSpecialDomains      SecurityLevelBoolean `json:",omitempty bson:",omitempty"` // Do not resolve special domains with assigned DNS Servers
	AlwaysPromptAtNewProfile        SecurityLevelBoolean `json:",omitempty bson:",omitempty"` // Always prompt user to review new profiles
	DenyNetworkUntilProfileApproved SecurityLevelBoolean `json:",omitempty bson:",omitempty"` // Deny network communication until a new profile is actively approved by the user

	// Generic Config
	CompetenceLevel   int8     `json:",omitempty bson:",omitempty"` // Select CompetenceLevel
	Beta              bool     `json:",omitempty bson:",omitempty"` // Take part in Beta
	PermanentVerdicts bool     `json:",omitempty bson:",omitempty"` // As soon as work on a link is finished, leave it to the system for performance and stability
	DNSServers        []string `json:",omitempty bson:",omitempty"` // DNS Servers to use for name resolution. Please refer to the user guide for further help.
	// regex: ^(DoH|DNS|TDNS)\|[A-Za-z0-9\.:\[\]]+(\|[A-Za-z0-9\.:]+)?$
	DNSServerRetryRate int64    `json:",omitempty bson:",omitempty"` // Amount of seconds to wait until failing DNS Servers may be retried.
	CountryBlacklist   []string `json:",omitempty bson:",omitempty"` // Do not connect to servers in these countries
	ASBlacklist        []uint32 `json:",omitempty bson:",omitempty"` // Do not connect to server in these AS

	LocalPort17Node  bool `json:",omitempty bson:",omitempty"` // Serve as local Port17 Node
	PublicPort17Node bool `json:",omitempty bson:",omitempty"` // Serve as public Port17 Node
}

var (
	configurationModel               *Configuration // only use this as parameter for database.EnsureModel-like functions
	configurationInstanceName        = "config"
	defaultConfigurationInstanceName = "default"
)

func initConfigurationModel() {
	database.RegisterModel(configurationModel, func() database.Model { return new(Configuration) })
}

// Create saves Configuration with the provided name in the default namespace.
func (m *Configuration) Create(name string) error {
	return m.CreateObject(&database.Me, name, m)
}

// CreateInNamespace saves Configuration with the provided name in the provided namespace.
func (m *Configuration) CreateInNamespace(namespace *datastore.Key, name string) error {
	return m.CreateObject(namespace, name, m)
}

// Save saves Configuration.
func (m *Configuration) Save() error {
	return m.SaveObject(m)
}

// GetConfiguration fetches Configuration with the provided name in the default namespace.
func GetConfiguration(name string) (*Configuration, error) {
	return GetConfigurationFromNamespace(&database.Me, name)
}

// GetConfigurationFromNamespace fetches Configuration with the provided name in the provided namespace.
func GetConfigurationFromNamespace(namespace *datastore.Key, name string) (*Configuration, error) {
	object, err := database.GetAndEnsureModel(namespace, name, configurationModel)
	if err != nil {
		return nil, err
	}
	model, ok := object.(*Configuration)
	if !ok {
		return nil, database.NewMismatchError(object, configurationModel)
	}
	return model, nil
}
