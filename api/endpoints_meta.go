package api

import (
	"encoding/json"
	"errors"
)

func registerMetaEndpoints() error {
	if err := RegisterEndpoint(Endpoint{
		Path:     "endpoints",
		Read:     PermitAnyone,
		MimeType: MimeTypeJSON,
		DataFn:   listEndpoints,
	}); err != nil {
		return err
	}

	if err := RegisterEndpoint(Endpoint{
		Path:     "permission",
		Read:     Require,
		StructFn: permissions,
	}); err != nil {
		return err
	}

	if err := RegisterEndpoint(Endpoint{
		Path:     "ping",
		Read:     PermitAnyone,
		ActionFn: ping,
	}); err != nil {
		return err
	}

	return nil
}

func listEndpoints(ar *Request) (data []byte, err error) {
	endpointsLock.Lock()
	defer endpointsLock.Unlock()

	data, err = json.Marshal(endpoints)
	return
}

func permissions(ar *Request) (i interface{}, err error) {
	if ar.AuthToken == nil {
		return nil, errors.New("authentication token missing")
	}

	return struct {
		Read          Permission
		Write         Permission
		ReadPermName  string
		WritePermName string
	}{
		Read:          ar.AuthToken.Read,
		Write:         ar.AuthToken.Write,
		ReadPermName:  ar.AuthToken.Read.String(),
		WritePermName: ar.AuthToken.Write.String(),
	}, nil
}

func ping(ar *Request) (msg string, err error) {
	return "Pong.", nil
}
