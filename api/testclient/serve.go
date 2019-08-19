package testclient

import (
	"net/http"

	"github.com/safing/portbase/api"
)

func init() {
	api.RegisterHandler("/test/", http.StripPrefix("/test/", http.FileServer(http.Dir("./api/testclient/root/"))))
}
