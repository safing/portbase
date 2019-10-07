package updater

import (
	"testing"
)

func TestVersionSelection(t *testing.T) {
	res := registry.newResource("test/a")

	err := res.AddVersion("1.2.3", true, true, false)
	if err != nil {
		t.Fatal(err)
	}
	err = res.AddVersion("1.2.4b", true, false, true)
	if err != nil {
		t.Fatal(err)
	}
	err = res.AddVersion("1.2.2", true, false, false)
	if err != nil {
		t.Fatal(err)
	}
	err = res.AddVersion("1.2.5", false, true, false)
	if err != nil {
		t.Fatal(err)
	}
	err = res.AddVersion("0", true, false, false)
	if err != nil {
		t.Fatal(err)
	}

	registry.Online = true
	registry.Beta = true
	registry.DevMode = true
	res.selectVersion()
	if res.SelectedVersion.VersionNumber != "0" {
		t.Errorf("selected version should be 0, not %s", res.SelectedVersion.VersionNumber)
	}

	registry.DevMode = false
	res.selectVersion()
	if res.SelectedVersion.VersionNumber != "1.2.4b" {
		t.Errorf("selected version should be 1.2.4b, not %s", res.SelectedVersion.VersionNumber)
	}

	registry.Beta = false
	res.selectVersion()
	if res.SelectedVersion.VersionNumber != "1.2.5" {
		t.Errorf("selected version should be 1.2.5, not %s", res.SelectedVersion.VersionNumber)
	}

	registry.Online = false
	res.selectVersion()
	if res.SelectedVersion.VersionNumber != "1.2.3" {
		t.Errorf("selected version should be 1.2.3, not %s", res.SelectedVersion.VersionNumber)
	}

	f123 := res.GetFile()
	f123.markActiveWithLocking()

	err = res.Blacklist("1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if res.SelectedVersion.VersionNumber != "1.2.2" {
		t.Errorf("selected version should be 1.2.2, not %s", res.SelectedVersion.VersionNumber)
	}

	if !f123.UpgradeAvailable() {
		t.Error("upgrade should be available (flag)")
	}
	select {
	case <-f123.WaitForAvailableUpgrade():
	default:
		t.Error("upgrade should be available (chan)")
	}

	t.Logf("resource: %+v", res)
	for _, rv := range res.Versions {
		t.Logf("version %s: %+v", rv.VersionNumber, rv)
	}
}
