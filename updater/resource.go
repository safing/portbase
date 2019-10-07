package updater

import (
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/safing/portbase/log"

	semver "github.com/hashicorp/go-version"
)

// Resource represents a resource (via an identifier) and multiple file versions.
type Resource struct {
	sync.Mutex
	registry *ResourceRegistry
	notifier *notifier

	Identifier string
	Versions   []*ResourceVersion

	ActiveVersion   *ResourceVersion
	SelectedVersion *ResourceVersion
	ForceDownload   bool
}

// ResourceVersion represents a single version of a resource.
type ResourceVersion struct {
	resource *Resource

	VersionNumber string
	semVer        *semver.Version
	Available     bool
	StableRelease bool
	BetaRelease   bool
	Blacklisted   bool
}

// Len is the number of elements in the collection. (sort.Interface for Versions)
func (res *Resource) Len() int {
	return len(res.Versions)
}

// Less reports whether the element with index i should sort before the element with index j. (sort.Interface for Versions)
func (res *Resource) Less(i, j int) bool {
	return res.Versions[i].semVer.GreaterThan(res.Versions[j].semVer)
}

// Swap swaps the elements with indexes i and j. (sort.Interface for Versions)
func (res *Resource) Swap(i, j int) {
	res.Versions[i], res.Versions[j] = res.Versions[j], res.Versions[i]
}

// available returns whether any version of the resource is available.
func (res *Resource) available() bool {
	for _, rv := range res.Versions {
		if rv.Available {
			return true
		}
	}
	return false
}

func (reg *ResourceRegistry) newResource(identifier string) *Resource {
	return &Resource{
		registry:   reg,
		Identifier: identifier,
		Versions:   make([]*ResourceVersion, 0, 1),
	}
}

// AddVersion adds a resource version to a resource.
func (res *Resource) AddVersion(version string, available, stableRelease, betaRelease bool) error {
	res.Lock()
	defer res.Unlock()

	// reset stable or beta release flags
	if stableRelease || betaRelease {
		for _, rv := range res.Versions {
			if stableRelease {
				rv.StableRelease = false
			}
			if betaRelease {
				rv.BetaRelease = false
			}
		}
	}

	var rv *ResourceVersion
	// check for existing version
	for _, possibleMatch := range res.Versions {
		if possibleMatch.VersionNumber == version {
			rv = possibleMatch
			break
		}
	}

	// create new version if none found
	if rv == nil {
		// parse to semver
		sv, err := semver.NewVersion(version)
		if err != nil {
			return err
		}

		rv = &ResourceVersion{
			resource:      res,
			VersionNumber: version,
			semVer:        sv,
		}
		res.Versions = append(res.Versions, rv)
	}

	// set flags
	if available {
		rv.Available = true
	}
	if stableRelease {
		rv.StableRelease = true
	}
	if betaRelease {
		rv.BetaRelease = true
	}

	return nil
}

// GetFile returns the selected version as a *File.
func (res *Resource) GetFile() *File {
	res.Lock()
	defer res.Unlock()

	// check for notifier
	if res.notifier == nil {
		// create new notifier
		res.notifier = newNotifier()
	}

	// check if version is selected
	if res.SelectedVersion == nil {
		res.selectVersion()
	}

	// create file
	return &File{
		resource:      res,
		version:       res.SelectedVersion,
		notifier:      res.notifier,
		versionedPath: res.SelectedVersion.versionedPath(),
		storagePath:   res.SelectedVersion.storagePath(),
	}
}

