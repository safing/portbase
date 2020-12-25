package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"sync"

	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/log"
)

// Endpoint describes an API Endpoint.
// Path and at least one permission are required.
// As is exactly one function.
type Endpoint struct {
	Path     string
	MimeType string
	Read     Permission
	Write    Permission

	// TODO: We _could_ expose more metadata to be able to build lists of actions
	// automatically.
	// Name           string
	// Description    string
	// Order          int
	// ExpertiseLevel config.ExpertiseLevel

	// ActionFn is for simple actions with a return message for the user.
	ActionFn ActionFn `json:"-"`

	// DataFn is for returning raw data that the caller for further processing.
	DataFn DataFn `json:"-"`

	// StructFn is for returning any kind of struct.
	StructFn StructFn `json:"-"`

	// RecordFn is for returning a database record. It will be properly locked
	// and marshalled including metadata.
	RecordFn RecordFn `json:"-"`

	// HandlerFn is the raw http handler.
	HandlerFn http.HandlerFunc `json:"-"`
}

type (
	// ActionFn is for simple actions with a return message for the user.
	ActionFn func(ar *Request) (msg string, err error)

	// DataFn is for returning raw data that the caller for further processing.
	DataFn func(ar *Request) (data []byte, err error)

	// StructFn is for returning any kind of struct.
	StructFn func(ar *Request) (i interface{}, err error)

	// RecordFn is for returning a database record. It will be properly locked
	// and marshalled including metadata.
	RecordFn func(ar *Request) (r record.Record, err error)
)

// MIME Types
const (
	MimeTypeJSON string = "application/json"
	MimeTypeText string = "text/plain"

	apiV1Path = "/api/v1/"
)

func init() {
	RegisterHandler(apiV1Path+"{endpointPath:.+}", &endpointHandler{})
}

var (
	endpoints     = make(map[string]*Endpoint)
	endpointsLock sync.RWMutex

	// ErrInvalidEndpoint is returned when an invalid endpoint is registered.
	ErrInvalidEndpoint = errors.New("endpoint is invalid")

	// ErrAlreadyRegistered is returned when there already is an endpoint with
	// the same path registered.
	ErrAlreadyRegistered = errors.New("an endpoint for this path is already registered")
)

func getAPIContext(r *http.Request) (apiEndpoint *Endpoint, apiRequest *Request) {
	// Get request context and check if we already have an action cached.
	apiRequest = GetAPIRequest(r)
	if apiRequest == nil {
		return nil, nil
	}
	var ok bool
	apiEndpoint, ok = apiRequest.HandlerCache.(*Endpoint)
	if ok {
		return apiEndpoint, apiRequest
	}

	// If not, get the action from the registry.
	endpointPath, ok := apiRequest.URLVars["endpointPath"]
	if !ok {
		return nil, apiRequest
	}

	endpointsLock.RLock()
	defer endpointsLock.RUnlock()

	apiEndpoint, ok = endpoints[endpointPath]
	if ok {
		// Cache for next operation.
		apiRequest.HandlerCache = apiEndpoint
	}
	return apiEndpoint, apiRequest
}

// RegisterEndpoint registers a new endpoint. An error will be returned if it
// does not pass the sanity checks.
func RegisterEndpoint(e Endpoint) error {
	if err := e.check(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidEndpoint, err)
	}

	endpointsLock.Lock()
	defer endpointsLock.Unlock()

	_, ok := endpoints[e.Path]
	if ok {
		return ErrAlreadyRegistered
	}

	endpoints[e.Path] = &e
	return nil
}

