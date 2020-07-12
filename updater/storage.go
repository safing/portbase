package updater

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/safing/portbase/log"
	"github.com/safing/portbase/utils"
)

// Common errors.
var (
	ErrEmptyIndex                 = errors.New("index file is empty")
	ErrSelectedVersionUnavailable = errors.New("selected version is not available")
)

// ScanStorage scans root within the storage dir and adds found
// resources to the registry. If an error occurred, it is logged
// and the last error is returned. Everything that was found
// despite errors is added to the registry anyway. Leave root
// empty to scan the full storage dir.
func (reg *ResourceRegistry) ScanStorage(root string) error {
	var lastError error

	// prep root
	if root == "" {
		root = reg.storageDir.Path
	} else {
		var err error
		root, err = filepath.Abs(root)
		if err != nil {
			return err
		}
		if !strings.HasPrefix(root, reg.storageDir.Path) {
			return fmt.Errorf("scan path: %w", utils.ErrOutOfDirScope)
		}
	}

	// walk fs
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			lastError = fmt.Errorf("%s: could not read %s: %w", reg.Name, path, err)
			log.Warning(lastError.Error())
			return nil
		}

		// get relative path to storage
		relativePath, err := filepath.Rel(reg.storageDir.Path, path)
		if err != nil {
			lastError = fmt.Errorf("%s: could not get relative path of %s: %w", reg.Name, path, err)
			log.Warning(lastError.Error())
			return nil
		}
		// ignore files in tmp dir
		if strings.HasPrefix(relativePath, reg.tmpDir.Path) {
			return nil
		}

		// convert to identifier and version
		relativePath = filepath.ToSlash(relativePath)
		identifier, version, ok := GetIdentifierAndVersion(relativePath)
		if !ok {
			// file does not conform to format
			return nil
		}

		// save
		err = reg.AddResource(identifier, version, true, false, false)
		if err != nil {
			lastError = fmt.Errorf("%s: could not get add resource %s v%s: %w", reg.Name, identifier, version, err)
			log.Warning(lastError.Error())
		}
		return nil
	})

	return lastError
}

// LoadIndexes loads the current release indexes from disk
// or will fetch a new version if not available and the
// registry is marked as online.
func (reg *ResourceRegistry) LoadIndexes(ctx context.Context) error {
	var firstErr error
	client := &http.Client{}
	for _, idx := range reg.getIndexes() {
		err := reg.loadIndexFile(idx)
		if err == nil {
			log.Debugf("%s: loaded index %s", reg.Name, idx.Path)
		} else if reg.Online {
			// try to download the index file if a local disk version
			// does not exist or we don't have permission to read it.
			if os.IsNotExist(err) || os.IsPermission(err) {
				err = reg.downloadIndex(ctx, client, idx)
			}
		}

		if err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}

func (reg *ResourceRegistry) getIndexes() []Index {
	reg.RLock()
	defer reg.RUnlock()
	indexes := make([]Index, len(reg.indexes))
	copy(indexes, reg.indexes)
	return indexes
}

func (reg *ResourceRegistry) loadIndexFile(idx Index) error {
	path := filepath.FromSlash(idx.Path)
	data, err := ioutil.ReadFile(filepath.Join(reg.storageDir.Path, path))
	if err != nil {
		return err
	}

	releases := make(map[string]string)
	err = json.Unmarshal(data, &releases)
	if err != nil {
		return err
	}

	if len(releases) == 0 {
		return fmt.Errorf("%s: %w", path, ErrEmptyIndex)
	}

	err = reg.AddResources(releases, false, idx.Stable, idx.Beta)
	if err != nil {
		log.Warningf("%s: failed to add resource: %s", reg.Name, err)
	}
	return nil
}

// CreateSymlinks creates a directory structure with unversioned symlinks to the given updates list.
func (reg *ResourceRegistry) CreateSymlinks(symlinkRoot *utils.DirStructure) error {
	err := os.RemoveAll(symlinkRoot.Path)
	if err != nil {
		return fmt.Errorf("failed to wipe symlink root: %w", err)
	}

	err = symlinkRoot.Ensure()
	if err != nil {
		return fmt.Errorf("failed to create symlink root: %w", err)
	}

	reg.RLock()
	defer reg.RUnlock()

	for _, res := range reg.resources {
		if res.SelectedVersion == nil {
			return fmt.Errorf("%s: %w", res.Identifier, ErrSelectedVersionUnavailable)
		}

		targetPath := res.SelectedVersion.storagePath()
		linkPath := filepath.Join(symlinkRoot.Path, filepath.FromSlash(res.Identifier))
		linkPathDir := filepath.Dir(linkPath)

		err = symlinkRoot.EnsureAbsPath(linkPathDir)
		if err != nil {
			return fmt.Errorf("failed to create dir for link: %w", err)
		}

		relativeTargetPath, err := filepath.Rel(linkPathDir, targetPath)
		if err != nil {
			return fmt.Errorf("failed to get relative target path: %w", err)
		}

		err = os.Symlink(relativeTargetPath, linkPath)
		if err != nil {
			return fmt.Errorf("failed to link %s: %w", res.Identifier, err)
		}
	}

	return nil
}