//nolint:gocognit // function already kept as simlpe as possible
func (res *Resource) selectVersion() {
	sort.Sort(res)

	// export after we finish
	defer func() {
		if res.ActiveVersion != nil && // resource has already been used
			res.SelectedVersion != res.ActiveVersion && // new selected version does not match previously selected version
			res.notifier != nil {
			res.notifier.markAsUpgradeable()
			res.notifier = nil
		}

		res.registry.notifyOfChanges()
	}()

	if len(res.Versions) == 0 {
		// TODO: find better way to deal with an empty version slice (which should not happen)
		res.SelectedVersion = nil
		return
	}

	// Target selection
	// 1) Dev release if dev mode is active and ignore blacklisting
	if res.registry.DevMode {
		// get last element
		rv := res.Versions[len(res.Versions)-1]
		// check if it's a dev version
		if rv.VersionNumber == "0" && rv.Available {
			res.SelectedVersion = rv
			return
		}
	}

	// 2) Beta release if beta is active
	if res.registry.Beta {
		for _, rv := range res.Versions {
			if rv.BetaRelease {
				if !rv.Blacklisted && (rv.Available || rv.resource.registry.Online) {
					res.SelectedVersion = rv
					return
				}
				break
			}
		}
	}

	// 3) Stable release
	for _, rv := range res.Versions {
		if rv.StableRelease {
			if !rv.Blacklisted && (rv.Available || rv.resource.registry.Online) {
				res.SelectedVersion = rv
				return
			}
			break
		}
	}

	// 4) Latest stable release
	for _, rv := range res.Versions {
		if !strings.HasSuffix(rv.VersionNumber, "b") && !rv.Blacklisted && (rv.Available || rv.resource.registry.Online) {
			res.SelectedVersion = rv
			return
		}
	}

	// 5) Latest of any type
	for _, rv := range res.Versions {
		if !rv.Blacklisted && (rv.Available || rv.resource.registry.Online) {
			res.SelectedVersion = rv
			return
		}
	}

	// 6) Default to newest
	res.SelectedVersion = res.Versions[0]
}

// Blacklist blacklists the specified version and selects a new version.
func (res *Resource) Blacklist(version string) error {
	res.Lock()
	defer res.Unlock()

	// count already blacklisted entries
	valid := 0
	for _, rv := range res.Versions {
		if rv.VersionNumber == "0" {
			continue // ignore dev versions
		}
		if !rv.Blacklisted {
			valid++
		}
	}
	if valid <= 1 {
		return errors.New("cannot blacklist last version") // last one, cannot blacklist!
	}

	// find version and blacklist
	for _, rv := range res.Versions {
		if rv.VersionNumber == version {
			// blacklist and update
			rv.Blacklisted = true
			res.selectVersion()
			return nil
		}
	}

	return errors.New("could not find version")
}

// Purge deletes old updates, retaining a certain amount, specified by the keep parameter. Will at least keep 2 updates per resource. After purging, new versions will be selected.
func (res *Resource) Purge(keep int) {
	res.Lock()
	defer res.Unlock()

	// safeguard
	if keep < 2 {
		keep = 2
	}

	// keep versions
	var validVersions int
	var skippedActiveVersion bool
	var skippedSelectedVersion bool
	var purgeFrom int
	for i, rv := range res.Versions {
		// continue to purging?
		if validVersions >= keep && // skip at least <keep> versions
			skippedActiveVersion && // skip until active version
			skippedSelectedVersion { // skip until selected version
			purgeFrom = i
			break
		}

		// keep active version
		if !skippedActiveVersion && rv == res.ActiveVersion {
			skippedActiveVersion = true
		}

		// keep selected version
		if !skippedSelectedVersion && rv == res.SelectedVersion {
			skippedSelectedVersion = true
		}

		// count valid (not blacklisted) versions
		if !rv.Blacklisted {
			validVersions++
		}
	}

	// check if there is anything to purge
	if purgeFrom < keep || purgeFrom > len(res.Versions) {
		return
	}

	// purge phase
	for _, rv := range res.Versions[purgeFrom:] {
		// delete
		err := os.Remove(rv.storagePath())
		if err != nil {
			log.Warningf("%s: failed to purge old resource %s: %s", res.registry.Name, rv.storagePath(), err)
		}
	}
	// remove entries of deleted files
	res.Versions = res.Versions[purgeFrom:]

	res.selectVersion()
}

func (rv *ResourceVersion) versionedPath() string {
	return GetVersionedPath(rv.resource.Identifier, rv.VersionNumber)
}

func (rv *ResourceVersion) storagePath() string {
	return filepath.Join(rv.resource.registry.storageDir.Path, filepath.FromSlash(rv.versionedPath()))
}
