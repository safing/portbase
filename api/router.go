package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

var (
	additionalRoutes map[string]func(arg1 http.ResponseWriter, arg2 *http.Request)
)

func RegisterAdditionalRoute(path string, handleFunc func(arg1 http.ResponseWriter, arg2 *http.Request)) {
	if additionalRoutes == nil {
		additionalRoutes = make(map[string]func(arg1 http.ResponseWriter, arg2 *http.Request))
	}
	additionalRoutes[path] = handleFunc
}

func Serve() {

	router := mux.NewRouter()
	router.HandleFunc("/api/database/v1", startDatabaseAPI)

	for path, handleFunc := range additionalRoutes {
		router.HandleFunc(path, handleFunc)
	}

}
