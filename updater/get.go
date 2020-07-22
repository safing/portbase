package updater

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/safing/portbase/log"
)

// Errors
var (
	ErrNotFound            = errors.New("the requested file could not be found")
	ErrNotAvailableLocally = errors.New("the requested file is not available locally")
)

// GetFile returns the selected (mostly newest) file with the given
// identifier or an error, if it fails.
func (reg *ResourceRegistry) GetFile(identifier string) (*File, error) {
	reg.RLock()
	res, ok := reg.resources[identifier]
	reg.RUnlock()
	if !ok {
		return nil, ErrNotFound
	}

	file := res.GetFile()
	// check if file is available locally
	if file.version.Available {
		file.markActiveWithLocking()
		return file, nil
	}

	// check if online
	if !reg.Online {
		return nil, ErrNotAvailableLocally
	}

	// check download dir
	err := reg.tmpDir.Ensure()
	if err != nil {
		return nil, fmt.Errorf("could not prepare tmp directory for download: %w", err)
	}

	// download file
	log.Tracef("%s: starting download of %s", reg.Name, file.versionedPath)
	client := &http.Client{}
	for tries := 0; tries < 5; tries++ {
		err = reg.fetchFile(context.TODO(), client, file.version, tries)
		if err != nil {
			log.Tracef("%s: failed to download %s: %s, retrying (%d)", reg.Name, file.versionedPath, err, tries+1)
		} else {
			file.markActiveWithLocking()
			return file, nil
		}
	}
	log.Warningf("%s: failed to download %s: %s", reg.Name, file.versionedPath, err)
	return nil, err
}
