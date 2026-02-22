package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/manas-solves/realworld-backend/internal/data"
)

// recoverPanic recovers from a panic, logs the details, and sends a 500 internal server error response.
func (app *application) recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.Header().Set("Connection", "close")
				app.serverErrorResponse(w, r, fmt.Errorf("%s", err))
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// authenticate checks the Authorization header and verifies the JWT.
// If the JWT is valid, it retrieves the user details based on the user ID and sets the user details in the request context.
// Unlike before, this middleware now rejects invalid tokens instead of silently treating them as anonymous.
// Only missing tokens result in anonymous access.
func (app *application) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")

		// No authorization header - proceed as anonymous user
		if header == "" {
			r = app.contextSetUser(r, data.AnonymousUser)
			next.ServeHTTP(w, r)
			return
		}

		// Authorization header present but malformed - reject explicitly
		if !strings.HasPrefix(header, "Token ") {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		tokenString := strings.TrimPrefix(header, "Token ")

		// Verify the token - reject if invalid or expired
		claims, err := app.jwtMaker.VerifyToken(tokenString)
		if err != nil {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		// GetByID now handles caching automatically
		user, err := app.modelStore.Users.GetByID(claims.UserID)
		if err != nil {
			// User not found - token references non-existent user (deleted account)
			if errors.Is(err, data.ErrRecordNotFound) {
				app.invalidAuthenticationTokenResponse(w, r)
				return
			}
			// Database error
			app.serverErrorResponse(w, r, err)
			return
		}

		// Set the token (not cached, as it's request-specific)
		user.Token = tokenString
		r = app.contextSetUser(r, user)
		next.ServeHTTP(w, r)
	})
}

// requireAuthenticatedUser checks if the user is authenticated.
// If not, it sends a 401 unauthorized response.
func (app *application) requireAuthenticatedUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := app.contextGetUser(r)
		if user.IsAnonymous() {
			app.invalidAuthenticationTokenResponse(w, r)
			return
		}

		next.ServeHTTP(w, r)
	})
}
