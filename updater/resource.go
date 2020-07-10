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

	// Identifier is the unique identifier for that resource.
	// It forms a file path using a forward-slash as the
	// path separator.
	Identifier string

	// Versions holds all available resource versions.
	Versions []*ResourceVersion

	// ActiveVersion is the last version of the resource
	// that someone requested using GetFile().
	ActiveVersion *ResourceVersion

	// SelectedVersion is newest, selectable version of
	// that resource that is available. A version
	// is selectable if it's not blacklisted by the user.
	// Note that it's not guaranteed that the selected version
	// is available locally. In that case, GetFile will attempt
	// to download the latest version from the updates servers
	// specified in the resource registry.
	SelectedVersion *ResourceVersion
}

// ResourceVersion represents a single version of a resource.
type ResourceVersion struct {
	resource *Resource

	// VersionNumber is the string representation of the resource
	// version.
	VersionNumber string
	semVer        *semver.Version

	// Available indicates if this version is available locally.
	Available bool

	// StableRelease indicates that this version is part of
	// a stable release index file.
	StableRelease bool

	// BetaRelease indicates that this version is part of
	// a beta release index file.
	BetaRelease bool

	// Blacklisted may be set to true if this version should
	// be skipped and not used. This is useful if the version
	// is known to be broken.
	Blacklisted bool
}

func (rv *ResourceVersion) String() string {
	return rv.VersionNumber
}

// isSelectable returns true if the version represented by rv is selectable.
// A version is selectable if it's not blacklisted and either already locally
// available or ready to be downloaded.
func (rv *ResourceVersion) isSelectable() bool {
	return !rv.Blacklisted && (rv.Available || rv.resource.registry.Online)
}

// isBetaVersionNumber checks if rv is marked as a beta version by checking
// the version string. It does not honor the BetaRelease field of rv!
func (rv *ResourceVersion) isBetaVersionNumber() bool {
	// "b" suffix check if for backwards compatibility
	// new versions should use the pre-release suffix as
	// declared by https://semver.org
	// i.e. 1.2.3-beta
	return strings.HasSuffix(rv.VersionNumber, "b") || strings.Contains(rv.semVer.Prerelease(), "beta")
}

// Len is the number of elements in the collection.
// It implements sort.Interface for ResourceVersion.
func (res *Resource) Len() int {
	return len(res.Versions)
}

// Less reports whether the element with index i should
// sort before the element with index j.
// It implements sort.Interface for ResourceVersions.
func (res *Resource) Less(i, j int) bool {
	return res.Versions[i].semVer.GreaterThan(res.Versions[j].semVer)
}

// Swap swaps the elements with indexes i and j.
// It implements sort.Interface for ResourceVersions.
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

// inUse returns true if the resource is currently in use.
func (res *Resource) inUse() bool {
	return res.ActiveVersion != nil
}

// AnyVersionAvailable returns true if any version of
// res is locally available.
func (res *Resource) AnyVersionAvailable() bool {
	res.Lock()
	defer res.Unlock()

	return res.available()
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

//nolint:gocognit // function already kept as simple as possible
func (res *Resource) selectVersion() {
	sort.Sort(res)

	// export after we finish
	defer func() {
		log.Debugf("updater: selected version %s for resource %s", res.SelectedVersion, res.Identifier)

		if res.inUse() &&
			res.SelectedVersion != res.ActiveVersion && // new selected version does not match previously selected version
			res.notifier != nil {

			res.notifier.markAsUpgradeable()
			res.notifier = nil

			log.Debugf("updater: active version of %s is %s, update available", res.Identifier, res.ActiveVersion.VersionNumber)
		}
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
				if rv.isSelectable() {
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
			if rv.isSelectable() {
				res.SelectedVersion = rv
				return
			}
			break
		}
	}

	// 4) Latest stable release
	for _, rv := range res.Versions {
		if !rv.isBetaVersionNumber() && rv.isSelectable() {
			res.SelectedVersion = rv
			return
		}
	}

	// 5) Latest of any type
	for _, rv := range res.Versions {
		if rv.isSelectable() {
			res.SelectedVersion = rv
			return
		}
	}

	// 6) Default to newest
	res.SelectedVersion = res.Versions[0]
	log.Warningf("updater: falling back to version %s for %s because we failed to find a selectable one", res.SelectedVersion, res.Identifier)
}

// Blacklist blacklists the specified version and selects a new version.
func (res *Resource) Blacklist(version string) error {
	res.Lock()
	defer res.Unlock()

	// count available and valid versions
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

// Purge deletes old updates, retaining a certain amount, specified by
// the keep parameter. Purge will always keep at least 2 versions so
// specifying a smaller keep value will have no effect. Note that
// blacklisted versions are not counted for the keep parameter.
// After purging a new version will be selected.
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
