package scheduler

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"wasimoff/broker/net/transport"
	"wasimoff/broker/provider"
	wasimoff "wasimoff/proto/v1"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

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
		job := fmt.Sprintf("websocket/%05d", jobSequence.Add(1))
		requestSequence := uint64(0)

		// channel for finished requests
		// TODO: limit task creation with an equally-sized ticket channel
		done := make(chan *provider.AsyncTask, 32)

		defer log.Printf("[%s] Client socket closed", addr)
		for {
			select {

			// connection closing
			case <-r.Context().Done():
				return
			case <-messenger.Closing():
				return

			// print any received events
			case event, ok := <-messenger.Events():
				if !ok { // messenger closing
					return
				}
				log.Printf("{client %s} %s", addr, prototext.Format(event))

			// dispatch received requests
			case request, ok := <-messenger.Requests():
				if !ok { // messenger closing
					return
				}
				switch taskrequest := request.Request.(type) {

				case *wasimoff.Task_Request:
					requestSequence++

					// resolve any filenames to storage hashes
					if ferr := store.Storage.ResolveTaskFiles(taskrequest); ferr != nil {
						request.Respond(r.Context(), nil, ferr)
						continue // handle next request
					}

					// assemble the task for internal dispatcher queue
					taskrequest.Info = &wasimoff.Task_Metadata{
						Id:        proto.String(fmt.Sprintf("%s/%d", job, requestSequence)),
						Requester: &addr,
					}
					response := wasimoff.Task_Response{}
					taskctx := context.WithValue(r.Context(), ctxkeyRequest{}, request)
					taskQueue <- provider.NewAsyncTask(taskctx, taskrequest, &response, done)
					// log.Printf("Task submit: %s :: %#v\n", wreq.Info.TaskID(), wreq.Task.Args)
					continue

				default: // unexpected message type
					request.Respond(r.Context(), nil, fmt.Errorf("expecting only Task_Request messages on this socket"))
					continue

				}

			// respond with finished results
			case task := <-done:
				request, ok := task.Context.Value(ctxkeyRequest{}).(transport.IncomingRequest)
				if !ok {
					log.Fatalf("ClientSocketHandler: couldn't get incoming request from context")
				}

				// pass through both internal and response errors directly
				request.Respond(r.Context(), task.Response, task.Error)
				// log.Printf("Task respond: %s :: %#v\n", task.Args.Info.TaskID(), task.Args.Task.Args)

			}
		}

	}
}

// typed key to store original request in a context
type ctxkeyRequest struct{}
