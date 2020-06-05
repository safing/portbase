package api

import (
	"context"
	"net/http"

	"github.com/safing/portbase/log"
)

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
	// loop in reverse to build the handler chain in the correct order
	for i := len(mwh.handlers) - 1; i >= 0; i-- {
		handler = mwh.handlers[i](handler)
	}

	// start
	handler.ServeHTTP(w, r)
}

// ModuleWorker is an http middleware that wraps the request in a module worker.
func ModuleWorker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = module.RunWorker("http request", func(_ context.Context) error {
			next.ServeHTTP(w, r)
			return nil
		})
	})
}

// LogTracer is an http middleware that attaches a log tracer to the request context.
func LogTracer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, tracer := log.AddTracer(r.Context())
		next.ServeHTTP(w, r.WithContext(ctx))
		tracer.Submit()
	})
}
