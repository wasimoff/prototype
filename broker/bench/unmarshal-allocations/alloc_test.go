package main

import (
	"testing"
	"wasimoff/broker/net/pb"
)

func BenchmarkOneofNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		panics1(UnmarshalOneofNew())
	}
}

func BenchmarkAnyNewOuter(b *testing.B) {
	for i := 0; i < b.N; i++ {
		panics1(UnmarshalAnyNewOuter())
	}
}

func BenchmarkAnyNew(b *testing.B) {
	for i := 0; i < b.N; i++ {
		panics1(UnmarshalAnyNew())
	}
}

func BenchmarkAnyNewKnown(b *testing.B) {
	for i := 0; i < b.N; i++ {
		panics1(UnmarshalAnyNewKnown())
	}
}

// func BenchmarkOneofNewClone(b *testing.B) {
// 	for i := 0; i < b.N; i++ {
// 		panics1(UnmarshalOneofNewClone())
// 	}
// }

func BenchmarkOneofIntoEnvelope(b *testing.B) {
	envelope := new(pb.Envelope)
	for i := 0; i < b.N; i++ {
		panics0(UnmarshalOneofIntoEnvelope(envelope))
	}
}

func BenchmarkAnyIntoEnvelopeAndNew(b *testing.B) {
	envelope := new(pb.EnvelopeAny)
	for i := 0; i < b.N; i++ {
		panics1(UnmarshalAnyIntoEnvelopeAndNew(envelope))
	}
}

func BenchmarkAnyIntoEnvelopeAndKnown(b *testing.B) {
	envelope := new(pb.EnvelopeAny)
	result := new(pb.FileListingResult)
	for i := 0; i < b.N; i++ {
		panics0(UnmarshalAnyIntoEnvelopeAndKnown(envelope, result))
	}
}

func BenchmarkAnyIntoEnvelopeAndKnown2(b *testing.B) {
	result := new(pb.FileListingResult)
	for i := 0; i < b.N; i++ {
		panics0(UnmarshalAnyIntoEnvelopeAndKnown2(result))
	}
}

func BenchmarkOneofMergeEnvelopeReset(b *testing.B) {
	envelope := new(pb.Envelope)
	for i := 0; i < b.N; i++ {
		envelope.Reset()
		panics0(UnmarshalOneofMergeEnvelope(envelope))
	}
}

func BenchmarkOneofMergeEnvelope(b *testing.B) {
	envelope := new(pb.Envelope)
	for i := 0; i < b.N; i++ {
		// keep previous values
		panics0(UnmarshalOneofMergeEnvelope(envelope))
	}
}

func BenchmarkOneofMergeResponse(b *testing.B) {
	response := new(pb.Response)
	for i := 0; i < b.N; i++ {
		// response.Reset()
		panics0(UnmarshalOneofMergeResponse(response))
	}
}

func BenchmarkOneofMergeResponseInto(b *testing.B) {
	response := new(pb.Response)
	for i := 0; i < b.N; i++ {
		// response.Reset()
		panics0(UnmarshalOneofMergeResponseInto(response))
	}
}
