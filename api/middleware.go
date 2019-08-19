package api

import "net/http"

// Middleware is a function that can be added as a middleware to the API endpoint.
type Middleware func(next http.Handler) http.Handler

type mwHandler struct {
	handlers []Middleware
	final    http.Handler
}

func (mwh *mwHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handlerLock.RLock()
	defer handlerLock.RUnlock()

	// final handler
	handler := mwh.final

	// build middleware chain
	for _, mw := range mwh.handlers {
		handler = mw(handler)
	}

	// start
	handler.ServeHTTP(w, r)
}
