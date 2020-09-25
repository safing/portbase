package fstree

import "github.com/safing/portbase/database/storage"

var (
	// Compile time interface checks.
	_ storage.Interface = &FSTree{}
)
