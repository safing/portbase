package api

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/tevino/abool"

	"github.com/safing/portbase/modules"

	"github.com/safing/portbase/log"
	"github.com/safing/portbase/rng"
)

// Permission defines an API requests permission.
type Permission int8

const (
	// NotFound declares that the operation does not exist.
	NotFound Permission = -2

	// Require declares that the operation requires permission to be processed,
	// but anyone can execute the operation.
	Require Permission = -1

	// NotSupported declares that the operation is not supported.
	NotSupported Permission = 0

	// PermitAnyone declares that anyone can execute the operation without any
	// authentication.
	PermitAnyone Permission = 1

	// PermitUser declares that the operation may be executed by authenticated
	// third party applications that are categorized as representing a simple
	// user and is limited in access.
	PermitUser Permission = 2

	// PermitAdmin declares that the operation may be executed by authenticated
	// third party applications that are categorized as representing an
	// administrator and has broad in access.
	PermitAdmin Permission = 3

	// PermitSelf declares that the operation may only be executed by the
	// software itself and its own (first party) components.
	PermitSelf Permission = 4
)

// Authenticator is a function that can be set as the authenticator for the API endpoint. If none is set, all requests will be permitted.
type Authenticator func(ctx context.Context, s *http.Server, r *http.Request) (*AuthToken, error)

// AuthToken represents either a set of required or granted permissions.
// All attributes must be set when the struct is built and must not be changed
// later. Functions may be called at any time.
// The Write permission implicitly also includes reading.
type AuthToken struct {
	Read  Permission
	Write Permission

	validUntil time.Time
	validLock  sync.Mutex
}

// Expired returns whether the token has expired.
func (token *AuthToken) Expired() bool {
	token.validLock.Lock()
	defer token.validLock.Unlock()

	return time.Now().After(token.validUntil)
}

// Refresh refreshes the validity of the token with the given TTL.
func (token *AuthToken) Refresh(ttl time.Duration) {
	token.validLock.Lock()
	defer token.validLock.Unlock()

	token.validUntil = time.Now().Add(ttl)
}

// AuthenticatedHandler defines the handler interface to specify custom
// permission for an API handler.
type AuthenticatedHandler interface {
	ReadPermission(*http.Request) Permission
	WritePermission(*http.Request) Permission
}

const (
	cookieName = "Portmaster-API-Token"
	cookieTTL  = 5 * time.Minute
)

var (
	authFnSet = abool.New()
	authFn    Authenticator

	authTokens     = make(map[string]*AuthToken)
	authTokensLock sync.Mutex

	// ErrAPIAccessDeniedMessage should be returned by Authenticator functions in
	// order to signify a blocked request, including a error message for the user.
	ErrAPIAccessDeniedMessage = errors.New("")
)

// SetAuthenticator sets an authenticator function for the API endpoint. If none is set, all requests will be permitted.
func SetAuthenticator(fn Authenticator) error {
	if module.Online() {
		return ErrAuthenticationImmutable
	}

	if authFnSet.IsSet() {
		return ErrAuthenticationAlreadySet
	}

	authFn = fn
	authFnSet.Set()
	return nil
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := authenticateRequest(w, r, nil)
		if token != nil {
			if _, apiRequest := getAPIContext(r); apiRequest != nil {
				apiRequest.AuthToken = token
			}
			next.ServeHTTP(w, r)
		}
	})
}

