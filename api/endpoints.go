package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gorilla/mux"

	"github.com/safing/portbase/database/record"
	"github.com/safing/portbase/log"
	"github.com/safing/portbase/modules"
)

// Endpoint describes an API Endpoint.
// Path and at least one permission are required.
// As is exactly one function.
type Endpoint struct {
	Path      string
	MimeType  string
	Read      Permission      `json:",omitempty"`
	Write     Permission      `json:",omitempty"`
	BelongsTo *modules.Module `json:"-"`

	// ActionFunc is for simple actions with a return message for the user.
	ActionFunc ActionFunc `json:"-"`

	// DataFunc is for returning raw data that the caller for further processing.
	DataFunc DataFunc `json:"-"`

	// StructFunc is for returning any kind of struct.
	StructFunc StructFunc `json:"-"`

	// RecordFunc is for returning a database record. It will be properly locked
	// and marshalled including metadata.
	RecordFunc RecordFunc `json:"-"`

	// HandlerFunc is the raw http handler.
	HandlerFunc http.HandlerFunc `json:"-"`

	// Documentation Metadata.

	Name        string
	Description string
	Parameters  []Parameter `json:",omitempty"`
}

// Parameter describes a parameterized variation of an endpoint.
type Parameter struct {
	Method      string
	Field       string
	Value       string
	Description string
}

type (
	// ActionFunc is for simple actions with a return message for the user.
	ActionFunc func(ar *Request) (msg string, err error)

	// DataFunc is for returning raw data that the caller for further processing.
	DataFunc func(ar *Request) (data []byte, err error)

	// StructFunc is for returning any kind of struct.
	StructFunc func(ar *Request) (i interface{}, err error)

	// RecordFunc is for returning a database record. It will be properly locked
	// and marshalled including metadata.
	RecordFunc func(ar *Request) (r record.Record, err error)
)

// MIME Types.
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
	endpointsMux  = mux.NewRouter()
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

	endpointsLock.RLock()
	defer endpointsLock.RUnlock()

	// Get handler for request.
	// Gorilla does not support handling this on our own very well.
	// See github.com/gorilla/mux.ServeHTTP for reference.
	var match mux.RouteMatch
	var handler http.Handler
	if endpointsMux.Match(r, &match) {
		handler = match.Handler
		apiRequest.Route = match.Route
		// Add/Override variables instead of replacing.
		for k, v := range match.Vars {
			apiRequest.URLVars[k] = v
		}
	} else {
		return nil, apiRequest
	}

	apiEndpoint, ok = handler.(*Endpoint)
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
	endpointsMux.Handle(apiV1Path+e.Path, &e)
	return nil
}

func (e *Endpoint) check() error {
	// Check path.
	if strings.TrimSpace(e.Path) == "" {
		return errors.New("path is missing")
	}

	// Check permissions.
	if e.Read < Dynamic || e.Read > PermitSelf {
		return errors.New("invalid read permission")
	}
	if e.Write < Dynamic || e.Write > PermitSelf {
		return errors.New("invalid write permission")
	}

	// Check functions.
	var defaultMimeType string
	fnCnt := 0
	if e.ActionFunc != nil {
		fnCnt++
		defaultMimeType = MimeTypeText
	}
	if e.DataFunc != nil {
		fnCnt++
		defaultMimeType = MimeTypeText
	}
	if e.StructFunc != nil {
		fnCnt++
		defaultMimeType = MimeTypeJSON
	}
	if e.RecordFunc != nil {
		fnCnt++
		defaultMimeType = MimeTypeJSON
	}
	if e.HandlerFunc != nil {
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

// ExportEndpoints exports the registered endpoints. The returned data must be
// treated as immutable.
func ExportEndpoints() []*Endpoint {
	endpointsLock.RLock()
	defer endpointsLock.RUnlock()

	// Copy the map into a slice.
	eps := make([]*Endpoint, 0, len(endpoints))
	for _, ep := range endpoints {
		eps = append(eps, ep)
	}

	sort.Sort(sortByPath(eps))
	return eps
}

type sortByPath []*Endpoint

func (eps sortByPath) Len() int           { return len(eps) }
func (eps sortByPath) Less(i, j int) bool { return eps[i].Path < eps[j].Path }
func (eps sortByPath) Swap(i, j int)      { eps[i], eps[j] = eps[j], eps[i] }

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

	apiEndpoint.ServeHTTP(w, r)
}

// ServeHTTP handles the http request.
func (e *Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	_, apiRequest := getAPIContext(r)
	if apiRequest == nil {
		http.NotFound(w, r)
		return
	}

	// Wait for the owning module to be ready.
	if !moduleIsReady(e.BelongsTo) {
		http.Error(w, "The API endpoint is not ready yet or the its module is not enabled. Please try again later.", http.StatusServiceUnavailable)
		return
	}

	switch r.Method {
	case http.MethodHead:
		w.WriteHeader(http.StatusOK)
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
	case http.MethodOptions:
		w.WriteHeader(http.StatusNoContent)
		return
	default:
		http.Error(w, "unsupported method for the actions API", http.StatusMethodNotAllowed)
		return
	}

	// Execute action function and get response data
	var responseData []byte
	var err error

	switch {
	case e.ActionFunc != nil:
		var msg string
		msg, err = e.ActionFunc(apiRequest)
		if !strings.HasSuffix(msg, "\n") {
			msg += "\n"
		}
		if err == nil {
			responseData = []byte(msg)
		}

	case e.DataFunc != nil:
		responseData, err = e.DataFunc(apiRequest)

	case e.StructFunc != nil:
		var v interface{}
		v, err = e.StructFunc(apiRequest)
		if err == nil && v != nil {
			responseData, err = json.Marshal(v)
		}

	case e.RecordFunc != nil:
		var rec record.Record
		rec, err = e.RecordFunc(apiRequest)
		if err == nil && r != nil {
			responseData, err = marshalRecord(rec, false)
		}

	case e.HandlerFunc != nil:
		e.HandlerFunc(w, r)
		return

	default:
		http.Error(w, "missing handler", http.StatusInternalServerError)
		return
	}

	// Check for handler error.
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Write response.
	w.Header().Set("Content-Type", e.MimeType+"; charset=utf-8")
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
		http.Error(w, "too much input data", http.StatusRequestEntityTooLarge)
		return nil, false
	}

	// Read and close body.
	inputData, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body"+err.Error(), http.StatusInternalServerError)
		return nil, false
	}
	return inputData, true
}
