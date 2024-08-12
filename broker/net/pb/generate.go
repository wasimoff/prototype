package pb

// this file is a stub to have `go generate` recompile the protobuf definitions
//go:generate protoc --go_out=paths=source_relative:./ --go_opt=Mmessages.proto=./pb -I=../../ ../../messages.proto
