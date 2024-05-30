package response

import (
	"context"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/hysios/mx/modifier"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func ResponseTypeModifier(writer modifier.MetadataWriter) modifier.ResponseModifier {
	return func(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
		md, ok := runtime.ServerMetadataFromContext(ctx)
		if !ok {
			return nil
		}

		if vals := md.HeaderMD.Get(KeyResponseType); len(vals) > 0 {
			if err := writer(w, md, resp); err != nil {
				return err
			}
		}

		return nil
	}
}

func SetResponseType(ctx context.Context, respType string) error {
	return grpc.SetHeader(ctx, metadata.Pairs(KeyResponseType, respType))
}

// SetResponseContent 设置响应内容
func SetResponseContent(ctx context.Context, content string, typ ...string) error {
	if len(typ) > 0 {
		if err := SetResponseType(ctx, typ[0]); err != nil {
			return err
		}
	}
	return grpc.SetHeader(ctx, metadata.Pairs(KeyResponseContent, content))
}