func authenticateRequest(w http.ResponseWriter, r *http.Request, targetHandler http.Handler) *AuthToken {
	tracer := log.Tracer(r.Context())

	// Check if authenticator is set.
	if !authFnSet.IsSet() {
		// Return highest available permissions.
		return &AuthToken{
			Read:  PermitSelf,
			Write: PermitSelf,
		}
	}

	// Check if request is read only.
	readRequest := isReadMethod(r.Method)

	// Get required permission for target handler.
	requiredPermission := PermitSelf
	if authdHandler, ok := targetHandler.(AuthenticatedHandler); ok {
		if readRequest {
			requiredPermission = authdHandler.ReadPermission(r)
		} else {
			requiredPermission = authdHandler.WritePermission(r)
		}
	}

	// Check if we need to do any authentication at all.
	switch requiredPermission {
	case NotFound:
		// Not found.
		tracer.Trace("api: authenticated handler reported: not found")
		http.Error(w, "Not found.", http.StatusNotFound)
		return nil
	case NotSupported:
		// A read or write permission can be marked as not supported.
		tracer.Trace("api: authenticated handler reported: not supported")
		http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
		return nil
	case PermitAnyone:
		// Don't process permissions, as we don't need them.
		tracer.Tracef("api: granted %s access to public handler", r.RemoteAddr)
		return &AuthToken{
			Read:  PermitAnyone,
			Write: PermitAnyone,
		}
	case Require:
		// Continue processing permissions, but treat as PermitAnyone.
		requiredPermission = PermitAnyone
	}

	// Check for valid permission after handling the specials.
	if requiredPermission < PermitAnyone || requiredPermission > PermitSelf {
		tracer.Warningf(
			"api: handler returned invalid permission: %s (%d)",
			requiredPermission,
			requiredPermission,
		)
		http.Error(w, "Internal server error during authentication.", http.StatusInternalServerError)
		return nil
	}

	// Check for an existing auth token.
	token := checkAuthToken(r)

	// Get auth token from authenticator if none was in the request.
	if token == nil {
		var err error
		token, err = authFn(r.Context(), server, r)
		if err != nil {
			// Check for internal error.
			if !errors.Is(err, ErrAPIAccessDeniedMessage) {
				tracer.Warningf("api: authenticator failed: %s", err)
				http.Error(w, "Internal server error during authentication.", http.StatusInternalServerError)
				return nil
			}

			// If authentication failed and we require authentication, return an
			// authentication error.
			if requiredPermission != PermitAnyone {
				// Return authentication error.
				tracer.Warningf("api: denying api access to %s", r.RemoteAddr)
				http.Error(w, err.Error(), http.StatusForbidden)
				return nil
			}

			token = &AuthToken{
				Read:  PermitAnyone,
				Write: PermitAnyone,
			}
		}

		// Apply auth token to request.
		err = applyAuthToken(w, token)
		if err != nil {
			tracer.Warningf("api: failed to create auth token: %s", err)
		}
	}

	// Get effective permission for request.
	var requestPermission Permission
	if readRequest {
		requestPermission = token.Read
	} else {
		requestPermission = token.Write
	}

	// Check for valid request permission.
	if requestPermission < PermitAnyone || requestPermission > PermitSelf {
		tracer.Warningf(
			"api: authenticator returned invalid permission: %s (%d)",
			requestPermission,
			requestPermission,
		)
		http.Error(w, "Internal server error during authentication.", http.StatusInternalServerError)
		return nil
	}

	// Check permission.
	if requestPermission < requiredPermission {
		http.Error(w, "Insufficient permissions.", http.StatusForbidden)
		return nil
	}

	tracer.Tracef("api: granted %s access to authenticated handler", r.RemoteAddr)
	return token
}

func checkAuthToken(r *http.Request) *AuthToken {
	// Get auth token from request.
	c, err := r.Cookie(cookieName)
	if err != nil {
		return nil
	}

	// Check if auth token is registered.
	authTokensLock.Lock()
	token, ok := authTokens[c.Value]
	authTokensLock.Unlock()
	if !ok {
		log.Tracer(r.Context()).Tracef("api: provided auth token %s is unknown", c.Value)
		return nil
	}

	// Check if token is still valid.
	if token.Expired() {
		log.Tracer(r.Context()).Tracef("api: provided auth token %s has expired", c.Value)
		return nil
	}

	// Refresh token and return.
	token.Refresh(cookieTTL)
	log.Tracer(r.Context()).Tracef("api: auth token %s is valid, refreshing", c.Value)
	return token
}

func applyAuthToken(w http.ResponseWriter, token *AuthToken) error {
	// Generate new token secret.
	secret, err := rng.Bytes(32) // 256 bit
	if err != nil {
		return err
	}
	secretHex := base64.RawURLEncoding.EncodeToString(secret)

	// Set token cookie in response.
	http.SetCookie(w, &http.Cookie{
		Name:     cookieName,
		Value:    secretHex,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// Set token TTL.
	token.Refresh(cookieTTL)

	// Save token.
	authTokensLock.Lock()
	defer authTokensLock.Unlock()
	authTokens[secretHex] = token

	return nil
}

func cleanAuthTokens(_ context.Context, _ *modules.Task) error {
	authTokensLock.Lock()
	defer authTokensLock.Unlock()

	for secret, token := range authTokens {
		if token.Expired() {
			delete(authTokens, secret)
		}
	}

	return nil
}

func isReadMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead
}

func (p Permission) String() string {
	switch p {
	case NotSupported:
		return "NotSupported"
	case Require:
		return "Require"
	case PermitAnyone:
		return "PermitAnyone"
	case PermitUser:
		return "PermitUser"
	case PermitAdmin:
		return "PermitAdmin"
	case PermitSelf:
		return "PermitSelf"
	case NotFound:
		return "NotFound"
	default:
		return "Unknown"
	}
}
