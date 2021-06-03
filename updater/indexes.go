package updater

// Index describes an index file pulled by the updater.
type Index struct {
	// Path is the path to the index file
	// on the update server.
	Path string

	// PreRelease signifies that all versions of this index should be marked as
	// pre-releases, no matter if the versions actually have a pre-release tag or
	// not.
	PreRelease bool
}
