// generate: protoc --go_out=. --go_opt=paths=source_relative note.proto

// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.30.0
// 	protoc        v3.21.12
// source: note.proto

package proto

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Note struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id      uint32 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
	// marks the end of messages, use the rest of the stream as wasm stream
	Wasm *RunWasm `protobuf:"bytes,4,opt,name=wasm,proto3" json:"wasm,omitempty"`
}

func (x *Note) Reset() {
	*x = Note{}
	if protoimpl.UnsafeEnabled {
		mi := &file_note_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Note) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Note) ProtoMessage() {}

func (x *Note) ProtoReflect() protoreflect.Message {
	mi := &file_note_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Note.ProtoReflect.Descriptor instead.
func (*Note) Descriptor() ([]byte, []int) {
	return file_note_proto_rawDescGZIP(), []int{0}
}

func (x *Note) GetId() uint32 {
	if x != nil {
		return x.Id
	}
	return 0
}

func (x *Note) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

func (x *Note) GetWasm() *RunWasm {
	if x != nil {
		return x.Wasm
	}
	return nil
}

// things to build an inline Response from remainder of the stream
type Response struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status  uint32            `protobuf:"varint,1,opt,name=status,proto3" json:"status,omitempty"`
	Text    string            `protobuf:"bytes,2,opt,name=text,proto3" json:"text,omitempty"`
	Headers map[string]string `protobuf:"bytes,3,rep,name=headers,proto3" json:"headers,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *Response) Reset() {
	*x = Response{}
	if protoimpl.UnsafeEnabled {
		mi := &file_note_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *Response) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*Response) ProtoMessage() {}

func (x *Response) ProtoReflect() protoreflect.Message {
	mi := &file_note_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use Response.ProtoReflect.Descriptor instead.
func (*Response) Descriptor() ([]byte, []int) {
	return file_note_proto_rawDescGZIP(), []int{1}
}

func (x *Response) GetStatus() uint32 {
	if x != nil {
		return x.Status
	}
	return 0
}

func (x *Response) GetText() string {
	if x != nil {
		return x.Text
	}
	return ""
}

func (x *Response) GetHeaders() map[string]string {
	if x != nil {
		return x.Headers
	}
	return nil
}

// arguments for the WASM instantiation
type RunWasm struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Response *Response         `protobuf:"bytes,1,opt,name=response,proto3" json:"response,omitempty"`
	Args     []string          `protobuf:"bytes,2,rep,name=args,proto3" json:"args,omitempty"`
	Envs     map[string]string `protobuf:"bytes,3,rep,name=envs,proto3" json:"envs,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *RunWasm) Reset() {
	*x = RunWasm{}
	if protoimpl.UnsafeEnabled {
		mi := &file_note_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *RunWasm) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*RunWasm) ProtoMessage() {}

