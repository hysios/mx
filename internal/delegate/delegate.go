package delegate

import (
	"context"
	"errors"
	"reflect"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
)

type (
	ClientCtor          any
	ServiceHandleClient any
)

var (
	clientConnInterType = reflect.TypeOf((*grpc.ClientConnInterface)(nil)).Elem()
	prtmuxType          = reflect.TypeOf((*runtime.ServeMux)(nil))
	ctxType             = reflect.TypeOf((*context.Context)(nil)).Elem()
)

func ClientValid(client ClientCtor) error {
	var cliVal = reflect.ValueOf(client)
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

func ServiceHandleValid(handleClient ServiceHandleClient) error {
	var cliVal = reflect.ValueOf(handleClient)
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

type ClientProxy struct {
	ClientCtor ClientCtor
}

type ServiceHandleClientProxy struct {
	HandleClient ServiceHandleClient
	ClientCtor   ClientCtor
}

func (client *ClientProxy) Call(conn grpc.ClientConnInterface) (interface{}, error) {
	if err := ClientValid(client.ClientCtor); err != nil {
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

func (handle *ServiceHandleClientProxy) Call(ctx context.Context, mux *runtime.ServeMux, conn grpc.ClientConnInterface) error {
	if err := ServiceHandleValid(handle.HandleClient); err != nil {
		return err
	}

	var cliVal = reflect.ValueOf(handle.HandleClient)
	var args []reflect.Value
	args = append(args, reflect.ValueOf(ctx))
	args = append(args, reflect.ValueOf(mux))
	var cli = ClientProxy{
		ClientCtor: handle.ClientCtor,
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
