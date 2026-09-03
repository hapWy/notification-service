package store

import "errors"

// ErrNotFound is returned by store methods when a lookup finds no row.
// Callers (service/API layers) should compare against it with errors.Is.
var ErrNotFound = errors.New("not found")
