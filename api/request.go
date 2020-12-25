package api

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
)

// Request is a support struct to pool more request related information.
type Request struct {
	// Request is the http request.
	Request *http.Request

	// InputData contains the request body for write operations.
	InputData []byte

	// Route of this request.
	Route *mux.Route

	// URLVars contains the URL variables extracted by the gorilla mux.
	URLVars map[string]string

	// AuthToken is the request-side authentication token assigned.
	AuthToken *AuthToken

	// HandlerCache can be used by handlers to cache data between handlers within a request.
	HandlerCache interface{}
}

// Ctx is a shortcut to access the request context.
func (ar *Request) Ctx() context.Context {
	return ar.Request.Context()
}

// apiRequestContextKey is a key used for the context key/value storage.
type apiRequestContextKey struct{}

var (
	requestContextKey = apiRequestContextKey{}
)

// GetAPIRequest returns the API Request of the given http request.
func GetAPIRequest(r *http.Request) *Request {
	ar, ok := r.Context().Value(requestContextKey).(*Request)
	if ok {
		return ar
	}
	return nil
}
