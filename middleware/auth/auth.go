package auth

import (
	"context"
	"net/http"

	"github.com/satriahrh/letter-block/jwt"
)

var userCtxKey = contextKey{"user"}

type contextKey struct {
	name string
}

func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authorization := r.Header.Get("Authorization")

		// validate jwt token
		tokenStr := authorization[7:]
		user, err := jwt.ParseToken(tokenStr)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusForbidden)
			return
		}

		// put it in context
		ctx := context.WithValue(r.Context(), "user", user)

		// and call the next with our new context
		r = r.WithContext(ctx)
		next.ServeHTTP(w, r)
	})
}

// ForContext finds the user from the context. REQUIRES Middleware to have run.
func ForContext(ctx context.Context) *jwt.User {
	raw, _ := ctx.Value(userCtxKey).(*jwt.User)
	return raw
}
