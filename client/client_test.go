package client

import (
	"context"
	"testing"

	"github.com/hysios/mx"
	pb "github.com/hysios/mx/_example/gen/proto"
	"github.com/hysios/mx/registry"
	"github.com/hysios/mx/registry/agent"
)

func TestMain(m *testing.M) {
	_ = agent.Default.Register(registry.ServiceDesc{
		ID:        "EchoService",
		Service:   "EchoService",
		TargetURI: "localhost:50051",
		Protocol:  "grpc",
		Address:   "localhost:50051",
		Namespace: "default",
	})

	Registry("EchoService", pb.NewEchoServiceClient)
	m.Run()
}

func TestMake(t *testing.T) {
	type args struct {
		serviceName string
		impl        interface{}
		opts        []MakeOptionFunc
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "make echo service",
			args: args{
				serviceName: "EchoService",
				impl:        new(pb.EchoServiceClient),
				opts:        []MakeOptionFunc{WithMockClient(mx.NopConn())},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Make(tt.args.serviceName, tt.args.impl, tt.args.opts...); (err != nil) != tt.wantErr {
				t.Errorf("Make() error = %v, wantErr %v", err, tt.wantErr)
			}
			echo := *(*pb.EchoServiceClient)(tt.args.impl.(*pb.EchoServiceClient))

			_, err := echo.Echo(context.Background(), &pb.EchoRequest{Say: "hello"})
			if err != nil {
				t.Errorf("Echo() error = %v", err)
			}
		})
	}
}
