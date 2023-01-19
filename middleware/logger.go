package middleware

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/hysios/mx"
)

func LoggerMiddleware() mx.Middleware {
	return func(h http.Handler) http.Handler {
		return handlers.CombinedLoggingHandler(os.Stdout, h)
	}
}
