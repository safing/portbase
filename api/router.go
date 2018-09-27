package api

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/Safing/portbase/log"
)

var (
	additionalRoutes map[string]http.Handler
)

func RegisterAdditionalRoute(path string, handler http.Handler) {
	if additionalRoutes == nil {
		additionalRoutes = make(map[string]http.Handler)
	}
	additionalRoutes[path] = handler
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ew := NewEnrichedResponseWriter(w)
		next.ServeHTTP(ew, r)
		log.Infof("api request: %s %d %s", r.RemoteAddr, ew.Status, r.RequestURI)
	})
}

func Serve() {

	router := mux.NewRouter()
	// router.HandleFunc("/api/database/v1", startDatabaseAPI)

	for path, handler := range additionalRoutes {
		router.Handle(path, handler)
	}

	router.Use(logger)

	http.Handle("/", router)
	http.HandleFunc("/api/database/v1", startDatabaseAPI)

	address := "127.0.0.1:18"
	log.Infof("api: starting to listen on %s", address)
	log.Errorf("api: failed to listen on %s: %s", address, http.ListenAndServe(address, nil))
}
