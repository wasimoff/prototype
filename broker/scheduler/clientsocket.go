package scheduler

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/net/transport"
	"wasimoff/broker/provider"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

// TODO text
// ClientSocketHandler returns a http.HandlerFunc to be used on a route that shall serve
// as an endpoint for Clients to connect to. This particular handler uses WebSocket
// transport with either Protobuf or JSON encoding, negotiated using subprotocol strings.
func ClientSocketHandler(store *provider.ProviderStore) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		addr := transport.ProxiedAddr(r)

		// upgrade the transport
		// using wildcard in Allowed-Origins because Client can be anywhere
		wst, err := transport.UpgradeToWebSocketTransport(w, r, []string{"*"})
		if err != nil {
			log.Printf("[%s] New Client socket: upgrade failed: %s", addr, err)
			return
		}
		messenger := transport.NewMessengerInterface(wst)
		log.Printf("[%s] New Client socket", addr)

		// all tasks on this socket are counted as one "job"
		job := fmt.Sprintf("ws/%05d", jobSequence.Add(1))
		requestSequence := uint64(0)

		// channel for finished requests
		done := make(chan *provider.AsyncWasiTask, 32)

		defer log.Printf("[%s] Client socket closed", addr)
		for {
			select {

			case <-r.Context().Done():
				return
			case <-messenger.Closing():
				return

			// print any received events
			case event, ok := <-messenger.Events():
				if !ok {
					return
				}
				log.Printf("{client %s} %s", addr, prototext.Format(event))

			// dispatch received requests
			case request, ok := <-messenger.Requests():
				if !ok {
					return
				}
				switch rq := request.Request.(type) {

				case *pb.ClientUploadRequest:
					request.Respond(r.Context(), &pb.ClientUploadResponse{}, proto.String("upload over socket not implemented yet"))
					continue

				case *pb.WasiTaskArgs:
					requestSequence++

					// resolve files
					errs := []error{}
					errs = append(errs, store.Storage.ResolvePbFile(rq.Binary))
					errs = append(errs, store.Storage.ResolvePbFile(rq.Rootfs))
					if err := errors.Join(errs...); err != nil {
						request.Respond(r.Context(), &pb.WasiTaskResult{}, proto.String(err.Error()))
						continue // next request
					}

					// assemble the task for internal dispatcher queue
					resp := pb.ExecuteWasiResponse{}
					wreq := pb.ExecuteWasiRequest{
						// common task metadata with index counter
						Info: &pb.TaskMetadata{
							JobID:  &job,
							Index:  proto.Uint64(requestSequence),
							Client: &addr,
						},
						Task: rq,
					}
					taskctx := context.WithValue(r.Context(), ctxkeyRequest{}, request)
					taskQueue <- provider.NewAsyncWasiTask(taskctx, &wreq, &resp, done)
					// log.Printf("Task submit: %s :: %#v\n", wreq.Info.TaskID(), wreq.Task.Args)
					continue

				default:
					request.Respond(r.Context(), &pb.GenericEvent{}, proto.String("request type not supported"))
					continue

				}

				// respond with finished results
			case task := <-done:
				request, ok := task.Context.Value(ctxkeyRequest{}).(transport.IncomingRequest)
				if !ok {
					log.Fatalf("ClientSocketHandler: couldn't get incoming request from context")
				}
				request.Respond(r.Context(), task.Response.Result, task.Response.Error)
				// log.Printf("Task respond: %s :: %#v\n", task.Args.Info.TaskID(), task.Args.Task.Args)

			}
		}

	}
}

// typed key to store original request in a context
type ctxkeyRequest struct{}
