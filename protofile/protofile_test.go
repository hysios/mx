package protofile

import (
	"testing"

	"github.com/kr/pretty"
	"github.com/tj/assert"
)

const helloProto = `syntax = "proto3";

package hello;

option go_package = "github.com/hysios/mx/_example/gen/proto;pb";

import "google/api/annotations.proto";
import "google/protobuf/any.proto";
import "protoc-gen-openapiv2/options/annotations.proto";

option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_swagger) = {
    info: { version : "1.0" }
    external_docs : {
      url:  "https://github.com/hysios/mx/example/proto"
      description: "mx framework api demo"
    }
    schemes:[HTTP, HTTPS];
};

service HelloService {
    rpc Hello(HelloRequest) returns (HelloResponse) {
        option (google.api.http) = {
            // Route to this method from GET requests to /api/v1/path
            get : "/api/hello"
        
        };
        option (grpc.gateway.protoc_gen_openapiv2.options.openapiv2_operation) = {
            summary : "Hello Method"
            tags : "HelloService"
        };
    }
}

message HelloRequest {
    string say = 1;
}

message HelloResponse {
    string message = 1;
}
`

func TestParse(t *testing.T) {
	protofile, err := Parse([]byte(helloProto))
	assert.NoError(t, err)
	assert.NotNil(t, protofile)
	protofile.proto = nil

	t.Logf("protofile % #v", pretty.Formatter(protofile))
	// 写出结果比较，验证 protofile 是否正确
	assert.Equal(t, "hello", protofile.Pkg.Name)
	// assert.Equal(t, "github.com/hysios/mx/_example/gen/proto", protofile.PkgPath)
	assert.Equal(t, "HelloRequest", protofile.Messages[0].Name)
	assert.Equal(t, "HelloResponse", protofile.Messages[1].Name)
	assert.Equal(t, "HelloService", protofile.Services[0].Name)
	assert.Equal(t, "Hello", protofile.Services[0].RPCs[0].Name)
	// HelloRequest Fields
	assert.Equal(t, "say", protofile.Messages[0].Fields[0].Name)
	assert.Equal(t, "string", protofile.Messages[0].Fields[0].Type)
	assert.Equal(t, "1", protofile.Messages[0].Fields[0].Number)
	// HelloResponse Fields
	assert.Equal(t, "message", protofile.Messages[1].Fields[0].Name)
	assert.Equal(t, "string", protofile.Messages[1].Fields[0].Type)
	assert.Equal(t, "1", protofile.Messages[1].Fields[0].Number)
}