func (e *Endpoint) check() error {
	// Check path.
	if e.Path == "" {
		return errors.New("path is missing")
	}

	// Check permissions.
	if e.Read < Require || e.Read > PermitSelf {
		return errors.New("invalid read permission")
	}
	if e.Write < Require || e.Write > PermitSelf {
		return errors.New("invalid write permission")
	}

	// Check functions.
	var defaultMimeType string
	fnCnt := 0
	if e.ActionFn != nil {
		fnCnt++
		defaultMimeType = MimeTypeText
	}
	if e.DataFn != nil {
		fnCnt++
		defaultMimeType = MimeTypeText
	}
	if e.StructFn != nil {
		fnCnt++
		defaultMimeType = MimeTypeJSON
	}
	if e.RecordFn != nil {
		fnCnt++
		defaultMimeType = MimeTypeJSON
	}
	if e.HandlerFn != nil {
		fnCnt++
		defaultMimeType = MimeTypeText
	}
	if fnCnt != 1 {
		return errors.New("only one function may be set")
	}

	// Set default mime type.
	if e.MimeType == "" {
		e.MimeType = defaultMimeType
	}

	return nil
}

type endpointHandler struct{}

var _ AuthenticatedHandler = &endpointHandler{} // Compile time interface check.

// ReadPermission returns the read permission for the handler.
func (eh *endpointHandler) ReadPermission(r *http.Request) Permission {
	apiEndpoint, _ := getAPIContext(r)
	if apiEndpoint != nil {
		return apiEndpoint.Read
	}
	return NotFound
}

// WritePermission returns the write permission for the handler.
func (eh *endpointHandler) WritePermission(r *http.Request) Permission {
	apiEndpoint, _ := getAPIContext(r)
	if apiEndpoint != nil {
		return apiEndpoint.Write
	}
	return NotFound
}

// ServeHTTP handles the http request.
func (eh *endpointHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	apiEndpoint, apiRequest := getAPIContext(r)
	if apiEndpoint == nil || apiRequest == nil {
		http.NotFound(w, r)
		return
	}

	switch r.Method {
	case http.MethodHead:
		http.Error(w, "", http.StatusOK)
		return
	case http.MethodPost, http.MethodPut:
		// Read body data.
		inputData, ok := readBody(w, r)
		if !ok {
			return
		}
		apiRequest.InputData = inputData
	case http.MethodGet:
		// Nothing special to do here.
	default:
		http.Error(w, "Unsupported method for the actions API.", http.StatusMethodNotAllowed)
		return
	}

	// Execute action function and get response data
	var responseData []byte
	var err error

	switch {
	case apiEndpoint.ActionFn != nil:
		var msg string
		msg, err = apiEndpoint.ActionFn(apiRequest)
		if err == nil {
			responseData = []byte(msg)
		}

	case apiEndpoint.DataFn != nil:
		responseData, err = apiEndpoint.DataFn(apiRequest)

	case apiEndpoint.StructFn != nil:
		var v interface{}
		v, err = apiEndpoint.StructFn(apiRequest)
		if err == nil && v != nil {
			responseData, err = json.Marshal(v)
		}

	case apiEndpoint.RecordFn != nil:
		var rec record.Record
		rec, err = apiEndpoint.RecordFn(apiRequest)
		if err == nil && r != nil {
			responseData, err = marshalRecord(rec, false)
		}

	case apiEndpoint.HandlerFn != nil:
		apiEndpoint.HandlerFn(w, r)
		return

	default:
		http.Error(w, "Internal server error: Missing handler.", http.StatusInternalServerError)
		return
	}

	// Check for handler error.
	if err != nil {
		http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response.
	w.Header().Set("Content-Type", apiEndpoint.MimeType+"; charset=utf-8")
	w.Header().Set("Content-Length", strconv.Itoa(len(responseData)))
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(responseData)
	if err != nil {
		log.Tracer(r.Context()).Warningf("api: failed to write response: %s", err)
	}
}

func readBody(w http.ResponseWriter, r *http.Request) (inputData []byte, ok bool) {
	// Check for too long content in order to prevent death.
	if r.ContentLength > 20000000 { // 20MB
		http.Error(w, "Too much input data.", http.StatusBadRequest)
		return nil, false
	}

	// Read and close body.
	inputData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body: "+err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	r.Body.Close()
	return inputData, true
}
