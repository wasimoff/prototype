package pb

// this file is a stub to have `go generate` recompile the protobuf definitions
//go:generate protoc --go_out=paths=source_relative:./ --go_opt=Mmessages.proto=./pb -I=../../ ../../messages.proto

//?-- example how to valide incoming types of Any messages
// var ValidEvents []string
// func init() {
// 	file_messages_proto_init()
// 	ValidEvents = []string{
// 		string((*GenericEvent)(nil).ProtoReflect().Descriptor().Name()),
// 		string((*ProviderInfo)(nil).ProtoReflect().Descriptor().Name()),
// 		string((*ProviderResources)(nil).ProtoReflect().Descriptor().Name()),
// 	}
// }
// func IsValidEvent(s string) bool {
// 	return slices.Contains(ValidEvents, s)
// }
// IsValidEvent(string(event.ProtoReflect().Descriptor().Name()))
// IsValidEvent(strings.Split(event.TypeUrl, "/")[-1])

//? -- example of a type union in an interface constraint for generics
// type EventMessages interface {
// 	*GenericEvent | *ProviderInfo | *ProviderResources
// }
// func foo[Ev EventMessages](m Ev) ([]byte, error) {
// 	return proto.Marshal(proto.Message(m)) // need to convert type
// }
