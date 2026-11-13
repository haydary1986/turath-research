package middleware

import "net/http"

// TrackPresence reports every authenticated request's user to seen, which must
// return at once (see presence.Tracker): it never delays or fails the request.
// Must be used after RequireAuth.
func TrackPresence(seen func(userID string)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if user := GetUser(r.Context()); user != nil && seen != nil {
				seen(user.UserID)
			}
			next.ServeHTTP(w, r)
		})
	}
}
