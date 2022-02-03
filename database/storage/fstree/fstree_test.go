package fstree

import "github.com/safing/portbase/database/storage"

// Compile time interface checks.
var _ storage.Interface = &FSTree{}
