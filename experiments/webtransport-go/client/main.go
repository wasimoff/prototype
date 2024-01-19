package main

import (
	"context"
	"log"

	"github.com/quic-go/webtransport-go"
)

const TRANSPORT = "https://localhost:4443/transport"

func main() {

	var d webtransport.Dialer
	log.Printf("dial %s", TRANSPORT)
	r, conn, err := d.Dial(context.Background(), TRANSPORT, nil)
	if err != nil {
		log.Fatalf("ERR: couldn't dial: %s", err)
	}
	log.Printf("response: %v", r)

	log.Println("open stream")
	stream, err := conn.OpenStream()
	if err != nil {
		log.Fatalf("ERR: couldn't open stream: %s", err)
	}
	defer stream.Close()

	log.Println("write a few bytes to stream")
	_, err = stream.Write([]byte("Hello, World!"))
	if err != nil {
		log.Fatalf("ERR: couldn't write to stream: %s", err)
	}

	log.Println("read bytes from stream")
	buf := make([]byte, 64)
	n, err := stream.Read(buf)
	if err != nil {
		log.Fatalf("ERR: couldn't read from stream: %s", err)
	}

	log.Printf("echoed back %d bytes: \"%s\"\n", n, buf[:n])

}
