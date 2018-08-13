// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package configuration

import (
	"github.com/Safing/safing-core/log"
)

var (
	defaultConfig Configuration
)

func initDefaultConfig() {
	defaultConfig = Configuration{
		// based on security level
		EnforceCT:                       3,
		EnforceRevocation:               3,
		DenyInsecureTLS:                 2,
		DenyTLSWithoutSNI:               2,
		DoNotUseAssignedDNS:             3,
		DoNotUseMDNS:                    2,
		DoNotForwardSpecialDomains:      2,
		AlwaysPromptAtNewProfile:        3,
		DenyNetworkUntilProfileApproved: 3,

		// generic configuration
		CompetenceLevel:   0,
		PermanentVerdicts: true,
		// Possible values: DNS, DoH (DNS over HTTPS - using Google's syntax: https://developers.google.com/speed/public-dns/docs/dns-over-https)
		// DNSServers: []string{"DoH|dns.google.com:443|df:www.google.com"},
		DNSServers: []string{"DNS|1.1.1.1:53", "DNS|1.0.0.1:53", "DNS|[2606:4700:4700::1111]:53", "DNS|[2606:4700:4700::1001]:53", "DNS|8.8.8.8:53", "DNS|8.8.4.4:53", "DNS|[2001:4860:4860::8888]:53", "DNS|[2001:4860:4860::8844]:53", "DNS|208.67.222.222:53", "DNS|208.67.220.220:53"},
		// DNSServers: []string{"DNS|[2001:4860:4860::8888]:53", "DNS|[2001:4860:4860::8844]:53"},
		// DNSServers: []string{"DoH|dns.google.com:443|df:www.google.com", "DNS|8.8.8.8:53", "DNS|8.8.4.4:53", "DNS|172.30.30.1:53", "DNS|172.20.30.2:53"},
		// DNSServers: []string{"DNS|208.67.222.222:53", "DNS|208.67.220.220:53", "DNS|8.8.8.8:53", "DNS|8.8.4.4:53"},
		// Amount of seconds to wait until failing DNS Servers may be retried.
		DNSServerRetryRate: 120,
		// CountryBlacklist    []string
		// ASBlacklist         []uint32
		LocalPort17Node:  false,
		PublicPort17Node: true,
	}
	err := defaultConfig.Create(defaultConfigurationInstanceName)
	if err != nil {
		log.Warningf("configuration: could not save default configuration: %s", err)
	}
}
