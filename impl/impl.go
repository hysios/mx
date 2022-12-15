package impl

import "github.com/hysios/mx"

func NewGateway() *mx.Gateway {
	return &mx.Gateway{}
}

func ListenServer(addr string, gw *mx.Gateway) error {
	return gw.Serve(addr)
}
