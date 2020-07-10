package record

import "errors"

// Common error definitions.
var (
	ErrMissingMeta        = errors.New("failed to get or missing meta")
	ErrUnwrapUnsupported  = errors.New("unwrap unsupported")
	ErrUnsupportedVersion = errors.New("version not supported")
	ErrFormatMismatch     = errors.New("could not dump model, wrapped object format mismatch")
)
