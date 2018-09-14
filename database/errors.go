// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package database

import (
	"errors"
)

// Errors
var (
	ErrNotFound         = errors.New("database entry could not be found")
	ErrPermissionDenied = errors.New("access to database record denied")
	ErrReadOnly         = errors.New("database is read only")
	ErrShuttingDown     = errors.New("database system is shutting down")
)
