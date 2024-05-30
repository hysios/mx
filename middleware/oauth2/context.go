package oauth2

import (
	"context"

	"github.com/go-oauth2/oauth2/v4/server"
)

type oauth2Server struct{}

// WithOAuth2 returns ctx that injects a oauth2 server into the context.
func WithOAuth2(ctx context.Context, server *server.Server) context.Context {
	return context.WithValue(ctx, oauth2Server{}, server)
}

// FromContext returns the oauth2 server from the context.
func FromContext(ctx context.Context) (*server.Server, bool) {
	server, ok := ctx.Value(oauth2Server{}).(*server.Server)
	return server, ok
}
