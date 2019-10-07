package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/safing/portbase/utils"

	"github.com/safing/portbase/log"
)

// UpdateIndexes downloads the current update indexes.
func (reg *ResourceRegistry) UpdateIndexes() error {
	err := reg.downloadIndex("stable.json", true, false)
	if err != nil {
		return err
	}

	return reg.downloadIndex("beta.json", false, true)
}

func (reg *ResourceRegistry) downloadIndex(name string, stableRelease, betaRelease bool) error {
	var err error
	var data []byte

	// download new index
	for tries := 0; tries < 3; tries++ {
		data, err = reg.fetchData(name, tries)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("failed to download index %s: %s", name, err)
	}

	// parse
	new := make(map[string]string)
	err = json.Unmarshal(data, &new)
	if err != nil {
		return fmt.Errorf("failed to parse index %s: %s", name, err)
	}

	// check for content
	if len(new) == 0 {
		return fmt.Errorf("index %s is empty", name)
	}

	// add resources to registry
	_ = reg.AddResources(new, false, stableRelease, betaRelease)

	// save index
	err = ioutil.WriteFile(filepath.Join(reg.storageDir.Path, name), data, 0644)
	if err != nil {
		log.Warningf("%s: failed to save updated index %s: %s", reg.Name, name, err)
	}

	log.Infof("%s: updated index %s", reg.Name, name)
	return nil
}

// DownloadUpdates checks if updates are available and downloads updates of used components.
func (reg *ResourceRegistry) DownloadUpdates(ctx context.Context) error {

	// create list of downloads
	var toUpdate []*ResourceVersion
	reg.RLock()
	for _, res := range reg.resources {
		res.Lock()

		// check if we want to download
		if res.ActiveVersion != nil || // resource is currently being used
			res.available() || // resource was used in the past
			utils.StringInSlice(reg.MandatoryUpdates, res.Identifier) { // resource is mandatory

			// add all non-available and eligible versions to update queue
			for _, rv := range res.Versions {
				if !rv.Available && (rv.StableRelease || reg.Beta && rv.BetaRelease) {
					toUpdate = append(toUpdate, rv)
				}
			}

		}

		res.Unlock()
	}
	reg.RUnlock()

	// nothing to update
	if len(toUpdate) == 0 {
		log.Infof("%s: everything up to date", reg.Name)
		return nil
	}

	// check download dir
	err := reg.tmpDir.Ensure()
	if err != nil {
		return fmt.Errorf("could not prepare tmp directory for download: %s", err)
	}

	// download updates
	log.Infof("%s: starting to download %d updates", reg.Name, len(toUpdate))
	for _, rv := range toUpdate {
		for tries := 0; tries < 3; tries++ {
			err = reg.fetchFile(rv, tries)
			if err == nil {
				break
			}
		}
		if err != nil {
			log.Warningf("%s: failed to download %s version %s: %s", reg.Name, rv.resource.Identifier, rv.VersionNumber, err)
		}
	}
	log.Infof("%s: finished downloading updates", reg.Name)

	// remove tmp folder after we are finished
	err = os.RemoveAll(reg.tmpDir.Path)
	if err != nil {
		log.Tracef("%s: failed to remove tmp dir %s after downloading updates: %s", reg.Name, reg.tmpDir.Path, err)
	}

	return nil
}
