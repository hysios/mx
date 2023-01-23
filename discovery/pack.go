package discovery

import (
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

func (p *FileDescriptorPacker) Unpack(src []byte) (protoreflect.FileDescriptor, error) {
	out := protoimpl.DescBuilder{
		GoPackagePath: reflect.TypeOf(struct{}{}).PkgPath(),
		RawDescriptor: src,
	}.Build()

	return out.File, nil
}