func (x *RunWasm) ProtoReflect() protoreflect.Message {
	mi := &file_note_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use RunWasm.ProtoReflect.Descriptor instead.
func (*RunWasm) Descriptor() ([]byte, []int) {
	return file_note_proto_rawDescGZIP(), []int{2}
}

func (x *RunWasm) GetResponse() *Response {
	if x != nil {
		return x.Response
	}
	return nil
}

func (x *RunWasm) GetArgs() []string {
	if x != nil {
		return x.Args
	}
	return nil
}

func (x *RunWasm) GetEnvs() map[string]string {
	if x != nil {
		return x.Envs
	}
	return nil
}

var File_note_proto protoreflect.FileDescriptor

var file_note_proto_rawDesc = []byte{
	0x0a, 0x0a, 0x6e, 0x6f, 0x74, 0x65, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x12, 0x05, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x22, 0x54, 0x0a, 0x04, 0x4e, 0x6f, 0x74, 0x65, 0x12, 0x0e, 0x0a, 0x02, 0x69,
	0x64, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x02, 0x69, 0x64, 0x12, 0x18, 0x0a, 0x07, 0x6d,
	0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x07, 0x6d, 0x65,
	0x73, 0x73, 0x61, 0x67, 0x65, 0x12, 0x22, 0x0a, 0x04, 0x77, 0x61, 0x73, 0x6d, 0x18, 0x04, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x75, 0x6e, 0x57,
	0x61, 0x73, 0x6d, 0x52, 0x04, 0x77, 0x61, 0x73, 0x6d, 0x22, 0xaa, 0x01, 0x0a, 0x08, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x0d, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x12, 0x12,
	0x0a, 0x04, 0x74, 0x65, 0x78, 0x74, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x04, 0x74, 0x65,
	0x78, 0x74, 0x12, 0x36, 0x0a, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x18, 0x03, 0x20,
	0x03, 0x28, 0x0b, 0x32, 0x1c, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x2e, 0x48, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72,
	0x79, 0x52, 0x07, 0x68, 0x65, 0x61, 0x64, 0x65, 0x72, 0x73, 0x1a, 0x3a, 0x0a, 0x0c, 0x48, 0x65,
	0x61, 0x64, 0x65, 0x72, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10, 0x0a, 0x03, 0x6b, 0x65,
	0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79, 0x12, 0x14, 0x0a, 0x05,
	0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x76, 0x61, 0x6c,
	0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x22, 0xb1, 0x01, 0x0a, 0x07, 0x52, 0x75, 0x6e, 0x57, 0x61,
	0x73, 0x6d, 0x12, 0x2b, 0x0a, 0x08, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x18, 0x01,
	0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x65, 0x73,
	0x70, 0x6f, 0x6e, 0x73, 0x65, 0x52, 0x08, 0x72, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12,
	0x12, 0x0a, 0x04, 0x61, 0x72, 0x67, 0x73, 0x18, 0x02, 0x20, 0x03, 0x28, 0x09, 0x52, 0x04, 0x61,
	0x72, 0x67, 0x73, 0x12, 0x2c, 0x0a, 0x04, 0x65, 0x6e, 0x76, 0x73, 0x18, 0x03, 0x20, 0x03, 0x28,
	0x0b, 0x32, 0x18, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x2e, 0x52, 0x75, 0x6e, 0x57, 0x61, 0x73,
	0x6d, 0x2e, 0x45, 0x6e, 0x76, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x52, 0x04, 0x65, 0x6e, 0x76,
	0x73, 0x1a, 0x37, 0x0a, 0x09, 0x45, 0x6e, 0x76, 0x73, 0x45, 0x6e, 0x74, 0x72, 0x79, 0x12, 0x10,
	0x0a, 0x03, 0x6b, 0x65, 0x79, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x03, 0x6b, 0x65, 0x79,
	0x12, 0x14, 0x0a, 0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x09, 0x52,
	0x05, 0x76, 0x61, 0x6c, 0x75, 0x65, 0x3a, 0x02, 0x38, 0x01, 0x42, 0x27, 0x5a, 0x25, 0x77, 0x61,
	0x73, 0x6d, 0x6f, 0x66, 0x66, 0x2f, 0x65, 0x78, 0x70, 0x65, 0x72, 0x69, 0x6d, 0x65, 0x6e, 0x74,
	0x2f, 0x77, 0x65, 0x62, 0x74, 0x72, 0x61, 0x6e, 0x73, 0x70, 0x6f, 0x72, 0x74, 0x2f, 0x70, 0x72,
	0x6f, 0x74, 0x6f, 0x62, 0x06, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_note_proto_rawDescOnce sync.Once
	file_note_proto_rawDescData = file_note_proto_rawDesc
)

func file_note_proto_rawDescGZIP() []byte {
	file_note_proto_rawDescOnce.Do(func() {
		file_note_proto_rawDescData = protoimpl.X.CompressGZIP(file_note_proto_rawDescData)
	})
	return file_note_proto_rawDescData
}

var file_note_proto_msgTypes = make([]protoimpl.MessageInfo, 5)
var file_note_proto_goTypes = []interface{}{
	(*Note)(nil),     // 0: proto.Note
	(*Response)(nil), // 1: proto.Response
	(*RunWasm)(nil),  // 2: proto.RunWasm
	nil,              // 3: proto.Response.HeadersEntry
	nil,              // 4: proto.RunWasm.EnvsEntry
}
var file_note_proto_depIdxs = []int32{
	2, // 0: proto.Note.wasm:type_name -> proto.RunWasm
	3, // 1: proto.Response.headers:type_name -> proto.Response.HeadersEntry
	1, // 2: proto.RunWasm.response:type_name -> proto.Response
	4, // 3: proto.RunWasm.envs:type_name -> proto.RunWasm.EnvsEntry
	4, // [4:4] is the sub-list for method output_type
	4, // [4:4] is the sub-list for method input_type
	4, // [4:4] is the sub-list for extension type_name
	4, // [4:4] is the sub-list for extension extendee
	0, // [0:4] is the sub-list for field type_name
}

func init() { file_note_proto_init() }
func file_note_proto_init() {
	if File_note_proto != nil {
		return
	}
	if !protoimpl.UnsafeEnabled {
		file_note_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Note); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_note_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*Response); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_note_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*RunWasm); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_note_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   5,
			NumExtensions: 0,
			NumServices:   0,
		},
		GoTypes:           file_note_proto_goTypes,
		DependencyIndexes: file_note_proto_depIdxs,
		MessageInfos:      file_note_proto_msgTypes,
	}.Build()
	File_note_proto = out.File
	file_note_proto_rawDesc = nil
	file_note_proto_goTypes = nil
	file_note_proto_depIdxs = nil
}