localhost.crt localhost.key:
	bash ./gencerts.sh

proto/note.pb.go: proto/note.proto
	cd proto/ && protoc --go_out=./ --go_opt=paths=source_relative note.proto