package discovery

import (
	"errors"
	"fmt"
	"reflect"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/runtime/protoimpl"
)

type FileDescriptorPacker struct {
}

func (p *FileDescriptorPacker) Pack(desc protoreflect.FileDescriptor) ([]byte, error) {
	descProto := protodesc.ToFileDescriptorProto(desc)
	return proto.MarshalOptions{AllowPartial: true, Deterministic: true}.Marshal(descProto)
}

func (p *FileDescriptorPacker) Unpack(src []byte) (desc protoreflect.FileDescriptor, err error) {
	defer func() {
		if _err := recover(); _err != nil {
			switch r := _err.(type) {
			case error:
				err = r
			case string:
				err = errors.New(r)
			default:
				err = fmt.Errorf("%v", r)
			}
		}
	}()

	out := protoimpl.DescBuilder{
		GoPackagePath: reflect.TypeOf(struct{}{}).PkgPath(),
		RawDescriptor: src,
	}.Build()

	desc = out.File
	return
}
