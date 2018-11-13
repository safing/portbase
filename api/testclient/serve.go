package testclient

import (
	"net/http"

	"github.com/Safing/portbase/api"
)

func init() {
	api.RegisterAdditionalRoute("/test/", http.StripPrefix("/test/", http.FileServer(http.Dir("./api/testclient/root/"))))
}
