package scheduler

// TODO: check out https://blog.questionable.services/article/http-handler-error-handling-revisited/

import (
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"sync/atomic"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/provider"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

//
// ----------> offloading jobs

// Simply use incrementing IDs for jobs
var jobSequence atomic.Uint64

// An OffloadingJob holds the pb.OffloadWasiArgs from the request along with
// some internal information about the requesting client.
type OffloadingJob struct {
	RequestID  string // used to track all tasks of this request
	ClientAddr string // remote address of the requesting client
	JobSpec    *pb.OffloadWasiJobArgs
}

// The ExecHandler returns a HTTP handler, which accepts run configurations for
// existing WASM binaries and dispatches them to available providers. Upon task
// completion, the results are returned to the HTTP requester.
func ExecHandler(selector Scheduler, benchmode bool) http.HandlerFunc {

	// create a queue for the tasks and start the dispatcher
	// TODO: reuse the ticketing from benchmode to limit concurrent scheduler jobs?
	queue := make(chan *Task, 10)
	go Dispatcher(queue, selector)

	// return the http handler to register as a route
	return func(w http.ResponseWriter, r *http.Request) {

		// if there's something in err upon return, we should log that
		var err error
		defer func() {
			if err != nil {
				log.Printf("ERR: Client [%s]: %s", err)
			}
		}()

		// check the content-type of the request
		mt, _, _ := mime.ParseMediaType(r.Header.Get("content-type"))
		if mt != "application/json" && mt != "application/protobuf" {
			http.Error(w, "unsupported request content-type", http.StatusUnsupportedMediaType)
			return
		}

		// read the entire body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "reading body failed", http.StatusUnprocessableEntity)
			err = fmt.Errorf("reading body failed: %w", err)
			return
		}

		// read the job specification from the request body
		job := OffloadingJob{JobSpec: &pb.OffloadWasiJobArgs{}}
		err = ReadJobArgs(body, mt, job.JobSpec)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			WriteResponse(w, mt, &pb.OffloadWasiJobResponse{
				Error: proto.String(err.Error()),
			})
			err = nil // don't log this
			return
		}

		// amend the job with information about client
		job.RequestID = fmt.Sprintf("%05d", jobSequence.Add(1))
		job.ClientAddr = r.RemoteAddr
		log.Printf("OffloadingJob [%s] from %q: %s, %d tasks\n",
			job.RequestID, job.ClientAddr, job.JobSpec.Binary.GetRef(), len(job.JobSpec.Tasks))

		// compute all the tasks of a request
		results := DispatchTasks(&job, queue)
		// for i, task := range tasks {
		// 	log.Printf("task[%d]: %v, result: %v", i, task, task.Result)
		// }

		// send the result back
		if results.GetError() != "" {
			// set an error code, if there's an error; we don't want a "200 Failed Successfully"
			w.WriteHeader(http.StatusFailedDependency)
		}
		err = WriteResponse(w, mt, results)
	}
}

func ReadJobArgs(body []byte, mt string, spec *pb.OffloadWasiJobArgs) (err error) {

	// try to decode the body to the expected job spec
	switch mt {
	case "application/json":
		err = protojson.Unmarshal(body, spec)
	case "application/protobuf":
		err = proto.Unmarshal(body, spec)
	default:
		panic("oops, unsupported content-type")
	}
	if err != nil {
		return fmt.Errorf("unmarshalling failed: %w", err)
	}

	// check the basic job specification requirements
	if spec.Binary == nil || (spec.Binary.GetRef() == "" && spec.Binary.Blob == nil) {
		err = errors.Join(err, fmt.Errorf("JobSpec: did not specify a binary"))
	}
	if spec.Tasks == nil || len(spec.Tasks) == 0 {
		err = errors.Join(err, fmt.Errorf("JobSpec: no tasks specified"))
	}
	return err

}

func WriteResponse(w http.ResponseWriter, mt string, result *pb.OffloadWasiJobResponse) (err error) {

	// marshal the response to desired format
	var body []byte
	switch mt {
	case "application/json":
		body, err = protojson.Marshal(result)
	case "application/protobuf":
		body, err = proto.Marshal(result)
	default:
		panic("oops, unsupported content-type")
	}
	if err != nil {
		err = fmt.Errorf("response marshalling failed: %w", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// write the response
	w.Header().Set("content-type", mt)
	_, err = w.Write(body)
	if err != nil {
		return fmt.Errorf("writing body failed: %w", err)
	}
	return nil
}

//
// ----------> uploading files

// The UploadHandler returns a HTTP handler, which takes the POSTed body
// and uploads it to the available providers. The providers get marked
// having this file "available" for selection during task execution.
func UploadHandler(store *provider.ProviderStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// if there's something in err upon return, we should log that
		var err error
		defer func() {
			if err != nil {
				log.Printf("ERR: Upload [%s]: %s", err)
			}
		}()

		// check the content-type of the request: accept zip or wasm
		ft, _, _ := mime.ParseMediaType(r.Header.Get("content-type"))
		if ft != "application/wasm" && ft != "application/zip" {
			http.Error(w, "unsupported filetype", http.StatusUnsupportedMediaType)
			return
		}

		// read the entire body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "reading body failed", http.StatusUnprocessableEntity)
			err = fmt.Errorf("reading body failed: %w", err)
			return
		}

		// can have a friendly lookup-name as query parameter
		name := r.URL.Query().Get("name")

		// insert file in storage
		addr, err := store.Storage.Insert(name, ft, body)
		if err != nil {
			http.Error(w, "inserting file in storage failed", http.StatusInternalServerError)
			err = fmt.Errorf("inserting in storage failed: %w", err)
			return
		}

		// return the content address to client
		w.WriteHeader(http.StatusOK)
		w.Header().Add("content-type", "text/plain")
		w.Write([]byte(addr))

	}
}
