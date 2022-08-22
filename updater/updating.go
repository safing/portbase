package updater

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/safing/jess/filesig"
	"github.com/safing/jess/lhash"
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

func (reg *ResourceRegistry) downloadIndex(ctx context.Context, client *http.Client, idx *Index) error {
	var (
		// Index.
		indexErr    error
		indexData   []byte
		downloadURL string

		// Signature.
		sigErr       error
		verifiedHash *lhash.LabeledHash
		sigFileData  []byte
		verifOpts    = reg.GetVerificationOptions(idx.Path)
	)

	// Upgrade to v2 index if verification is enabled.
	downloadIndexPath := idx.Path
	if verifOpts != nil {
		downloadIndexPath = strings.TrimSuffix(downloadIndexPath, baseIndexExtension) + v2IndexExtension
	}

	// Download new index and signature.
	for tries := 0; tries < 3; tries++ {
		// Index and signature need to be fetched together, so that they are
		// fetched from the same source. One source should always have a matching
		// index and signature. Backup sources may be behind a little.
		// If the signature verification fails, another source should be tried.

		// Get index data.
		indexData, downloadURL, indexErr = reg.fetchData(ctx, client, downloadIndexPath, tries)
		if indexErr != nil {
			log.Debugf("%s: failed to fetch index %s: %s", reg.Name, downloadURL, indexErr)
			continue
		}

		// Get signature and verify it.
		if verifOpts != nil {
			verifiedHash, sigFileData, sigErr = reg.fetchAndVerifySigFile(
				ctx, client,
				verifOpts, downloadIndexPath+filesig.Extension, nil,
				tries,
			)
			if sigErr != nil {
				log.Debugf("%s: failed to verify signature of %s: %s", reg.Name, downloadURL, sigErr)
				continue
			}

			// Check if the index matches the verified hash.
			if verifiedHash.MatchesData(indexData) {
				log.Infof("%s: verified signature of %s", reg.Name, downloadURL)
			} else {
				sigErr = ErrIndexChecksumMismatch
				log.Debugf("%s: checksum does not match file from %s", reg.Name, downloadURL)
				continue
			}
		}

		break
	}
	if indexErr != nil {
		return fmt.Errorf("failed to fetch index %s: %w", downloadIndexPath, indexErr)
	}
	if sigErr != nil {
		return fmt.Errorf("failed to fetch or verify index %s signature: %w", downloadIndexPath, sigErr)
	}

	// Parse the index file.
	indexFile, err := ParseIndexFile(indexData, idx.Channel, idx.LastRelease)
	if err != nil {
		return fmt.Errorf("failed to parse index %s: %w", idx.Path, err)
	}

	// Add index data to registry.
	if len(indexFile.Releases) > 0 {
		// Check if all resources are within the indexes' authority.
		authoritativePath := path.Dir(idx.Path) + "/"
		if authoritativePath == "./" {
			// Fix path for indexes at the storage root.
			authoritativePath = ""
		}
		cleanedData := make(map[string]string, len(indexFile.Releases))
		for key, version := range indexFile.Releases {
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

	// Check if dest dir exists.
	indexDir := filepath.FromSlash(path.Dir(idx.Path))
	err = reg.storageDir.EnsureRelPath(indexDir)
	if err != nil {
		log.Warningf("%s: failed to ensure directory for updated index %s: %s", reg.Name, idx.Path, err)
	}

	// Index files must be readable by portmaster-staert with user permissions in order to load the index.
	err = ioutil.WriteFile( //nolint:gosec
		filepath.Join(reg.storageDir.Path, filepath.FromSlash(idx.Path)),
		indexData, 0o0644,
	)
	if err != nil {
		log.Warningf("%s: failed to save updated index %s: %s", reg.Name, idx.Path, err)
	}

	// Write signature file, if we have one.
	if len(sigFileData) > 0 {
		err = ioutil.WriteFile( //nolint:gosec
			filepath.Join(reg.storageDir.Path, filepath.FromSlash(idx.Path)+filesig.Extension),
			sigFileData, 0o0644,
		)
		if err != nil {
			log.Warningf("%s: failed to save updated index signature %s: %s", reg.Name, idx.Path+filesig.Extension, err)
		}
	}

	log.Infof("%s: updated index %s with %d entries", reg.Name, idx.Path, len(indexFile.Releases))
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
					log.Errorf("%s: %s: captured panic: %s", reg.Name, rv.resource.Identifier, x)
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
