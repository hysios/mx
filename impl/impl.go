package impl

import (
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/hysios/mx"
	_ "github.com/hysios/mx/discovery/consul"
)

func NewGateway() *mx.Gateway {
	var gw = &mx.Gateway{}

	gw.Use(handlers.RecoveryHandler())
	gw.Use(func(h http.Handler) http.Handler {
		return handlers.CombinedLoggingHandler(os.Stdout, h)
	})

	gw.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello world"))
	})

	return gw
}

func ListenServer(addr string, gw *mx.Gateway) error {
	return gw.Serve(addr)
}
