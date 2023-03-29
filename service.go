package mx

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"
	"github.com/hysios/mx/httprule"
	"github.com/hysios/mx/internal/delegate"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/dynamicpb"
)

type Service interface {
	ServiceName() string
	Register(ctx context.Context, gw *Gateway) error
	// Invoke(ctx context.Context, method string, args, reply interface{}) error
}

type DynamicService interface {
	Service

	AddConn(serviceId string, conn *grpc.ClientConn) error
	RemoveConn(serviceId string) error
}

type ServerVersion interface {
	Version() string
}

type nopConn struct {
}

func (*nopConn) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	return nil
}

func (*nopConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}

func NopConn() grpc.ClientConnInterface {
	return &nopConn{}
}

func NewClientService(name string, conn *grpc.ClientConn, registerHandler interface{}) (Service, error) {
	var handler = delegate.ServiceHandler{
		Handler: registerHandler,
	}

	if err := handler.Valid(); err != nil {
		return nil, err
	}

	return &clientService{
		name:    name,
		conn:    conn,
		handler: handler,
	}, nil
}

type clientService struct {
	name    string
	conn    *grpc.ClientConn
	handler delegate.ServiceHandler
}

func (c *clientService) ServiceName() string {
	return c.name
}

func (c *clientService) Register(ctx context.Context, gw *Gateway) error {
	return c.handler.Call(ctx, gw.gwmux, c.conn)
}

func NewLocalService(name string, serviceImpl interface{}, registerHandler interface{}) (Service, error) {
	var handler = delegate.ServiceHandlerServer{
		ServiceImpl:   serviceImpl,
		HandlerServer: registerHandler,
	}

	if err := handler.Valid(); err != nil {
		return nil, err
	}

	return &localService{
		name:        name,
		serviceImpl: serviceImpl,
		handler:     handler,
	}, nil
}

type localService struct {
	name        string
	serviceImpl interface{}
	handler     delegate.ServiceHandlerServer
}

// ServiceName returns the name of the service.
func (l *localService) ServiceName() string {
	return l.name
}

func (l *localService) Register(ctx context.Context, gw *Gateway) error {
	return l.handler.Call(ctx, gw.gwmux, l.serviceImpl)
}

type dynamicService struct {
	name    string
	connMap map[string]*grpc.ClientConn
	conns   Muxer
	handler delegate.ServiceHandlerClient
}

func NewDynamicService(name string, registerHandler interface{}, clientCtor interface{}) (DynamicService, error) {
	var handler = delegate.ServiceHandlerClient{
		HandlerClient: registerHandler,
		ClientCtor:    clientCtor,
	}

	if err := handler.Valid(); err != nil {
		return nil, err
	}

	return &dynamicService{
		name:    name,
		connMap: make(map[string]*grpc.ClientConn),
		handler: handler,
	}, nil
}

func (d *dynamicService) ServiceName() string {
	return d.name
}

func (d *dynamicService) Register(ctx context.Context, gw *Gateway) error {
	return d.handler.Call(ctx, gw.gwmux, &d.conns)
}

func (d *dynamicService) AddConn(serviceId string, conn *grpc.ClientConn) error {
	d.connMap[serviceId] = conn
	d.conns.Add(serviceId, conn)
	return nil
}

func (d *dynamicService) RemoveConn(serviceId string) error {
	delete(d.connMap, serviceId)
	d.conns.Remove(serviceId)
	return nil
}

type descriptorBuilderService struct {
	name           string
	filedescriptor protoreflect.FileDescriptor
	logger         *zap.Logger
	connMap        map[string]*grpc.ClientConn
	conns          Muxer
	handlers       map[string]httpMethod
	// handles map[string]
}

type httpMethod struct {
	Method  string `json:"method"`
	Pattern runtime.Pattern
	Handler runtime.HandlerFunc
}

// NewDescriptorBuilderService creates a new instance of the descriptorBuilderService type.
// name is the name of the service.
// filedescriptor is the protobuf file descriptor that contains the service.
func NewDescriptorBuilderService(name string, filedescriptor protoreflect.FileDescriptor) *descriptorBuilderService {
	return &descriptorBuilderService{
		name:           name,
		filedescriptor: filedescriptor,
		connMap:        make(map[string]*grpc.ClientConn),
		handlers:       make(map[string]httpMethod),
	}
}

func (d *descriptorBuilderService) SetLogger(logger *zap.Logger) {
	d.logger = logger
}

