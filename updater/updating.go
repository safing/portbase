package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/safing/portbase/utils"

	"github.com/safing/portbase/log"
)

// UpdateIndexes downloads all indexes and returns the first error encountered.
func (reg *ResourceRegistry) UpdateIndexes() error {
	var firstErr error

	for _, idx := range reg.getIndexes() {
		if err := reg.downloadIndex(idx); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

func (reg *ResourceRegistry) downloadIndex(idx Index) error {
	var err error
	var data []byte

	// download new index
	for tries := 0; tries < 3; tries++ {
		data, err = reg.fetchData(idx.Path, tries)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("failed to download index %s: %w", idx.Path, err)
	}

	// parse
	new := make(map[string]string)
	err = json.Unmarshal(data, &new)
	if err != nil {
		return fmt.Errorf("failed to parse index %s: %w", idx.Path, err)
	}

	// check for content
	if len(new) == 0 {
		return fmt.Errorf("index %s is empty", idx.Path)
	}

	// add resources to registry
	err = reg.AddResources(new, false, idx.Stable, idx.Beta)
	if err != nil {
		log.Warningf("%s: failed to add resources: %s", reg.Name, err)
	}

	// check if dest dir exists
	err = reg.storageDir.EnsureRelPath(filepath.Dir(idx.Path))
	if err != nil {
		log.Warningf("%s: failed to ensure directory for updated index %s: %s", reg.Name, idx.Path, err)
	}

	// save index
	err = ioutil.WriteFile(filepath.Join(reg.storageDir.Path, idx.Path), data, 0644)
	if err != nil {
		log.Warningf("%s: failed to save updated index %s: %s", reg.Name, idx.Path, err)
	}

	log.Infof("%s: updated index %s", reg.Name, idx.Path)
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
		if res.inUse() ||
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
		return fmt.Errorf("could not prepare tmp directory for download: %w", err)
	}

	// download updates
	log.Infof("%s: starting to download %d updates", reg.Name, len(toUpdate))
	for _, rv := range toUpdate {
		for tries := 0; tries < 3; tries++ {
			err = reg.fetchFile(rv, tries)
			if err == nil {
				rv.Available = true
				break
			}
		}
		if err != nil {
			log.Warningf("%s: failed to download %s version %s: %s", reg.Name, rv.resource.Identifier, rv.VersionNumber, err)
		}
	}
	log.Infof("%s: finished downloading updates", reg.Name)

	return nil
}
