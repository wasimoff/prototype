package pb

import (
	"google.golang.org/protobuf/proto"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	"google.golang.org/protobuf/types/known/anypb"
)

// URL prefix used in google.protobuf.Any messages packed with `Any`
const AnyTypeURLPrefix = "wasimoff/"

// helper to marshal a google.protobuf.Any message using custom type URL prefix
func Any(src proto.Message) (*anypb.Any, error) {
	if src == nil {
		// use the same error as anypb.MarshalFrom
		return nil, protoimpl.X.NewError("invalid nil source message")
	}
	buf, err := proto.Marshal(src)
	if err != nil {
		return nil, err
	}
	return &anypb.Any{
		TypeUrl: AnyTypeURLPrefix + string(src.ProtoReflect().Descriptor().FullName()),
		Value:   buf,
	}, nil
}