func (d *descriptorBuilderService) ServiceName() string {
	return d.name
}

func (d *descriptorBuilderService) Register(ctx context.Context, gw *Gateway) error {
	err := d.Build(ctx, gw.gwmux)
	if err != nil {
		return err
	}

	for _, handler := range d.handlers {
		d.logger.Info("registering http handler", zap.String("method", handler.Method), zap.String("pattern", handler.Pattern.String()))
		gw.gwmux.Handle(handler.Method, handler.Pattern, handler.Handler)
	}

	return nil
}

func (d *descriptorBuilderService) Build(ctx context.Context, mux *runtime.ServeMux) error {
	// build runtime.ServerMux handler from filedescriptor
	// read services
	for i := 0; i < d.filedescriptor.Services().Len(); i++ {
		service := d.filedescriptor.Services().Get(i)
		for j := 0; j < service.Methods().Len(); j++ {
			var (
				method = service.Methods().Get(j)
			)
			methHandler, err := d.buildHttpHandler(mux, method)
			if err != nil {
				return err
			}
			d.handlers[string(method.FullName())] = methHandler
		}
	}

	return nil
}

func (d *descriptorBuilderService) AddConn(serviceId string, conn *grpc.ClientConn) error {
	d.connMap[serviceId] = conn
	d.conns.Add(serviceId, conn)
	return nil
}

func (d *descriptorBuilderService) RemoveConn(serviceId string) error {
	delete(d.connMap, serviceId)
	d.conns.Remove(serviceId)
	return nil
}

type methodOptions struct {
	GoogleAPIHTTP GoogleAPIHTTP                                          `json:"[google.api.http]"`
	GrpcGateway   GrpcGatewayProtocGenOpenapiv2OptionsOpenapiv2Operation `json:"[grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation]"`
}

// GoogleAPIHTTP is a wrapper for the Google API HTTP client.
type GoogleAPIHTTP struct {
	Get    string `json:"get"`
	Post   string `json:"post"`
	Put    string `json:"put"`
	Patch  string `json:"patch"`
	Delete string `json:"delete"`
	Body   string `json:"body"`
}

// GrpcGatewayProtocGenOpenapiv2OptionsOpenapiv2Operation describes an operation
// produced by the OpenAPIv2 generator, for more information about OpenAPI
// v2.0, see:
type GrpcGatewayProtocGenOpenapiv2OptionsOpenapiv2Operation struct {
	Tags    []string `json:"tags"`
	Summary string   `json:"summary"`
}

func (apiHttp *GoogleAPIHTTP) Method() string {
	switch {
	case apiHttp.Get != "":
		return http.MethodGet
	case apiHttp.Post != "":
		return http.MethodPost
	case apiHttp.Put != "":
		return http.MethodPut
	case apiHttp.Patch != "":
		return http.MethodPatch
	case apiHttp.Delete != "":
		return http.MethodDelete
	default:
		return http.MethodGet
	}
}

// Path returns the path of the request.
func (apiHttp *GoogleAPIHTTP) Path() string {
	switch {
	case apiHttp.Get != "":
		return apiHttp.Get
	case apiHttp.Post != "":
		return apiHttp.Post
	case apiHttp.Put != "":
		return apiHttp.Put
	case apiHttp.Patch != "":
		return apiHttp.Patch
	case apiHttp.Delete != "":
		return apiHttp.Delete
	default:
		return ""
	}
}

// unmarshalOptions unmarshals the options proto message into a methodOptions
// struct.
func (d *descriptorBuilderService) unmarshalOptions(options protoreflect.ProtoMessage) (*methodOptions, error) {
	// Marshal the options into JSON.
	b, err := protojson.Marshal(options)
	if err != nil {
		return nil, err
	}

	// Unmarshal the JSON into a methodOptions struct.
	var opts methodOptions
	if err := json.Unmarshal(b, &opts); err != nil {
		return nil, err
	}

	return &opts, nil
}

