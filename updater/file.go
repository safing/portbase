package updater

import "github.com/safing/portbase/log"

// File represents a file from the update system.
type File struct {
	resource      *Resource
	version       *ResourceVersion
	notifier      *notifier
	versionedPath string
	storagePath   string
}

// Identifier returns the identifier of the file.
func (file *File) Identifier() string {
	return file.resource.Identifier
}

// Version returns the version of the file.
func (file *File) Version() string {
	return file.version.VersionNumber
}

// Path returns the absolute filepath of the file.
func (file *File) Path() string {
	return file.storagePath
}

// Blacklist notifies the update system that this file is somehow broken, and should be ignored from now on, until restarted.
func (file *File) Blacklist() error {
	return file.resource.Blacklist(file.version.VersionNumber)
}

// used marks the file as active
func (file *File) markActiveWithLocking() {
	file.resource.Lock()
	defer file.resource.Unlock()

	// update last used version
	if file.resource.ActiveVersion != file.version {
		log.Debugf("updater: setting active version of resource %s from %s to %s", file.resource.Identifier, file.resource.ActiveVersion, file.version.VersionNumber)
		file.resource.ActiveVersion = file.version
	}
}
