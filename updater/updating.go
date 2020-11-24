package updater

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/safing/portbase/utils"

	"github.com/safing/portbase/log"
)

// UpdateIndexes downloads all indexes and returns the first error encountered.
func (reg *ResourceRegistry) UpdateIndexes(ctx context.Context) error {
	var firstErr error

	client := &http.Client{}
	for _, idx := range reg.getIndexes() {
		if err := reg.downloadIndex(ctx, client, idx); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	return firstErr
}

func (reg *ResourceRegistry) downloadIndex(ctx context.Context, client *http.Client, idx Index) error {
	var err error
	var data []byte

	// download new index
	for tries := 0; tries < 3; tries++ {
		data, err = reg.fetchData(ctx, client, idx.Path, tries)
		if err == nil {
			break
		}
	}
	if err != nil {
		return fmt.Errorf("failed to download index %s: %w", idx.Path, err)
	}

	// parse
	newIndexData := make(map[string]string)
	err = json.Unmarshal(data, &newIndexData)
	if err != nil {
		return fmt.Errorf("failed to parse index %s: %w", idx.Path, err)
	}

	// check for content
	if len(newIndexData) == 0 {
		return fmt.Errorf("index %s is empty", idx.Path)
	}

	// Check if all resources are within the indexes' authority.
	authoritativePath := path.Dir(idx.Path) + "/"
	if authoritativePath == "./" {
		// Fix path for indexes at the storage root.
		authoritativePath = ""
	}
	cleanedData := make(map[string]string, len(newIndexData))
	for key, version := range newIndexData {
		if strings.HasPrefix(key, authoritativePath) {
			cleanedData[key] = version
		} else {
			log.Warningf("%s: index %s oversteps it's authority by defining version for %s", reg.Name, idx.Path, key)
		}
	}

	// add resources to registry
	err = reg.AddResources(cleanedData, false, idx.Stable, idx.Beta)
	if err != nil {
		log.Warningf("%s: failed to add resources: %s", reg.Name, err)
	}

	// check if dest dir exists
	indexDir := filepath.FromSlash(path.Dir(idx.Path))
	err = reg.storageDir.EnsureRelPath(indexDir)
	if err != nil {
		log.Warningf("%s: failed to ensure directory for updated index %s: %s", reg.Name, idx.Path, err)
	}

	// save index
	indexPath := filepath.FromSlash(idx.Path)
	err = ioutil.WriteFile(filepath.Join(reg.storageDir.Path, indexPath), data, 0644)
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
	client := &http.Client{}
	for _, rv := range toUpdate {
		for tries := 0; tries < 3; tries++ {
			err = reg.fetchFile(ctx, client, rv, tries)
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
