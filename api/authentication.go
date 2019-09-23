package api

import (
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/safing/portbase/crypto/random"
	"github.com/safing/portbase/log"
)

var (
	validTokens     map[string]time.Time
	validTokensLock sync.Mutex

	authFnLock sync.Mutex
	authFn     Authenticator
)

const (
	cookieName = "T17"

	// in seconds
	cookieBaseTTL = 300 // 5 minutes
	cookieTTL     = cookieBaseTTL * time.Second
	cookieRefresh = cookieBaseTTL * 0.9 * time.Second
)

// Authenticator is a function that can be set as the authenticator for the API endpoint. If none is set, all requests will be allowed.
type Authenticator func(s *http.Server, r *http.Request) (grantAccess bool, err error)

// SetAuthenticator sets an authenticator function for the API endpoint. If none is set, all requests will be allowed.
func SetAuthenticator(fn Authenticator) error {
	authFnLock.Lock()
	defer authFnLock.Unlock()

	if authFn == nil {
		authFn = fn
		return nil
	}

	return ErrAuthenticationAlreadySet
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// check existing auth cookie
		c, err := r.Cookie(cookieName)
		if err == nil {
			// get token
			validTokensLock.Lock()
			validUntil, valid := validTokens[c.Value]
			validTokensLock.Unlock()

			// check if token is valid
			if valid && time.Now().Before(validUntil) {
				// maybe refresh cookie
				if time.Now().After(validUntil.Add(-cookieRefresh)) {
					validTokensLock.Lock()
					validTokens[c.Value] = time.Now()
					validTokensLock.Unlock()
				}
				next.ServeHTTP(w, r)
				return
			}
		}

		// get authenticator
		authFnLock.Lock()
		authenticator := authFn
		authFnLock.Unlock()

		// permit if no authenticator set
		if authenticator == nil {
			next.ServeHTTP(w, r)
			return
		}

		// get auth decision
		grantAccess, err := authenticator(server, r)
		if err != nil {
			log.Warningf("api: authenticator failed: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		if !grantAccess {
			log.Warningf("api: denying api access to %s", r.RemoteAddr)
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		// write new cookie
		token, err := random.Bytes(32) // 256 bit
		if err != nil {
			log.Warningf("api: failed to generate random token: %s", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		tokenString := base64.RawURLEncoding.EncodeToString(token)
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    tokenString,
			HttpOnly: true,
			MaxAge:   int(cookieTTL.Seconds()),
		})

		// serve
		log.Tracef("api: granted %s", r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}
