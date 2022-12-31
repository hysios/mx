package mx

import "google.golang.org/grpc"

type ConnString string

func (conn ConnString) Open() (*grpc.ClientConn, error) {
	return grpc.Dial(string(conn), grpc.WithInsecure())
}
