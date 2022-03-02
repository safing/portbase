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
	"sync"

	"github.com/safing/portbase/log"
	"github.com/safing/portbase/utils"
)

// UpdateIndexes downloads all indexes. An error is only returned when all
// indexes fail to update.
func (reg *ResourceRegistry) UpdateIndexes(ctx context.Context) error {
	var lastErr error
	var anySuccess bool

	client := &http.Client{}
	for _, idx := range reg.getIndexes() {
		if err := reg.downloadIndex(ctx, client, idx); err != nil {
			lastErr = err
			log.Warningf("%s: failed to update index %s: %s", reg.Name, idx.Path, err)
		} else {
			anySuccess = true
		}
	}

	if !anySuccess {
		return fmt.Errorf("failed to update all indexes, last error was: %w", lastErr)
	}
	return nil
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

	// Add index data to registry.
	if len(newIndexData) > 0 {
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
		err = reg.AddResources(cleanedData, false, true, idx.PreRelease)
		if err != nil {
			log.Warningf("%s: failed to add resources: %s", reg.Name, err)
		}
	} else {
		log.Debugf("%s: index %s is empty", reg.Name, idx.Path)
	}

	// check if dest dir exists
	indexDir := filepath.FromSlash(path.Dir(idx.Path))
	err = reg.storageDir.EnsureRelPath(indexDir)
	if err != nil {
		log.Warningf("%s: failed to ensure directory for updated index %s: %s", reg.Name, idx.Path, err)
	}

	// save index
	indexPath := filepath.FromSlash(idx.Path)
	// Index files must be readable by portmaster-staert with user permissions in order to load the index.
	err = ioutil.WriteFile(filepath.Join(reg.storageDir.Path, indexPath), data, 0o0644) //nolint:gosec
	if err != nil {
		log.Warningf("%s: failed to save updated index %s: %s", reg.Name, idx.Path, err)
	}

	log.Infof("%s: updated index %s with %d entries", reg.Name, idx.Path, len(newIndexData))
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
				if !rv.Available && rv.CurrentRelease {
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
	if err := reg.tmpDir.Ensure(); err != nil {
		return fmt.Errorf("could not prepare tmp directory for download: %w", err)
	}

	// download updates
	log.Infof("%s: starting to download %d updates parallel", reg.Name, len(toUpdate))
	var wg sync.WaitGroup

	wg.Add(len(toUpdate))
	client := &http.Client{}

	for idx := range toUpdate {
		go func(rv *ResourceVersion) {
			var err error

			defer wg.Done()
			defer func() {
				if x := recover(); x != nil {
					log.Errorf("%s: captured panic: %s", rv.resource.Identifier, x)
				}
			}()

			for tries := 0; tries < 3; tries++ {
				err = reg.fetchFile(ctx, client, rv, tries)
				if err == nil {
					rv.Available = true
					return
				}
			}
			if err != nil {
				log.Warningf("%s: failed to download %s version %s: %s", reg.Name, rv.resource.Identifier, rv.VersionNumber, err)
			}
		}(toUpdate[idx])
	}

	wg.Wait()

	log.Infof("%s: finished downloading updates", reg.Name)

	return nil
}
