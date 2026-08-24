package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey guarantees that the keys will be unique within the entire application
type contextKey string

const (
	UserIDKey contextKey = "user_id"
	RoleKey contextKey = "role"
)

/** WHY: func accepts a secret key for token verification and returns 
	a function that takes the next handler (`next`) and returns a new handler. 
	This allows to "chain" authorization checks onto any of the application's handlers
*/
// Auth — middleware for validating the JWT token and saving user data to the context
func Auth(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Retrieve the Authorization header from the HTTP request
			header := r.Header.Get("Authorization")
			if header == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				// NOTE: If the word "Bearer" is missing or the token isn't provided, the client receives an error
				http.Error(w, `{"error":"invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			// 2. JWT token parsing and validation
			token, err := jwt.Parse(parts[1], func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			// 3. Payload extraction
			claims, ok := token.Claims.(jwt.MapClaims)
			// NOTE: If the token is valid, the code extracts the data we embedded in it during login
			if !ok {
				http.Error(w, `{"error":"invalid claims"}`, http.StatusUnauthorized)
				return
			}

			userID, _ := claims["user_id"].(string)
			role, _ := claims["role"].(string)

			// 4. Passing data further along the chain
			ctx := context.WithValue(r.Context(), UserIDKey, userID) // WHY: Take the current request context and pack the user ID and role into it
			ctx = context.WithValue(ctx, RoleKey, role) // WHY: Create a copy of the request, but with the context already embedded

			next.ServeHTTP(w, r.WithContext(ctx)) // Launches main handler
		})
	}
}