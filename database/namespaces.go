// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import datastore "github.com/ipfs/go-datastore"

var (
	// Persistent data that is fetched or gathered, entries may be deleted
	Cache                 = datastore.NewKey("/Cache")
	DNSCache              = Cache.ChildString("Dns")
	IntelCache            = Cache.ChildString("Intel")
	FileInfoCache         = Cache.ChildString("FileInfo")
	ProfileCache          = Cache.ChildString("Profile")
	IPInfoCache           = Cache.ChildString("IPInfo")
	CertCache             = Cache.ChildString("Cert")
	CARevocationInfoCache = Cache.ChildString("CARevocationInfo")

	// Volatile, in-memory (recommended) namespace for storing runtime information, cleans itself
	Run                = datastore.NewKey("/Run")
	Processes          = Run.ChildString("Processes")
	OrphanedConnection = Run.ChildString("OrphanedConnections")
	OrphanedLink       = Run.ChildString("OrphanedLinks")
	Api                = Run.ChildString("Api")
	ApiSessions        = Api.ChildString("ApiSessions")

	// Namespace for current device, will be mounted into /Devices/[device]
	Me = datastore.NewKey("/Me")

	// Holds data of all Devices
	Devices = datastore.NewKey("/Devices")

	// Holds persistent data
	Data     = datastore.NewKey("/Data")
	Profiles = Data.ChildString("Profiles")

	// Holds data distributed by the System (coming from the Community and Devs)
	Dist         = datastore.NewKey("/Dist")
	DistProfiles = Dist.ChildString("Profiles")
	DistUpdates  = Dist.ChildString("Updates")

	// Holds data issued by company
	Company         = datastore.NewKey("/Company")
	CompanyProfiles = Company.ChildString("Profiles")
	CompanyUpdates  = Company.ChildString("Updates")

	// Server
	// The Authority namespace is used by authoritative servers (Safing or Company) to store data (Intel, Profiles, ...) to be served to clients
	Authority            = datastore.NewKey("/Authority")
	AthoritativeIntel    = Authority.ChildString("Intel")
	AthoritativeProfiles = Authority.ChildString("Profiles")
	// The Staging namespace is the same as the Authority namespace, but for rolling out new things first to a selected list of clients for testing
	AuthorityStaging            = datastore.NewKey("/Staging")
	AthoritativeStagingProfiles = AuthorityStaging.ChildString("Profiles")

	// Holds data of Apps
	Apps = datastore.NewKey("/Apps")

	// Test & Invalid namespace
	Tests   = datastore.NewKey("/Tests")
	Invalid = datastore.NewKey("/Invalid")
)
