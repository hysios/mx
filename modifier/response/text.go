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

const (
	KeyResponseType    = "x-response-type"
	KeyResponseContent = "x-response-content"
	KeyTextExtractKey  = "x-text-extract-key"
)

type Extractor interface{}

func TextExtractor(key string, writer modifier.MetadataWriter) modifier.ResponseModifier {
	return func(ctx context.Context, w http.ResponseWriter, resp proto.Message) error {
		md, ok := runtime.ServerMetadataFromContext(ctx)
		if !ok {
			return nil
		}

		if vals := md.HeaderMD.Get(key); len(vals) > 0 {
			if err := writer(w, md, resp); err != nil {
				return err
			}
		}

		return nil
	}
}

func SetTextExtractKey(ctx context.Context, key string) error {
	return grpc.SetHeader(ctx, metadata.Pairs(KeyTextExtractKey, key))
}
