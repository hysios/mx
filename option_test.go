package mx

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	pb "github.com/hysios/mx/example/gen/proto"
	"github.com/hysios/mx/internal/delegate"
	"google.golang.org/grpc"
)

func TestOptServiceClient(t *testing.T) {
	type args struct {
		clientCtor   ClientCtor
		handleClient ServiceHandleClient
	}
	tests := []struct {
		name string
		args args
		want error
	}{
		{
			name: "test",
			args: args{
				clientCtor:   pb.NewHelloServiceClient,
				handleClient: pb.RegisterHelloServiceHandlerClient,
			},
			want: nil,
		},
		{
			name: "failed",
			args: args{
				clientCtor:   func(a int) int { return a },
				handleClient: func() {},
			},
			want: errors.New("clientCtor must be a function and first args must grpc.ClientConnInterface"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts RegisterOption
			if got := WithServiceClient(tt.args.clientCtor, tt.args.handleClient)(&opts); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("OptServiceClient() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_serviceHandleClientProxy_Call(t *testing.T) {
	type fields struct {
		handleClient ServiceHandleClient
		clientCtor   ClientCtor
	}
	type args struct {
		ctx  context.Context
		mux  *runtime.ServeMux
		conn grpc.ClientConnInterface
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "test",
			fields: fields{
				handleClient: pb.RegisterHelloServiceHandlerClient,
				clientCtor:   pb.NewHelloServiceClient,
			},
			args: args{
				ctx:  context.Background(),
				mux:  runtime.NewServeMux(),
				conn: &grpc.ClientConn{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := delegate.ServiceHandleClientProxy{
				HandleClient: tt.fields.handleClient,
				ClientCtor:   tt.fields.clientCtor,
			}
			if err := client.Call(tt.args.ctx, tt.args.mux, tt.args.conn); (err != nil) != tt.wantErr {
				t.Errorf("serviceHandleClientProxy.Call() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
