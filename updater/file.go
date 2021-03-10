package updater

import (
	"io"
	"os"
	"strings"

	"github.com/safing/portbase/log"
	"github.com/safing/portbase/utils"

	semver "github.com/hashicorp/go-version"
)

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

// SemVer returns the semantic version of the file.
func (file *File) SemVer() *semver.Version {
	return file.version.semVer
}

// EqualsVersion normalizes the given version and checks equality with semver.
func (file *File) EqualsVersion(version string) bool {
	return file.version.EqualsVersion(version)
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

// Unpacker describes the function that is passed to
// File.Unpack. It receives a reader to the compressed/packed
// file and should return a reader that provides
// unpacked file contents. If the returned reader implements
// io.Closer it's close method is invoked when an error
// or io.EOF is returned from Read().
type Unpacker func(io.Reader) (io.Reader, error)

// Unpack returns the path to the unpacked version of file and
// unpacks it on demand using unpacker.
func (file *File) Unpack(suffix string, unpacker Unpacker) (string, error) {
	path := strings.TrimSuffix(file.Path(), suffix)

	if suffix == "" {
		path += "-unpacked"
	}

	_, err := os.Stat(path)
	if err == nil {
		return path, nil
	}

	if !os.IsNotExist(err) {
		return "", err
	}

	f, err := os.Open(file.Path())
	if err != nil {
		return "", err
	}
	defer f.Close()

	r, err := unpacker(f)
	if err != nil {
		return "", err
	}

	ioErr := utils.CreateAtomic(path, r, &utils.AtomicFileOptions{
		TempDir: file.resource.registry.TmpDir().Path,
	})

	if c, ok := r.(io.Closer); ok {
		if err := c.Close(); err != nil && ioErr == nil {
			// if ioErr is already set we ignore the error from
			// closing the unpacker.
			ioErr = err
		}
	}

	return path, ioErr
}