func (d *descriptorBuilderService) buildHttpHandler(mux *runtime.ServeMux, method protoreflect.MethodDescriptor) (httpMethod, error) {
	// var methodName = string(method.FullName())
	var (
		parent      = method.Parent()
		serviceName = string(parent.FullName())
		methodName  = string(method.Name())
		fullname    = string("/" + serviceName + "/" + methodName)
	)

	options, err := d.unmarshalOptions(method.Options())
	if err != nil {
		return httpMethod{}, err
	}

	compile, err := httprule.Parse(options.GoogleAPIHTTP.Path())
	if err != nil {
		return httpMethod{}, err
	}

	tmpl := compile.Compile()

	pattern, err := runtime.NewPattern(tmpl.Version, tmpl.OpCodes, tmpl.Pool, tmpl.Verb)
	if err != nil {
		return httpMethod{}, err
	}

	d.logger.Info("method options", zap.String("method", fullname), zap.Any("options", options))
	switch options.GoogleAPIHTTP.Method() {
	case http.MethodGet:
		return httpMethod{
			Method:  "GET",
			Pattern: pattern,
			Handler: func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
				ctx, cancel := context.WithCancel(req.Context())
				defer cancel()
				inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
				var err error
				var annotatedContext context.Context
				annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, fullname, runtime.WithHTTPPathPattern(options.GoogleAPIHTTP.Path()))
				if err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				input := dynamicpb.NewMessage(method.Input())
				output := dynamicpb.NewMessage(method.Output())

				if err := d.applyParams(input, method.Input(), tmpl, pathParams); err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				resp, md, err := d.request_GetMethod(annotatedContext, inboundMarshaler, fullname, input, output, req, pathParams)
				annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
				if err != nil {
					runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
					return
				}

				runtime.ForwardResponseMessage(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
			},
		}, nil
	case http.MethodPost:
		return httpMethod{
			Method:  "POST",
			Pattern: pattern,
			Handler: func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
				ctx, cancel := context.WithCancel(req.Context())
				defer cancel()
				inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
				var err error
				var annotatedContext context.Context
				annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, fullname, runtime.WithHTTPPathPattern(options.GoogleAPIHTTP.Path()))
				if err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				input := dynamicpb.NewMessage(method.Input())
				output := dynamicpb.NewMessage(method.Output())

				if err := d.applyParams(input, method.Input(), tmpl, pathParams); err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				resp, md, err := d.request_PostMethod(annotatedContext, inboundMarshaler, fullname, input, output, req, pathParams)
				annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
				if err != nil {
					runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
					return
				}

				runtime.ForwardResponseMessage(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
			},
		}, nil
	case http.MethodPut:
		return httpMethod{
			Method:  "PUT",
			Pattern: pattern,
			Handler: func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
				ctx, cancel := context.WithCancel(req.Context())
				defer cancel()
				inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
				var err error
				var annotatedContext context.Context
				annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, fullname, runtime.WithHTTPPathPattern(options.GoogleAPIHTTP.Path()))
				if err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				input := dynamicpb.NewMessage(method.Input())
				output := dynamicpb.NewMessage(method.Output())

				if err := d.applyParams(input, method.Input(), tmpl, pathParams); err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				resp, md, err := d.request_PostMethod(annotatedContext, inboundMarshaler, fullname, input, output, req, pathParams)
				annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
				if err != nil {
					runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
					return
				}

				runtime.ForwardResponseMessage(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
			},
		}, nil
	case http.MethodPatch:
		return httpMethod{
			Method:  "PATCH",
			Pattern: pattern,
			Handler: func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
				ctx, cancel := context.WithCancel(req.Context())
				defer cancel()
				inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
				var err error
				var annotatedContext context.Context
				annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, fullname, runtime.WithHTTPPathPattern(options.GoogleAPIHTTP.Path()))
				if err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				input := dynamicpb.NewMessage(method.Input())
				output := dynamicpb.NewMessage(method.Output())

				if err := d.applyParams(input, method.Input(), tmpl, pathParams); err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				resp, md, err := d.request_PostMethod(annotatedContext, inboundMarshaler, fullname, input, output, req, pathParams)
				annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
				if err != nil {
					runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
					return
				}

				runtime.ForwardResponseMessage(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
			},
		}, nil
	case http.MethodDelete:
		return httpMethod{
			Method:  "DELETE",
			Pattern: pattern,
			Handler: func(w http.ResponseWriter, req *http.Request, pathParams map[string]string) {
				ctx, cancel := context.WithCancel(req.Context())
				defer cancel()
				inboundMarshaler, outboundMarshaler := runtime.MarshalerForRequest(mux, req)
				var err error
				var annotatedContext context.Context
				annotatedContext, err = runtime.AnnotateContext(ctx, mux, req, fullname, runtime.WithHTTPPathPattern(options.GoogleAPIHTTP.Path()))
				if err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				input := dynamicpb.NewMessage(method.Input())
				output := dynamicpb.NewMessage(method.Output())

				if err := d.applyParams(input, method.Input(), tmpl, pathParams); err != nil {
					runtime.HTTPError(ctx, mux, outboundMarshaler, w, req, err)
					return
				}

				resp, md, err := d.request_PostMethod(annotatedContext, inboundMarshaler, fullname, input, output, req, pathParams)
				annotatedContext = runtime.NewServerMetadataContext(annotatedContext, md)
				if err != nil {
					runtime.HTTPError(annotatedContext, mux, outboundMarshaler, w, req, err)
					return
				}

				runtime.ForwardResponseMessage(annotatedContext, mux, outboundMarshaler, w, req, resp, mux.GetForwardResponseOptions()...)
			},
		}, nil
	default:
		panic("unknown method")
	}
}

