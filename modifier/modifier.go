package modifier

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/protobuf/proto"
)

type ResponseModifier func(context.Context, http.ResponseWriter, proto.Message) error

type MetadataWriter func(http.ResponseWriter, runtime.ServerMetadata, proto.Message) error
