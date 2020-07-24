package updater

import (
	"os"
	"runtime"
	"sync"

	"github.com/safing/portbase/log"
	"github.com/safing/portbase/utils"
)

const (
	onWindows = runtime.GOOS == "windows"
)

// ResourceRegistry is a registry for managing update resources.
type ResourceRegistry struct {
	sync.RWMutex

	Name       string
	storageDir *utils.DirStructure
	tmpDir     *utils.DirStructure
	indexes    []Index

	resources        map[string]*Resource
	UpdateURLs       []string
	UserAgent        string
	MandatoryUpdates []string

	Beta    bool
	DevMode bool
	Online  bool
}

// AddIndex adds a new index to the resource registry.
func (reg *ResourceRegistry) AddIndex(idx Index) {
	reg.Lock()
	defer reg.Unlock()

	reg.indexes = append(reg.indexes, idx)
}

// Initialize initializes a raw registry struct and makes it ready for usage.
func (reg *ResourceRegistry) Initialize(storageDir *utils.DirStructure) error {
	// check if storage dir is available
	err := storageDir.Ensure()
	if err != nil {
		return err
	}

	// set default name
	if reg.Name == "" {
		reg.Name = "updater"
	}

	// initialize private attributes
	reg.storageDir = storageDir
	reg.tmpDir = storageDir.ChildDir("tmp", 0700)
	reg.resources = make(map[string]*Resource)

	// remove tmp dir to delete old entries
	err = reg.Cleanup()
	if err != nil {
		log.Warningf("%s: failed to remove tmp dir: %s", reg.Name, err)
	}

	// (re-)create tmp dir
	err = reg.tmpDir.Ensure()
	if err != nil {
		log.Warningf("%s: failed to create tmp dir: %s", reg.Name, err)
	}

	return nil
}

// StorageDir returns the main storage dir of the resource registry.
func (reg *ResourceRegistry) StorageDir() *utils.DirStructure {
	return reg.storageDir
}

// TmpDir returns the temporary working dir of the resource registry.
func (reg *ResourceRegistry) TmpDir() *utils.DirStructure {
	return reg.tmpDir
}

// SetDevMode sets the development mode flag.
func (reg *ResourceRegistry) SetDevMode(on bool) {
	reg.Lock()
	defer reg.Unlock()

	reg.DevMode = on
}

// SetBeta sets the beta flag.
func (reg *ResourceRegistry) SetBeta(on bool) {
	reg.Lock()
	defer reg.Unlock()

	reg.Beta = on
}

// AddResource adds a resource to the registry. Does _not_ select new version.
func (reg *ResourceRegistry) AddResource(identifier, version string, available, stableRelease, betaRelease bool) error {
	reg.Lock()
	defer reg.Unlock()

	err := reg.addResource(identifier, version, available, stableRelease, betaRelease)
	return err
}

func (reg *ResourceRegistry) addResource(identifier, version string, available, stableRelease, betaRelease bool) error {
	res, ok := reg.resources[identifier]
	if !ok {
		res = reg.newResource(identifier)
		reg.resources[identifier] = res
	}
	return res.AddVersion(version, available, stableRelease, betaRelease)
}

// AddResources adds resources to the registry. Errors are logged, the last one is returned. Despite errors, non-failing resources are still added. Does _not_ select new versions.
func (reg *ResourceRegistry) AddResources(versions map[string]string, available, stableRelease, betaRelease bool) error {
	reg.Lock()
	defer reg.Unlock()

	// add versions and their flags to registry
	var lastError error
	for identifier, version := range versions {
		lastError = reg.addResource(identifier, version, available, stableRelease, betaRelease)
		if lastError != nil {
			log.Warningf("%s: failed to add resource %s: %s", reg.Name, identifier, lastError)
		}
	}

	return lastError
}

// SelectVersions selects new resource versions depending on the current registry state.
func (reg *ResourceRegistry) SelectVersions() {
	reg.RLock()
	defer reg.RUnlock()

	for _, res := range reg.resources {
		res.Lock()
		res.selectVersion()
		res.Unlock()
	}
}

// GetSelectedVersions returns a list of the currently selected versions.
func (reg *ResourceRegistry) GetSelectedVersions() (versions map[string]string) {
	reg.RLock()
	defer reg.RUnlock()

	for _, res := range reg.resources {
		res.Lock()
		versions[res.Identifier] = res.SelectedVersion.VersionNumber
		res.Unlock()
	}

	return
}

// Purge deletes old updates, retaining a certain amount, specified by the keep parameter. Will at least keep 2 updates per resource.
func (reg *ResourceRegistry) Purge(keep int) {
	reg.RLock()
	defer reg.RUnlock()

	for _, res := range reg.resources {
		res.Purge(keep)
	}
}

// Cleanup removes temporary files.
func (reg *ResourceRegistry) Cleanup() error {
	// delete download tmp dir
	return os.RemoveAll(reg.tmpDir.Path)
}
