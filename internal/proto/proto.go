package proto

import (
	"bytes"

	"github.com/yoheimuta/go-protoparser/v4"
	"github.com/yoheimuta/go-protoparser/v4/parser"
)

type Proto = parser.Proto

func Parse(b []byte) (*Proto, error) {
	return protoparser.Parse(
		bytes.NewBuffer(b),
		protoparser.WithDebug(true),
		protoparser.WithPermissive(true),
	)
}
