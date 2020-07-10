package api

import "errors"

// API Errors.
var (
	ErrAuthenticationAlreadySet = errors.New("the authentication function has already been set")
	ErrAuthenticationImmutable  = errors.New("the authentication function can only be set before the api has started")
)

// internal errors.
var (
	errMissingMapValue = errors.New("values must be in a map")
	errInvalidKey      = errors.New("keys must be strings")
	errValueNotExists  = errors.New("value does not exist")
	errNoHijacker      = errors.New("response does not implement http.Hijacker")
	errNoListenAddr    = errors.New("no listen address for api available")
)
