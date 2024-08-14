package main

import (
	"encoding/base64"
	"flag"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/net/transport"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

const handlerpath = "/messagesock"

// testing broker/net/transport
func main() {

	dx := flag.Bool("dxtest", false, "run dxtest")
	s := flag.String("server", "", "start server listening on this address")
	c := flag.String("client", "", "connect to a server on this address")
	flag.Parse()
	wg := new(sync.WaitGroup)
	defer wg.Wait()

	if *dx {
		dxtests()
	}

	if s != nil && *s != "" {
		wg.Add(1)
		go func() { server(*s); wg.Done() }()
	}

	if c != nil && *c != "" {
		wg.Add(1)
		go func() { client(*c); wg.Done() }()
	}

}

func server(addr string) {

	// register the handler
	http.HandleFunc(handlerpath, func(w http.ResponseWriter, r *http.Request) {
		var wg sync.WaitGroup
		// upgrade the transport, ignoring origin
		t, err := transport.UpgradeToWebSocketTransport(w, r, []string{"*"})
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// wrap a messaging interface around it
		messaging := transport.NewMessagingInterface(t)
		// send rpc requests in a loop on interval
		wg.Add(1)
		go func() {
			defer wg.Done()
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				log.Println("messagesock: sending request")
				response, err := messaging.RequestSync(&pb.Request{Request: &pb.Request_FileListingArgs{
					FileListingArgs: &pb.FileListingArgs{},
				}})
				s, _ := DebugProto(response)
				log.Println(s)
				log.Printf("messagesock: RPC response: err=%#v, res=%#v\n", err, response.GetFileListingResult())
				if err != nil {
					break
				}
			}
			log.Println("--- ERROR --- exited rpc loop:", err)
		}()
		// print incoming events
		wg.Add(1)
		go func() {
			defer wg.Done()
			events := messaging.IncomingEvents()
			for event := range events {
				var ev proto.Message
				switch event.Event.(type) {
				case *pb.Event_ProviderInfo:
					ev = event.GetProviderInfo()
				case *pb.Event_ProviderResources:
					ev = event.GetProviderResources()
				}
				log.Println("event message", event.ProtoReflect().Descriptor().FullName(), ev)
			}
		}()
		wg.Wait()
	})

	// start listening until error
	log.Println("server listening on", addr, handlerpath)
	if err := http.ListenAndServe(addr, http.DefaultServeMux); err != nil {
		log.Println("server error:", err)
	}

}

func client(url string) {
	log.Fatalln("client not implemented yet")
}

// place to debug how messages are constructed and look internally
func dxtests() {

	envelope := &pb.Envelope{
		Sequence: proto.Uint64(33),
		Message: &pb.Envelope_Response{Response: &pb.Response{
			Error: proto.String("nope"),
			Response: &pb.Response_FileListingResult{FileListingResult: &pb.FileListingResult{
				Files: []*pb.FileStat{
					{
						Filename:    proto.String("tsp.wasm"),
						Contenttype: proto.String("application/wasm"),
						Length:      proto.Uint64(64),
						Epoch:       proto.Uint64(uint64(time.Now().Unix())),
						Hash:        make([]byte, 0),
					},
				},
			}},
		}},
	}
	// print the normal envelope
	e, _ := DebugProto(envelope)
	fmt.Printf("\n\n-- Envelope --\n%v\n", e)

	// pack the entire envelope in an Any
	envelopeAny, _ := pb.Any(envelope)
	envelopeAnyValue := base64.StdEncoding.EncodeToString(envelopeAny.Value)
	envelopeAnyJson, _ := protojson.Marshal(envelopeAny)
	fmt.Printf("\n-- Envelope as Any --\nstruct --> %v\nvalue --> %v\njson --> %v\n",
		envelopeAny, envelopeAnyValue, string(envelopeAnyJson))

	// marshal the Any as messagepack, this doesn't work as I hoped
	envelopeAnyMsgp, _ := transport.MarshalMessagepackJson(envelopeAny)
	fmt.Printf("\n-- Envelope as Any as Messagepack --\n%v\n", base64.StdEncoding.EncodeToString(envelopeAnyMsgp))

	// decode the any into wrong type
	res := new(pb.Response)
	err := envelopeAny.UnmarshalTo(res)
	fmt.Printf("\n-- decoded into wrong type --\n%#v, %s\n", res, err)

	// what if you had another message with oneof of all your messages, but only for type-safety?
	// message can be Request | Response | Event here, we don't care; serialize it
	message := envelope.GetMessage()
	//! won't work, message is not a protoreflect.ProtoMessage and we can't get any innards
	fmt.Printf("\n-- marshal a oneof type --\n%#v\n", message)

}

// DebugProto is an internal helper for debugging the structure of serialized messages
func DebugProto(m protoreflect.ProtoMessage) (json string, b64 string) {
	stringified := protojson.Format(m)
	b, err := proto.Marshal(m)
	if err != nil {
		panic("oops, failed marshalling message in DebugProto")
	}
	return stringified, base64.StdEncoding.EncodeToString(b)
}
