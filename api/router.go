package api

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Safing/portbase/log"
)

var (
	router = mux.NewRouter()
)

// RegisterHandleFunc registers an additional handle function with the API endoint.
func RegisterHandleFunc(path string, handleFunc func(http.ResponseWriter, *http.Request)) *mux.Route {
	return router.HandleFunc(path, handleFunc)
}

// RequestLogger is a logging middleware
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Tracef("api request: %s ___ %s", r.RemoteAddr, r.RequestURI)
		ew := NewEnrichedResponseWriter(w)
		next.ServeHTTP(ew, r)
		log.Infof("api request: %s %d %s", r.RemoteAddr, ew.Status, r.RequestURI)
	})
}

// Serve starts serving the API endpoint.
func Serve() {
	router.Use(RequestLogger)

	mainMux := http.NewServeMux()
	mainMux.Handle("/", router)                              // net/http pattern matching /*
	mainMux.HandleFunc("/api/database/v1", startDatabaseAPI) // net/http pattern matching only this exact path

	address := getListenAddress()
	log.Infof("api: starting to listen on %s", address)
	log.Errorf("api: failed to listen on %s: %s", address, http.ListenAndServe(address, mainMux))
}
