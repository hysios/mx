package session

import (
	"context"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/hysios/mx"
)

type (
	sessionKey struct{}
)

type ContextFunc func(ctx context.Context, sess *sessions.Session) context.Context

// WithSession returns ctx that injects a session store into the context.
func WithSession(ctx context.Context, store sessions.Store) context.Context {
	return context.WithValue(ctx, sessionKey{}, store)
}

// FromContext returns the session store from the context.
func FromContext(ctx context.Context) (sessions.Store, bool) {
	store, ok := ctx.Value(sessionKey{}).(sessions.Store)
	return store, ok
}

func Middleware(sessionName string, store sessions.Store, loadFn ...ContextFunc) mx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var ctx = r.Context()

			if len(loadFn) > 0 {
				sess, err := store.Get(r, sessionName)
				if err == nil {
					load := loadFn[0]
					ctx = load(ctx, sess)
				}
			}

			ctx = WithSession(ctx, store)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
