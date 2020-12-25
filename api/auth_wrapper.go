package api

import "net/http"

// WrapInAuthHandler wraps a simple http.HandlerFunc into a handler that
// exposes the given API permissions.
func WrapInAuthHandler(fn http.HandlerFunc, read, write Permission) http.Handler {
	return &wrappedAuthenticatedHandler{
		handleFunc: fn,
		read:       read,
		write:      write,
	}
}

type wrappedAuthenticatedHandler struct {
	handleFunc http.HandlerFunc
	read       Permission
	write      Permission
}

// ServeHTTP handles the http request.
func (wah *wrappedAuthenticatedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	wah.handleFunc(w, r)
}

// ReadPermission returns the read permission for the handler.
func (wah *wrappedAuthenticatedHandler) ReadPermission(r *http.Request) Permission {
	return wah.read
}

// WritePermission returns the write permission for the handler.
func (wah *wrappedAuthenticatedHandler) WritePermission(r *http.Request) Permission {
	return wah.write
}
