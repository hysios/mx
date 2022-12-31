package middleware

import "github.com/hysios/mx"

var (
	Defaults = []mx.Middleware{
		LoggerMiddleware(),
		RecoveryMiddleware(),
	}
)
