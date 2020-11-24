package osdetail

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateBinaryNameFromPath(t *testing.T) {
	assert.Equal(t, "Nslookup", GenerateBinaryNameFromPath("nslookup.exe"))
	assert.Equal(t, "System Settings", GenerateBinaryNameFromPath("SystemSettings.exe"))
	assert.Equal(t, "One Drive Setup", GenerateBinaryNameFromPath("OneDriveSetup.exe"))
	assert.Equal(t, "Msedge", GenerateBinaryNameFromPath("msedge.exe"))
	assert.Equal(t, "SIH Client", GenerateBinaryNameFromPath("SIHClient.exe"))
	assert.Equal(t, "Openvpn Gui", GenerateBinaryNameFromPath("openvpn-gui.exe"))
	assert.Equal(t, "Portmaster Core v0-1-2", GenerateBinaryNameFromPath("portmaster-core_v0-1-2.exe"))
	assert.Equal(t, "Win Store App", GenerateBinaryNameFromPath("WinStore.App.exe"))
	assert.Equal(t, "Test Script", GenerateBinaryNameFromPath(".test-script"))
	assert.Equal(t, "Browser Broker", GenerateBinaryNameFromPath("browser_broker.exe"))
}

func TestCleanFileDescription(t *testing.T) {
	assert.Equal(t, "Product Name", cleanFileDescription(strings.Fields("Product Name. Does this and that.")))
	assert.Equal(t, "Product Name", cleanFileDescription(strings.Fields("Product Name - Does this and that.")))
	assert.Equal(t, "Product Name", cleanFileDescription(strings.Fields("Product Name / Does this and that.")))
	assert.Equal(t, "Product Name", cleanFileDescription(strings.Fields("Product Name :: Does this and that.")))
	assert.Equal(t, "/ Product Name", cleanFileDescription(strings.Fields("/ Product Name")))
	assert.Equal(t, "Product", cleanFileDescription(strings.Fields("Product / Name")))

	assert.Equal(t,
		"Product Name a Does this and that.",
		cleanFileDescription(strings.Fields("Product Name a Does this and that.")),
	)
}
