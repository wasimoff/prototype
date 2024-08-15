package main

import (
	"encoding/base64"
	"fmt"
	"wasimoff/broker/net/pb"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// these strings both encode basically the same message; reserialize with prototext, if needed
const (
	// using nested oneof in Envelope
	textOne = `sequence:33 response:{error:"nope" fileListingResult:{files:{filename:"tsp.wasm" contenttype:"application/wasm" length:64 epoch:1723710503 hash:""}}}`
	byteOne = "CCFaMAoEbm9wZWIoCiYKCHRzcC53YXNtEhBhcHBsaWNhdGlvbi93YXNtGEAiACin+Pa1Bg=="
	// using an Any in Envelope
	textAny = `sequence:33 type:Response error:"nope" payload:{[wasimoff/FileListingResult]:{files:{filename:"tsp.wasm" contenttype:"application/wasm" length:64 epoch:1723710503 hash:""}}}`
	byteAny = "CCEQAhoEbm9wZSJGChp3YXNpbW9mZi9GaWxlTGlzdGluZ1Jlc3VsdBIoCiYKCHRzcC53YXNtEhBhcHBsaWNhdGlvbi93YXNtGEAiACin+Pa1Bg=="
)

var (
	bufOne []byte
	bufAny []byte
)

func init() {
	bufOne, _ = base64.StdEncoding.DecodeString(byteOne)
	bufAny, _ = base64.StdEncoding.DecodeString(byteAny)
}

/**
README
------
This file serves as a playground to explore the memory allocation behaviour of
unmarshalling my Envelope messages in Go. The results really make me appreciate
the two-message design of net/rpc a lot more because it lets you precisely
control the location of your allocation and you can pass in a stack pointer
into the request method to be filled upon completion.

	> The nested oneof messages make typing pretty strict, which is nice. But it
	also means that you need to Unmarshal the entire buffer into a single
	Envelope message, allocating new pointers to the correct subtype as needed.

	> This means that you cannot pass in a pointer to memory on the stack to use
	because when the receiver() loop unmarshals the message, it doesn't know a)
	what type of message this is or b) who (if it's a response) the inner part
	should be routed to (from pending calls). So really the best you can do is
	pass in a res **Response and then set res = &envelope.GetResponse(), or just
	return the pointer directly, since you're allocating a new one anyway.

	> An "easy" optimization is to reuse an outer envelope struct in the receiver
	loop and always Unmarshal into it. The implicit Reset() will allocate a new
	pointer for the Response anyway but now you don't carry around the padding
	from all those discarded Envelopes in your memory; you can't free them as long
	as the Response inside is still in use. This makes for a difference of about
	60 bytes between BenchmarkUnmarshalNew and BenchmarkUnmarshalIntoEnvelope.

	> In net/rpc, you unpack the header, always. All you need to route the message
	and decide what type/pointer the body should be is in there. Only then you read
	the body and directly Unmarshal it into the memory preallocated by the caller;
	this will usually reside in their stack and thus be faster overall. If I adopt
	this two-message approach for myself, I might put the Event into the header
	and omit the second message for those, though that would make reading harder.

	> Another quasi-rescue while maintaining my single-message approach is an Any
	message, since you need to Unmarshal that to the correct message as a second
	step. At that point you will already have all the enclosing envelope fields
	and can anypb.UnmarshalTo() the correct preallocated message directly. You
	still need a double-encode of the inner Any message so there's still a lot of
	allocation going on but results indicate that it's faster nonetheless.

	> At the same time you should get rid of the oneof's because each layer
	introduces another two pointers in Go: one for the quasi-union type and one
	for the concrete message type.

Highlights:

BenchmarkOneofNew-16                    	 1545615	       778.6 ns/op	     416 B/op	      16 allocs/op
BenchmarkAnyNewKnown-16                 	 1669023	       718.2 ns/op	     512 B/op	      17 allocs/op
--> Allocate whatever is needed on the heap and pass the result back up.

BenchmarkOneofIntoEnvelope-16           	 1612502	       745.2 ns/op	     352 B/op	      15 allocs/op
BenchmarkAnyIntoEnvelopeAndKnown2-16    	 1855912	       643.5 ns/op	     368 B/op	      15 allocs/op
--> Envelope (and inner Any type) are passed in as pointers and Protobuf unmarshals
    *into* them. Oneof still needs a lot of allocs because the struct is nested and
    Any probably has double encoding going on because the struct itself is flatter.

BenchmarkAnyNewOuter-16                 	 3690267	       321.3 ns/op	     272 B/op	       8 allocs/op
--> Not really useful because only the outer Envelope is unpacked and not the payload.
    But yeah, half the time is spent on unpacking the Any payload in all other runs.

BenchmarkOneofMergeEnvelope-16          	 2215387	       536.9 ns/op	     215 B/op	       8 allocs/op
--> The complete nested Envelope struct is already in the right form and reused.
    By far the fastest but unfortunately unusable in a single-message RPC where you
		*don't* know the incoming structure beforehand.


Full benchmark output:

go test -bench . -benchmem
goos: linux
goarch: amd64
pkg: wasimoff/broker/bench/unmarshal-allocations
cpu: AMD Ryzen 7 PRO 5850U with Radeon Graphics
BenchmarkOneofNew-16                    	 1545615	       778.6 ns/op	     416 B/op	      16 allocs/op
BenchmarkAnyNewOuter-16                 	 3690267	       321.3 ns/op	     272 B/op	       8 allocs/op
BenchmarkAnyNew-16                      	 1498089	       797.8 ns/op	     512 B/op	      17 allocs/op
BenchmarkAnyNewKnown-16                 	 1669023	       718.2 ns/op	     512 B/op	      17 allocs/op
BenchmarkOneofIntoEnvelope-16           	 1612502	       745.2 ns/op	     352 B/op	      15 allocs/op
BenchmarkAnyIntoEnvelopeAndNew-16       	 1582948	       757.4 ns/op	     432 B/op	      16 allocs/op
BenchmarkAnyIntoEnvelopeAndKnown-16     	 1864393	       640.4 ns/op	     368 B/op	      15 allocs/op
BenchmarkAnyIntoEnvelopeAndKnown2-16    	 1855912	       643.5 ns/op	     368 B/op	      15 allocs/op
BenchmarkOneofMergeEnvelopeReset-16     	 1667136	       713.0 ns/op	     352 B/op	      15 allocs/op
BenchmarkOneofMergeEnvelope-16          	 2253222	       537.3 ns/op	     225 B/op	       8 allocs/op
BenchmarkOneofMergeResponse-16          	  983617	      1218 ns/op	     557 B/op	      21 allocs/op
BenchmarkOneofMergeResponseInto-16      	 2063744	       545.3 ns/op	     226 B/op	       9 allocs/op
PASS
ok  	wasimoff/broker/bench/unmarshal-allocations	21.747s


*/

func main() {

	// check if the pointers returned by functions are different
	fmt.Println("UnmarshalNew:")
	fmt.Printf("r1: %p\n", must1(UnmarshalOneofNew()))
	fmt.Printf("r2: %p\n", must1(UnmarshalOneofNew()))
	// --> response pointers differ

	// into the same envelope
	fmt.Println("UnmarshalIntoEnvelope:")
	e := new(pb.Envelope)
	UnmarshalOneofIntoEnvelope(e)
	fmt.Printf("r1: %p\n", e.GetResponse())
	UnmarshalOneofIntoEnvelope(e)
	fmt.Printf("r2: %p\n", e.GetResponse())
	// --> response pointers differ

	// merge into prepared struct reuses the response
	fmt.Println("UnmarshalMergeEnvelope (prepared):")
	e = &pb.Envelope{Message: &pb.Envelope_Response{
		Response: &pb.Response{},
	}}
	fmt.Printf("%p (fresh)\n", e.GetResponse())
	UnmarshalOneofMergeEnvelope(e)
	fmt.Printf("%p (merged)\n", e.GetResponse())
	// --> pointer is unchanged, saved a few allocs
	e.Reset()
	UnmarshalOneofMergeEnvelope(e)
	fmt.Printf("%p (reset & merged)\n", e.GetResponse())
	// --> reset removes the inner pointer, so new alloc

	// merge into incorrectly prepared struct allocs new response
	fmt.Println("UnmarshalMergeEnvelope (incorrect):")
	e = &pb.Envelope{Message: &pb.Envelope_Request{
		Request: &pb.Request{},
	}}
	fmt.Printf("req[0]: %p (fresh)\n", e.GetRequest())
	fmt.Printf("res[0]: %p (nil)\n", e.GetResponse())
	UnmarshalOneofMergeEnvelope(e)
	fmt.Printf("req[1]: %p (merged, nil)\n", e.GetRequest())
	fmt.Printf("res[1]: %p (merged, alloc)\n", e.GetResponse())

}

// ignore errors, return value
func must1[T any](t T, _ error) T {
	return t
}

// panic on error
func panics0(e error) {
	if e != nil {
		panic("oops")
	}
}

// panic on error, discard value
func panics1(_ any, e error) {
	if e != nil {
		panic("oops")
	}
}

//go:noinline
func UnmarshalOneofNew() (*pb.Response, error) {
	envelope := new(pb.Envelope)
	err := proto.Unmarshal(bufOne, envelope)
	return envelope.GetResponse(), err
}

//go:noinline
func UnmarshalAnyNew() (*pb.FileListingResult, error) {
	envelope := new(pb.EnvelopeAny)
	err := proto.Unmarshal(bufAny, envelope)
	if err != nil {
		return nil, err
	}
	inner, err := envelope.Payload.UnmarshalNew()
	payload := inner.(*pb.FileListingResult)
	return payload, err
}

//go:noinline
func UnmarshalAnyNewOuter() (*pb.EnvelopeAny, error) {
	envelope := new(pb.EnvelopeAny)
	err := proto.Unmarshal(bufAny, envelope)
	if err != nil {
		return nil, err
	}
	return envelope, nil
}

//go:noinline
func UnmarshalAnyNewKnown() (*pb.FileListingResult, error) {
	envelope := new(pb.EnvelopeAny)
	err := proto.Unmarshal(bufAny, envelope)
	if err != nil {
		return nil, err
	}
	payload := new(pb.FileListingResult)
	err = envelope.Payload.UnmarshalTo(payload)
	return payload, err
}

//go:noinline
func UnmarshalOneofNewClone() (*pb.Response, error) {
	// This makes no sense whatsoever compared to ..New()
	envelope := new(pb.Envelope)
	err := proto.Unmarshal(bufOne, envelope)
	return proto.Clone(envelope.GetResponse()).(*pb.Response), err
}

//go:noinline
func UnmarshalOneofIntoEnvelope(envelope *pb.Envelope) error {
	return proto.Unmarshal(bufOne, envelope)
}

//go:noinline
func UnmarshalAnyIntoEnvelopeAndNew(envelope *pb.EnvelopeAny) (message protoreflect.ProtoMessage, err error) {
	// to make this comparison fair, we need to unpack the inner as well
	if err = proto.Unmarshal(bufAny, envelope); err != nil {
		return nil, err
	}
	message, err = envelope.Payload.UnmarshalNew()
	return
}

//go:noinline
func UnmarshalAnyIntoEnvelopeAndKnown(envelope *pb.EnvelopeAny, result *pb.FileListingResult) error {
	// to make this comparison fair, we need to unpack the inner as well
	if err := proto.Unmarshal(bufAny, envelope); err != nil {
		return err
	}
	err := envelope.Payload.UnmarshalTo(result)
	return err
}

var tmpEnvelopeAny *pb.EnvelopeAny

//go:noinline
func UnmarshalAnyIntoEnvelopeAndKnown2(result *pb.FileListingResult) error {
	// this shouldn't make a difference to the previous func, just where the envelope is kept
	if tmpEnvelopeAny == nil {
		tmpEnvelopeAny = new(pb.EnvelopeAny)
	}
	if err := proto.Unmarshal(bufAny, tmpEnvelopeAny); err != nil {
		return err
	}
	err := tmpEnvelopeAny.Payload.UnmarshalTo(result)
	return err
}

//go:noinline
func UnmarshalOneofMergeEnvelope(envelope *pb.Envelope) error {
	return proto.UnmarshalOptions{Merge: true}.Unmarshal(bufOne, envelope)
}

var tmpEnvelopeOne *pb.Envelope

//go:noinline
func UnmarshalOneofMergeResponse(response *pb.Response) error {
	if tmpEnvelopeOne == nil {
		tmpEnvelopeOne = new(pb.Envelope)
	}
	if err := proto.Unmarshal(bufOne, tmpEnvelopeOne); err != nil {
		return err
	}
	//! Response is only a single pointer to the inner concrete type, so this is
	//! effectively ~ a Clone, hence the bad performance
	proto.Merge(response, tmpEnvelopeOne.GetResponse())
	return nil
}

//go:noinline
func UnmarshalOneofMergeResponseInto(response *pb.Response) error {
	if tmpEnvelopeOne == nil {
		tmpEnvelopeOne = new(pb.Envelope)
	}
	tmpEnvelopeOne.Message = &pb.Envelope_Response{Response: response}
	if err := UnmarshalOneofMergeEnvelope(tmpEnvelopeOne); err != nil {
		return err
	}
	return nil
}
