package delegate

import (
	"context"
	"errors"
	"reflect"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

// type (
// 	ClientCtor          any
// 	ServiceHandleClient any
// 	ServiceHandler      any
// )

var (
	clientConnInterType = reflect.TypeOf((*grpc.ClientConnInterface)(nil)).Elem()
	prtmuxType          = reflect.TypeOf((*runtime.ServeMux)(nil))
	ctxType             = reflect.TypeOf((*context.Context)(nil)).Elem()
	connType            = reflect.TypeOf((*grpc.ClientConn)(nil))
	errType             = reflect.TypeOf((*error)(nil)).Elem()
)

type ClientCtor struct {
	ClientCtor any
}

type ServiceHandlerClient struct {
	HandlerClient any
	ClientCtor    any
}

type ServiceHandler struct {
	Handler any
}

type ServiceHandlerServer struct {
	ServiceImpl   any
	HandlerServer any
}

func (client *ClientCtor) Valid() error {
	var cliVal = reflect.ValueOf(client.ClientCtor)
	if cliVal.Kind() != reflect.Func {
		return errors.New("clientCtor must be a function")
	}

	var cliType = cliVal.Type()
	// input args must have one and it must be implement grpc.ClientConnInterface
	if cliType.NumIn() == 1 && !cliType.In(0).Implements(clientConnInterType) {
		return errors.New("clientCtor must be a function and first args must grpc.ClientConnInterface")
	}

	// output args must have one, and it must be a interface{}
	if cliType.NumOut() == 1 && cliType.Out(0).Kind() != reflect.Interface {
		return errors.New("clientCtor must be a function with one return value")
	}

	return nil
}

func (client *ClientCtor) Call(conn grpc.ClientConnInterface) (interface{}, error) {
	if err := client.Valid(); err != nil {
		return nil, err
	}

	var cliVal = reflect.ValueOf(client.ClientCtor)
	var args []reflect.Value
	if cliVal.Type().NumIn() == 1 {
		args = append(args, reflect.ValueOf(conn))
	}

	var out = cliVal.Call(args)
	if len(out) == 0 {
		return nil, errors.New("clientCtor return value is nil")
	}

	return out[0].Interface(), nil
}

func (handleClient *ServiceHandlerClient) Valid() error {
	var cliVal = reflect.ValueOf(handleClient.HandlerClient)
	if cliVal.Kind() != reflect.Func {
		return errors.New("handleClient must be a function")
	}

	var cliType = cliVal.Type()
	// input args must equal 3
	if cliType.NumIn() != 3 {
		return errors.New("handleClient must be a function with three arguments")
	}

	// input args 1 it must implement context.Context
	if !cliType.In(0).Implements(ctxType) {
		return errors.New("handleClient first argument must be a context.Context")
	}

	// input args 2 it must be a *runtime.ServeMux
	if cliType.In(1) != prtmuxType {
		return errors.New("handleClient second argument must be a *runtime.ServeMux")
	}

	// input args 3 it must be a interface{}
	if cliType.In(2).Kind() != reflect.Interface {
		return errors.New("handleClient third argument must be a interface{}")
	}

	// output args must have one, and it must be a interface{}
	if cliType.NumOut() != 1 && cliType.Out(0).Kind() != reflect.Interface {
		return errors.New("handleClient must be a function with one return value")
	}

	return nil
}

func (handleClient *ServiceHandlerClient) Call(ctx context.Context, mux *runtime.ServeMux, conn grpc.ClientConnInterface) error {
	if err := handleClient.Valid(); err != nil {
		return err
	}

	var cliVal = reflect.ValueOf(handleClient.HandlerClient)
	var args []reflect.Value
	args = append(args, reflect.ValueOf(ctx))
	args = append(args, reflect.ValueOf(mux))
	var cli = ClientCtor{
		ClientCtor: handleClient.ClientCtor,
	}
	clientInter, err := cli.Call(conn)
	if err != nil {
		return err
	}

	inter := reflect.ValueOf(clientInter)
	args = append(args, inter)

	var out = cliVal.Call(args)
	if len(out) == 0 {
		return errors.New("handleClient return value is nil")
	}

	if !out[0].IsNil() {
		return out[0].Interface().(error)
	}

	return nil
}

func (handler *ServiceHandler) Valid() error {
	var cliVal = reflect.ValueOf(handler.Handler)
	if cliVal.Kind() != reflect.Func {
		return errors.New("handler must be a function")
	}

	var cliType = cliVal.Type()
	// input args must equal 3
	if cliType.NumIn() != 3 {
		return errors.New("handler must be a function with three arguments")
	}

	// input args 1 it must implement context.Context
	if !cliType.In(0).Implements(ctxType) {
		return errors.New("handler first argument must be a context.Context")
	}

	// input args 2 it must be a *runtime.ServeMux
	if cliType.In(1) != prtmuxType {
		return errors.New("handler second argument must be a *runtime.ServeMux")
	}
	// input args 3 it must be a *grpc.ClientConn
	if cliType.In(2) != connType {
		return errors.New("handler third argument must be a *grpc.ClientConn")
	}

	// output args must have one, and it must be a error
	if cliType.NumOut() != 1 && cliType.Out(0).Implements(errType) {
		return errors.New("handler must be a function with one return value")
	}

	return nil

}

func (handler *ServiceHandler) Call(ctx context.Context, mux *runtime.ServeMux, conn *grpc.ClientConn) error {
	if err := handler.Valid(); err != nil {
		return err
	}

	var cliVal = reflect.ValueOf(handler.Handler)
	var args []reflect.Value
	args = append(args, reflect.ValueOf(ctx))
	args = append(args, reflect.ValueOf(mux))
	args = append(args, reflect.ValueOf(conn))

	var out = cliVal.Call(args)
	if len(out) == 0 {
		return errors.New("handler return value is nil")
	}

	if !out[0].IsNil() {
		return out[0].Interface().(error)
	}

	return nil
}

func (server *ServiceHandlerServer) Valid() error {
	var (
		cliVal = reflect.ValueOf(server.HandlerServer)
	)

	if cliVal.Kind() != reflect.Func {
		return errors.New("server must be a function")
	}

	var cliType = cliVal.Type()
	// input args must equal 3
	if cliType.NumIn() != 3 {
		return errors.New("server must be a function with three arguments")
	}

	// input args 1 it must implement context.Context
	if !cliType.In(0).Implements(ctxType) {
		return errors.New("server first argument must be a context.Context")
	}

	// input args 2 it must be a *runtime.ServeMux
	if cliType.In(1) != prtmuxType {
		return errors.New("server second argument must be a *runtime.ServeMux")
	}

	// input args 3 it must be a interface{}
	if cliType.In(2).Kind() != reflect.Interface {
		return errors.New("server third argument must be a interface{}")
	}

	// output args must have one, and it must be a error
	if cliType.NumOut() != 1 && cliType.Out(0).Implements(errType) {
		return errors.New("server must be a function with one return value")
	}

	return nil
}

func (server *ServiceHandlerServer) Call(ctx context.Context, mux *runtime.ServeMux, serverImpl any) error {
	if err := server.Valid(); err != nil {
		return err
	}

	var cliVal = reflect.ValueOf(server.HandlerServer)
	var args []reflect.Value
	args = append(args, reflect.ValueOf(ctx))
	args = append(args, reflect.ValueOf(mux))

	// get server.HanderServer third argument type
	var serverType = cliVal.Type().In(2)
	// convert serverImpl to server.HanderServer third argument type
	var serverImplVal = reflect.ValueOf(serverImpl)
	if serverImplVal.Type().ConvertibleTo(serverType) {
		serverImplVal = serverImplVal.Convert(serverType)
	} else {
		return errors.New("serverImpl is not convertible to server.HanderServer third argument type")
	}

	args = append(args, serverImplVal)

	var out = cliVal.Call(args)
	if len(out) == 0 {
		return errors.New("server return value is nil")
	}

	if !out[0].IsNil() {
		return out[0].Interface().(error)
	}

	return nil
}
