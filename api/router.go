package api

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"

	"github.com/safing/portbase/log"
)

var (
	// gorilla mux
	mainMux = mux.NewRouter()

	// middlewares
	middlewareHandler = &mwHandler{
		final: mainMux,
		handlers: []Middleware{
			RequestLogger,
			authMiddleware,
		},
	}

	// main server and lock
	server      = &http.Server{}
	handlerLock sync.RWMutex
)

// RegisterHandler registers a handler with the API endoint.
func RegisterHandler(path string, handler http.Handler) *mux.Route {
	handlerLock.Lock()
	defer handlerLock.Unlock()
	return mainMux.Handle(path, handler)
}

// RegisterHandleFunc registers a handle function with the API endoint.
func RegisterHandleFunc(path string, handleFunc func(http.ResponseWriter, *http.Request)) *mux.Route {
	handlerLock.Lock()
	defer handlerLock.Unlock()
	return mainMux.HandleFunc(path, handleFunc)
}

// RegisterMiddleware registers a middle function with the API endoint.
func RegisterMiddleware(middleware Middleware) {
	handlerLock.Lock()
	defer handlerLock.Unlock()
	middlewareHandler.handlers = append(middlewareHandler.handlers, middleware)
}

// Serve starts serving the API endpoint.
func Serve() {
	// configure server
	server.Addr = GetAPIAddress()
	server.Handler = middlewareHandler

	// start serving
	log.Infof("api: starting to listen on %s", server.Addr)
	// TODO: retry if failed
	log.Errorf("api: failed to listen on %s: %s", server.Addr, server.ListenAndServe())
}

// GetMuxVars wraps github.com/gorilla/mux.Vars in order to mitigate context key issues in multi-repo projects.
func GetMuxVars(r *http.Request) map[string]string {
	return mux.Vars(r)
}
