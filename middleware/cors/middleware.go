package cors

import (
	"net/http"

	"github.com/hysios/mx"
	"github.com/rs/cors"
)

func Default() mx.Middleware {
	c := cors.New(cors.Options{
		// AllowedOrigins: domains,
		AllowOriginFunc: func(origin string) bool {
			return true
		},
		MaxAge: 3600,
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowedHeaders: []string{
			"Access-Control-Allow-Origin",
			"Access-Control-Allow-Credentials",
			"Cookie",
			"Content-Type",
			"Set-Cookie",
			// "*",
		},
		AllowCredentials: true,
		// Debug:            true,
	})

	return c.Handler
}
