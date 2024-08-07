package main

import (
	ctx "context"
	"io"
	"log"
	"net/http"
	"os"
	"wasimoff/experiment/webtransport/proto"

	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	pb "google.golang.org/protobuf/encoding/protodelim"
)

const TLS_CRT = "localhost.crt"
const TLS_KEY = "localhost.key"
const LISTEN = ":4443"
const HOSTNAME = "localhost"

func main() {

	// run static file server in background
	go httpIndex()

	// new server muxer
	mux := http.NewServeMux()

	// create UDP listening server on muxer
	server := webtransport.Server{
		H3: http3.Server{
			EnableDatagrams: true,
			Addr:            LISTEN,
			Handler:         mux,
		},
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	defer server.Close()

	// webtransport endpoint
	mux.HandleFunc("/transport", func(w http.ResponseWriter, r *http.Request) {

		// connection upgrade
		log.Println("WebTransport connection attempt")
		conn, err := server.Upgrade(w, r)
		if err != nil {
			log.Printf("connection upgrade failed: %s", err)
			w.WriteHeader(500)
			return
		}

		// write something to unidirectional stream
		uni, err := conn.OpenUniStream()
		if err != nil {
			log.Printf("failed to open unidirectional stream: %s", err)
		} else {

			// write simple messages
			notes := []string{"One", "Two", "Three", "Hello, Stream!"}
			for i, note := range notes {
				n, err := pb.MarshalTo(uni, &proto.Note{Id: uint32(i + 1), Message: note})
				if err != nil {
					log.Printf("failed to marshal protobuf to wire: %s", err)
				} else {
					log.Printf("wrote %d bytes to unidirectional stream", n)
				}
			}

			// write a final DoResponse message
			resp := &proto.Note{
				Wasm: &proto.RunWasm{
					Response: &proto.Response{
						Status: 200,
						Text:   "Run Wasmoff",
						Headers: map[string]string{
							"content-type": "application/wasm",
							"server":       "wasmoff streamer",
						},
					},
					Args: []string{"exe.wasm", "-from", "WebTransport_Stream"},
					Envs: map[string]string{
						"PROJECT": "wasmoff",
						"version": "very alpha",
					},
				},
			}
			n, err := pb.MarshalTo(uni, resp)
			if err != nil {
				log.Printf("failed to marshal final message: %s", err)
			} else {
				log.Printf("wrote %d bytes to stream: %#v", n, resp)
				// copy an actual wasm file
				wasm, err := os.Open("../../wasi-hello-world/web/exe.wasm")
				if err != nil {
					panic(err) // TODO ...
				}
				defer wasm.Close()
				k, err := io.Copy(uni, wasm)
				if err != nil {
					log.Printf("failed to wasm to stream: %s", err)
				} else {
					log.Printf("wrote %d bytes of WASM to stream", k)
				}
			}

		}
		uni.Close()

		// application logic
		for {
			log.Printf("accepting streams from %s", conn.RemoteAddr())
			stream, err := conn.AcceptStream(ctx.Background())
			if err != nil {
				break
			}
			defer stream.Close()
			log.Println("running echo handler for stream")
			if _, err = io.Copy(stream, stream); err != nil {
				log.Printf("echo handler failed to copy: %s", err)
				break
			}
			log.Println("successfully copied stream")
			stream.Write(nil)

			// only handle a single stream
			break

		}

	})

	log.Printf("webtransport listening on https://%s%s/transport (UDP)", HOSTNAME, LISTEN)
	if err := server.ListenAndServeTLS(TLS_CRT, TLS_KEY); err != nil {
		log.Fatalf("oops: %s", err)
	}

}

func httpIndex() {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("./"))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// log.Printf("REQUEST: %#v", r) //! DEBUG
		fs.ServeHTTP(w, r)
	}))
	log.Printf("static files listening on https://%s%s/", HOSTNAME, LISTEN)
	if err := http.ListenAndServeTLS(LISTEN, TLS_CRT, TLS_KEY, mux); err != nil {
		log.Fatalf("static files server failed: %s", err)
	}
}
