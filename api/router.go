package api

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"

	"github.com/safing/portbase/log"
)

var (
	// gorilla mux
	mainMux = mux.NewRouter()

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

// Serve starts serving the API endpoint.
func Serve() {
	// configure server
	server.Addr = listenAddressConfig()
	server.Handler = &mainHandler{
		// TODO: mainMux should not be modified anymore.
		mux: mainMux,
	}

	// start serving
	log.Infof("api: starting to listen on %s", server.Addr)
	backoffDuration := 10 * time.Second
	for {
		// always returns an error
		err := module.RunWorker("http endpoint", func(ctx context.Context) error {
			return server.ListenAndServe()
		})
		// return on shutdown error
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		// log error and restart
		log.Errorf("api: http endpoint failed: %s - restarting in %s", err, backoffDuration)
		time.Sleep(backoffDuration)
	}
}

type mainHandler struct {
	mux *mux.Router
}

func (mh *mainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_ = module.RunWorker("http request", func(_ context.Context) error {
		return mh.handle(w, r)
	})
}

func (mh *mainHandler) handle(w http.ResponseWriter, r *http.Request) error {
	// Setup context trace logging.
	ctx, tracer := log.AddTracer(r.Context())
	// Add request context.
	apiRequest := &Request{
		Request: r,
	}
	ctx = context.WithValue(ctx, requestContextKey, apiRequest)
	// Add context back to request.
	r = r.WithContext(ctx)
	lrw := NewLoggingResponseWriter(w, r)

	tracer.Tracef("api request: %s ___ %s %s", r.RemoteAddr, lrw.Request.Method, r.RequestURI)
	defer func() {
		// Log request status.
		if lrw.Status != 0 {
			// If lrw.Status is 0, the request may have been hijacked.
			tracer.Debugf("api request: %s %d %s %s", lrw.Request.RemoteAddr, lrw.Status, lrw.Request.Method, lrw.Request.RequestURI)
		}
		tracer.Submit()
	}()

	// Clean URL.
	cleanedRequestPath := cleanRequestPath(r.URL.Path)

	// If the cleaned URL differs from the original one, redirect to there.
	if r.URL.Path != cleanedRequestPath {
		redirURL := *r.URL
		redirURL.Path = cleanedRequestPath
		http.Redirect(lrw, r, redirURL.String(), http.StatusMovedPermanently)
		return nil
	}

	// Get handler for request.
	// Gorilla does not support handling this on our own very well.
	// See github.com/gorilla/mux.ServeHTTP for reference.
	var match mux.RouteMatch
	var handler http.Handler
	if mh.mux.Match(r, &match) {
		handler = match.Handler
		apiRequest.Route = match.Route
		apiRequest.URLVars = match.Vars
	}

	// Be sure that URLVars always is a map.
	if apiRequest.URLVars == nil {
		apiRequest.URLVars = make(map[string]string)
	}

	// Check authentication.
	apiRequest.AuthToken = authenticateRequest(lrw, r, handler)
	if apiRequest.AuthToken == nil {
		// Authenticator already replied.
		return nil
	}

	// Wait for the owning module to be ready.
	if moduleHandler, ok := handler.(ModuleHandler); ok {
		if !moduleIsReady(moduleHandler.BelongsTo()) {
			http.Error(lrw, "The API endpoint is not ready yet. Please try again later.", http.StatusServiceUnavailable)
			return nil
		}
	}

	// Add security headers.
	if !devMode() {
		w.Header().Set(
			"Content-Security-Policy",
			"default-src 'self'; "+
				"connect-src https://*.safing.io 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+
				"img-src 'self' data:",
		)
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "deny")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("X-DNS-Prefetch-Control", "off")
	} else {
		w.Header().Set("Access-Control-Allow-Origin", "*")
	}

	// Handle request.
	switch {
	case handler != nil:
		handler.ServeHTTP(lrw, r)
	case errors.Is(match.MatchErr, mux.ErrMethodMismatch):
		http.Error(lrw, "Method not allowed.", http.StatusMethodNotAllowed)
	default: // handler == nil or other error
		http.Error(lrw, "Not found.", http.StatusNotFound)
	}

	return nil
}

// cleanRequestPath cleans and returns a request URL.
func cleanRequestPath(requestPath string) string {
	// If the request URL is empty, return a request for "root".
	if requestPath == "" || requestPath == "/" {
		return "/"
	}
	// If the request URL does not start with a slash, prepend it.
	if !strings.HasPrefix(requestPath, "/") {
		requestPath = "/" + requestPath
	}

	// Clean path to remove any relative parts.
	cleanedRequestPath := path.Clean(requestPath)
	// Because path.Clean removes a trailing slash, we need to add it back here
	// if the original URL had one.
	if strings.HasSuffix(requestPath, "/") {
		cleanedRequestPath += "/"
	}

	return cleanedRequestPath
}
