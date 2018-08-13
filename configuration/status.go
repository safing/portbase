// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package configuration

import (
	"github.com/Safing/safing-core/database"

	datastore "github.com/ipfs/go-datastore"
)

// SystemStatus saves basic information about the current system status.
type SystemStatus struct {
	database.Base
	CurrentSecurityLevel  int8
	SelectedSecurityLevel int8

	ThreatLevel  int8   `json:",omitempty" bson:",omitempty"`
	ThreatReason string `json:",omitempty" bson:",omitempty"`

	PortmasterStatus    int8   `json:",omitempty" bson:",omitempty"`
	PortmasterStatusMsg string `json:",omitempty" bson:",omitempty"`

	Port17Status    int8   `json:",omitempty" bson:",omitempty"`
	Port17StatusMsg string `json:",omitempty" bson:",omitempty"`
}

var (
	systemStatusModel        *SystemStatus // only use this as parameter for database.EnsureModel-like functions
	systemStatusInstanceName = "status"
)

func initSystemStatusModel() {
	database.RegisterModel(systemStatusModel, func() database.Model { return new(SystemStatus) })
}

// Create saves SystemStatus with the provided name in the default namespace.
func (m *SystemStatus) Create() error {
	return m.CreateObject(&database.Me, systemStatusInstanceName, m)
}

// CreateInNamespace saves SystemStatus with the provided name in the provided namespace.
func (m *SystemStatus) CreateInNamespace(namespace *datastore.Key) error {
	return m.CreateObject(namespace, systemStatusInstanceName, m)
}

// Save saves SystemStatus.
func (m *SystemStatus) Save() error {
	return m.SaveObject(m)
}

// FmtSecurityLevel returns the current security level as a string.
func (m *SystemStatus) FmtSecurityLevel() string {
	var s string
	switch m.CurrentSecurityLevel {
	case SecurityLevelOff:
		s = "Off"
	case SecurityLevelDynamic:
		s = "Dynamic"
	case SecurityLevelSecure:
		s = "Secure"
	case SecurityLevelFortress:
		s = "Fortress"
	}
	if m.CurrentSecurityLevel != m.SelectedSecurityLevel {
		s += "*"
	}
	return s
}

// GetSystemStatus fetches SystemStatus with the provided name in the default namespace.
func GetSystemStatus() (*SystemStatus, error) {
	return GetSystemStatusFromNamespace(&database.Me)
}

// GetSystemStatusFromNamespace fetches SystemStatus with the provided name in the provided namespace.
func GetSystemStatusFromNamespace(namespace *datastore.Key) (*SystemStatus, error) {
	object, err := database.GetAndEnsureModel(namespace, systemStatusInstanceName, systemStatusModel)
	if err != nil {
		return nil, err
	}
	model, ok := object.(*SystemStatus)
	if !ok {
		return nil, database.NewMismatchError(object, systemStatusModel)
	}
	return model, nil
}
