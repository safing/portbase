// Copyright Safing ICS Technologies GmbH. Use of this source code is governed by the AGPL license that can be found in the LICENSE file.

package api

import (
	"net/http"

	"github.com/gorilla/mux"
)

func NewRouter() *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	for _, route := range routes {
		var handler http.Handler

		handler = route.Handler
		handler = Logger(handler, route.Name)

		router.
			Methods(route.Method).
			PathPrefix(route.Path).
			Name(route.Name).
			Handler(handler)
	}

	return router
}
