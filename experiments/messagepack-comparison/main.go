package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"

	msg "github.com/vmihailenco/msgpack/v5"
	pb "google.golang.org/protobuf/proto"
)

// generate the serialization methods for tinylib/msgp
//go:generate msgp

type MyNote struct {
	Id      uint32 `msg:"id"  msgpack:"id"`
	Message string `msg:"msg" msgpack:"msg"`
	RunWasm bool   `msg:"run" msgpack:"run"`
	Binary  []byte `msg:"bin" msgpack:"bin"`
}

func main() {

	// create a dummy note
	note := &MyNote{
		Id:      30,
		Message: "Hello",
		RunWasm: false,
		Binary:  []byte("What the Hack."),
	}
	fmt.Println("Note", note)

	// marshal it with standard JSON
	js, err := json.Marshal(note)
	if err != nil {
		panic(err)
	}
	fmt.Println("JSON               ", string(js))

	// encode it with the uptrace messagepack library
	var mpbufUptrace bytes.Buffer
	enc := msg.NewEncoder(&mpbufUptrace)
	enc.UseCompactInts(true)
	err = enc.Encode(note)
	if err != nil {
		panic(err)
	}
	fmt.Println("MessagePack Uptrace", base64.StdEncoding.EncodeToString(mpbufUptrace.Bytes()))

	// encode it with the generated tinylib methods
	mpbufTinylib, err := note.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	fmt.Println("MessagePack Tinylib", base64.StdEncoding.EncodeToString(mpbufTinylib))

	// try to decode uptrace message with tinylib
	new := &MyNote{}
	_, err = new.UnmarshalMsg(mpbufUptrace.Bytes())
	if err != nil {
		panic(err)
	}

	// encode an equivalent message with protobuf
	pbuf, err := pb.Marshal(&Note{
		Id:      note.Id,
		Message: note.Message,
		Wasm:    note.RunWasm,
		Binary:  note.Binary,
	})
	if err != nil {
		panic(err)
	}
	fmt.Println("Protobuf           ", base64.StdEncoding.EncodeToString(pbuf))

}
