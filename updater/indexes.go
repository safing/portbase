package updater

// Index describes an index file pulled by the updater.
type Index struct {
	// Path is the path to the index file
	// on the update server.
	Path string

	// Stable is set if the index file contains only stable
	// releases.
	Stable bool

	// Beta is set if the index file contains beta
	// releases.
	Beta bool
}
