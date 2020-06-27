package api

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/safing/portbase/modules"

	"github.com/safing/portbase/log"
	"github.com/safing/portbase/rng"
)

var (
	validTokens     = make(map[string]time.Time)
	validTokensLock sync.Mutex

	authFnLock sync.Mutex
	authFn     Authenticator

	// ErrAPIAccessDeniedMessage should be returned by Authenticator functions in
	// order to signify a blocked request, including a error message for the user.
	ErrAPIAccessDeniedMessage = errors.New("")
)

const (
	cookieName = "Portmaster-API-Token"

	cookieTTL = 5 * time.Minute
)

// Authenticator is a function that can be set as the authenticator for the API endpoint. If none is set, all requests will be permitted.
type Authenticator func(ctx context.Context, s *http.Server, r *http.Request) (err error)

// SetAuthenticator sets an authenticator function for the API endpoint. If none is set, all requests will be permitted.
func SetAuthenticator(fn Authenticator) error {
	if module.Online() {
		return ErrAuthenticationAlreadySet
	}

	authFnLock.Lock()
	defer authFnLock.Unlock()

	if authFn != nil {
		return ErrAuthenticationAlreadySet
	}

	authFn = fn
	return nil
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracer := log.Tracer(r.Context())

		// get authenticator
		authFnLock.Lock()
		authenticator := authFn
		authFnLock.Unlock()

		// permit if no authenticator set
		if authenticator == nil {
			next.ServeHTTP(w, r)
			return
		}

		// check existing auth cookie
		c, err := r.Cookie(cookieName)
		if err == nil {
			// get token
			validTokensLock.Lock()
			validUntil, valid := validTokens[c.Value]
			validTokensLock.Unlock()

			// check if token is valid
			if valid && time.Now().Before(validUntil) {
				tracer.Tracef("api: auth token %s is valid, refreshing", c.Value)
				// refresh cookie
				validTokensLock.Lock()
				validTokens[c.Value] = time.Now().Add(cookieTTL)
				validTokensLock.Unlock()
				// continue
				next.ServeHTTP(w, r)
				return
			}

			tracer.Tracef("api: provided auth token %s is invalid", c.Value)
		}

		// get auth decision
		err = authenticator(r.Context(), server, r)
		if err != nil {
			if errors.Is(err, ErrAPIAccessDeniedMessage) {
				tracer.Warningf("api: denying api access to %s", r.RemoteAddr)
				http.Error(w, err.Error(), http.StatusForbidden)
			} else {
				tracer.Warningf("api: authenticator failed: %s", err)
				http.Error(w, "Internal server error during authentication.", http.StatusInternalServerError)
			}
			return
		}

		// generate new token
		token, err := rng.Bytes(32) // 256 bit
		if err != nil {
			tracer.Warningf("api: failed to generate random token: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		tokenString := base64.RawURLEncoding.EncodeToString(token)
		// write new cookie
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    tokenString,
			HttpOnly: true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int(cookieTTL.Seconds()),
		})
		// save cookie
		validTokensLock.Lock()
		validTokens[tokenString] = time.Now().Add(cookieTTL)
		validTokensLock.Unlock()

		// serve
		tracer.Tracef("api: granted %s, assigned auth token %s", r.RemoteAddr, tokenString)
		next.ServeHTTP(w, r)
	})
}

func cleanAuthTokens(_ context.Context, _ *modules.Task) error {
	validTokensLock.Lock()
	defer validTokensLock.Unlock()

	now := time.Now()
	for token, validUntil := range validTokens {
		if now.After(validUntil) {
			delete(validTokens, token)
		}
	}

	return nil
}