func (d *descriptorBuilderService) request_GetMethod(ctx context.Context, marshaler runtime.Marshaler, method string, input, output *dynamicpb.Message, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	if err := req.ParseForm(); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	// Create a new filter
	filters := &utilities.DoubleArray{Encoding: map[string]int{}, Base: []int(nil), Check: []int(nil)}

	if err := runtime.PopulateQueryParameters(input, req.Form, filters); err != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	err := d.conns.Invoke(ctx, method, input, output, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return output, metadata, err
}

func (d *descriptorBuilderService) request_PostMethod(ctx context.Context, marshaler runtime.Marshaler, method string, input, output *dynamicpb.Message, req *http.Request, pathParams map[string]string) (proto.Message, runtime.ServerMetadata, error) {
	var metadata runtime.ServerMetadata

	newReader, berr := utilities.IOReaderFactory(req.Body)
	if berr != nil {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", berr)
	}
	if err := marshaler.NewDecoder(newReader()).Decode(input); err != nil && err != io.EOF {
		return nil, metadata, status.Errorf(codes.InvalidArgument, "%v", err)
	}

	err := d.conns.Invoke(ctx, method, input, output, grpc.Header(&metadata.HeaderMD), grpc.Trailer(&metadata.TrailerMD))
	return output, metadata, err
}

func (d *descriptorBuilderService) applyParams(msg *dynamicpb.Message, descriptor protoreflect.MessageDescriptor, pattern httprule.Template, pathParams map[string]string) error {
	var err error
	for _, v := range pattern.Fields {
		if pathParams[v] == "" {
			return status.Errorf(codes.InvalidArgument, "missing parameter %s", v)
		}
	}

	// check msg field type and set value
	for _, v := range pattern.Fields {
		field := descriptor.Fields().ByName(protoreflect.Name(v))
		if field == nil {
			return status.Errorf(codes.InvalidArgument, "missing field %s", v)
		}

		switch field.Kind() {
		case protoreflect.StringKind:
			msg.Set(field, protoreflect.ValueOf(pathParams[v]))
		case protoreflect.Int32Kind:
			var i int32
			i, err = runtime.Int32(pathParams[v])
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid parameter %s", v)
			}
			msg.Set(field, protoreflect.ValueOf(i))
		case protoreflect.Int64Kind:
			var i int64
			i, err = runtime.Int64(pathParams[v])
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid parameter %s", v)
			}
			msg.Set(field, protoreflect.ValueOf(i))
		case protoreflect.Uint32Kind:
			var i uint32
			i, err = runtime.Uint32(pathParams[v])
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid parameter %s", v)
			}
			msg.Set(field, protoreflect.ValueOf(i))
		case protoreflect.Uint64Kind:
			var i uint64
			i, err = runtime.Uint64(pathParams[v])
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid parameter %s", v)
			}
			msg.Set(field, protoreflect.ValueOf(i))
		case protoreflect.BoolKind:
			var b bool
			b, err = runtime.Bool(pathParams[v])
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid parameter %s", v)
			}
			msg.Set(field, protoreflect.ValueOf(b))
		case protoreflect.FloatKind:
			var f float32
			f, err = runtime.Float32(pathParams[v])
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid parameter %s", v)
			}
			msg.Set(field, protoreflect.ValueOf(f))
		case protoreflect.DoubleKind:
			var f float64
			f, err = runtime.Float64(pathParams[v])
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "invalid parameter %s", v)
			}
			msg.Set(field, protoreflect.ValueOf(f))
		default:
			return status.Errorf(codes.InvalidArgument, "invalid field type %s", field.Kind())
		}
	}

	return nil
}
