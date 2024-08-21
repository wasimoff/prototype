package pb

// this file is a stub to have `go generate` recompile the protobuf definitions
//go:generate protoc --go_out=paths=source_relative:./ --go_opt=Mmessages.proto=./pb -I=../../ ../../messages.proto

// TODO: example how to valide incoming types of Any messages
//
// func init() {
// 	file_messages_proto_init()
// 	ValidEvents = []string{
// 		string((*GenericEvent)(nil).ProtoReflect().Descriptor().Name()),
// 		string((*ProviderInfo)(nil).ProtoReflect().Descriptor().Name()),
// 		string((*ProviderResources)(nil).ProtoReflect().Descriptor().Name()),
// 	}
// }
//
// var ValidEvents []string
// func IsValidEvent(event proto.Message) bool {
// 	return slices.Contains(ValidEvents, string(event.ProtoReflect().Descriptor().Name()))
// }
