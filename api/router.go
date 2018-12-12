package api

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Safing/portbase/log"
)

var (
	additionalRoutes map[string]http.Handler
)

// RegisterAdditionalRoute registers an additional route with the API endoint.
func RegisterAdditionalRoute(path string, handler http.Handler) {
	if additionalRoutes == nil {
		additionalRoutes = make(map[string]http.Handler)
	}
	additionalRoutes[path] = handler
}

// RequestLogger is a logging middleware
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ew := NewEnrichedResponseWriter(w)
		next.ServeHTTP(ew, r)
		log.Infof("api request: %s %d %s", r.RemoteAddr, ew.Status, r.RequestURI)
	})
}

// Serve starts serving the API endpoint.
func Serve() {

	router := mux.NewRouter()
	// router.HandleFunc("/api/database/v1", startDatabaseAPI)

	for path, handler := range additionalRoutes {
		router.Handle(path, handler)
	}

	router.Use(RequestLogger)

	http.Handle("/", router)
	http.HandleFunc("/api/database/v1", startDatabaseAPI)

	address := getListenAddress()
	log.Infof("api: starting to listen on %s", address)
	log.Errorf("api: failed to listen on %s: %s", address, http.ListenAndServe(address, nil))
}
