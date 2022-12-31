package tests

import (
	"testing"

	pb "github.com/hysios/mx/example/gen/proto"
	"github.com/tj/assert"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func Test(t *testing.T) {
	var req = pb.HelloRequest{
		Say: "hello",
	}
	b, err := prototext.Marshal(&req)
	assert.NoError(t, err)
	assert.Equal(t, "say:\"hello\"", string(b))
}

func TestMarshal(t *testing.T) {
	var req = pb.HelloRequest{
		Say: "hello",
	}
	b, err := proto.Marshal(&req)
	assert.NoError(t, err)
	assert.Equal(t, "say:\"hello\"", string(b))
}
