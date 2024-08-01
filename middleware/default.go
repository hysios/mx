package middleware

import (
	"github.com/gorilla/handlers"
	"github.com/hysios/mx"
)

var (
	Defaults = []mx.Middleware{
		LoggerMiddleware(),
		handlers.RecoveryHandler(handlers.PrintRecoveryStack(true)),
	}
)
