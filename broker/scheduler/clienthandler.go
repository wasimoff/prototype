package scheduler

// TODO: check out https://blog.questionable.services/article/http-handler-error-handling-revisited/

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"sync/atomic"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/provider"
	"wasimoff/broker/storage"

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
	JobID      string // used to track all tasks of this request
	ClientAddr string // remote address of the requesting client
	JobSpec    *pb.OffloadWasiJobRequest
}

// The ExecHandler returns a HTTP handler, which accepts run configurations for
// existing WASM binaries and dispatches them to available providers. Upon task
// completion, the results are returned to the HTTP requester.
// MARK: ExecHdl
func ExecHandler(store *provider.ProviderStore, selector Scheduler) http.HandlerFunc {

	// create a queue for the tasks and start the dispatcher
	// TODO: reuse the ticketing from benchmode to limit concurrent scheduler jobs
	queue := make(chan *provider.AsyncWasiTask, 10)
	go Dispatcher(selector, queue)

	// return the http handler to register as a route
	return func(w http.ResponseWriter, r *http.Request) {

		// if there's something in err upon return, we should log that
		var err error
		defer func() {
			if err != nil {
				log.Printf("ERR: Client [%s]: %s", r.RemoteAddr, err)
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
		job := OffloadingJob{JobSpec: &pb.OffloadWasiJobRequest{}}
		err = UnmarshalJobArgs(body, mt, job.JobSpec)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			MarshalJobResponse(w, mt, &pb.OffloadWasiJobResponse{
				Failure: proto.String(err.Error()),
			})
			err = nil // don't log this
			return
		}

		// amend the job with information about client
		job.JobID = fmt.Sprintf("%05d", jobSequence.Add(1))
		job.ClientAddr = r.RemoteAddr
		log.Printf("OffloadingJob [%s] from %q: %d tasks\n",
			job.JobID, job.ClientAddr, len(job.JobSpec.Tasks))

		// compute all the tasks of a request
		results := DispatchTasks(r.Context(), store, &job, queue)

		// send the result back, if not canceled
		if cerr := r.Context().Err(); cerr != nil {
			log.Printf("OffloadingJob [%s] from %q: canceled!", job.JobID, r.RemoteAddr)
			http.Error(w, "request canceled", http.StatusRequestTimeout)
		} else {
			if results.GetFailure() != "" {
				// set an error code, if there's an error; we don't want a "200 Failed Successfully"
				w.WriteHeader(http.StatusFailedDependency)
			}
			err = MarshalJobResponse(w, mt, results)
		}
	}
}

// DispatchTasks takes a run configuration, generates individual tasks from it,
// schedules them in the queue and eventually returns with the results of all
// those tasks.
// TODO: accept a Context, so pending tasks can be cancelled from ExecHandler
// MARK: Dispat.
func DispatchTasks(
	ctx context.Context,
	store *provider.ProviderStore,
	job *OffloadingJob,
	queue chan *provider.AsyncWasiTask,
) *pb.OffloadWasiJobResponse {

	// go through all the *pb.Files and resolve them from storage
	errs := []error{}
	if job.JobSpec.Parent != nil {
		errs = append(errs, store.Storage.ResolvePbFile(job.JobSpec.Parent.Binary))
		errs = append(errs, store.Storage.ResolvePbFile(job.JobSpec.Parent.Rootfs))
	}
	for _, task := range job.JobSpec.Tasks {
		errs = append(errs, store.Storage.ResolvePbFile(task.Binary))
		errs = append(errs, store.Storage.ResolvePbFile(task.Rootfs))
	}
	if err := errors.Join(errs...); err != nil {
		return &pb.OffloadWasiJobResponse{
			Failure: proto.String(err.Error()),
		}
	}

	// create slice for queued tasks and a sufficiently large channel for done signals
	pending := make([]*provider.AsyncWasiTask, len(job.JobSpec.Tasks))
	doneChan := make(chan *provider.AsyncWasiTask, len(pending)+10)

	for i, spec := range job.JobSpec.Tasks {

		// create the request+response for remote procedure call
		response := pb.ExecuteWasiResponse{}
		request := pb.ExecuteWasiRequest{
			// common task metadata with index counter
			Info: &pb.TaskMetadata{
				JobID:  &job.JobID,
				Index:  proto.Uint64(uint64(i)),
				Client: &job.ClientAddr,
			},
			// inherit empty parameters from the parent job
			Task: spec.InheritNil(job.JobSpec.Parent),
		}

		// create the async task with the common done channel and queue it for dispatch
		task := provider.NewAsyncWasiTask(ctx, &request, &response, doneChan)
		pending[i] = task
		queue <- task
	}

	// wait for all tasks to finish
	done := 0
	for t := range doneChan {
		done++
		if t.Error == nil {
			store.RateTick()
		}
		if done == len(pending) {
			break
		}
	}

	// collect the task responses
	jobResponse := &pb.OffloadWasiJobResponse{
		Tasks: make([]*pb.ExecuteWasiResponse, len(pending)),
	}
	for i, task := range pending {
		jobResponse.Tasks[i] = task.Response
		if task.Error != nil && task.Response.Error == nil {
			jobResponse.Tasks[i].Error = proto.String(task.Error.Error())
		}
	}

	return jobResponse
}

// MARK: Marshal
func UnmarshalJobArgs(body []byte, mt string, spec *pb.OffloadWasiJobRequest) (err error) {

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
	if len(spec.Tasks) == 0 {
		err = errors.Join(err, fmt.Errorf("JobSpec: no tasks specified"))
	}
	return err
}

func MarshalJobResponse(w http.ResponseWriter, mt string, result *pb.OffloadWasiJobResponse) (err error) {

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
// MARK: Upload
func UploadHandler(store *provider.ProviderStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// if there's something in err upon return, we should log that
		var err error
		defer func() {
			if err != nil {
				log.Printf("ERR: Upload [%s]: %s", r.RemoteAddr, err)
			}
		}()

		// check the content-type of the request: accept zip or wasm
		ft, err := storage.CheckMediaType(r.Header.Get("content-type"))
		if err != nil {
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
		file, err := store.Storage.Insert(name, ft, body)
		if err != nil {
			http.Error(w, "inserting file in storage failed", http.StatusInternalServerError)
			err = fmt.Errorf("inserting in storage failed: %w", err)
			return
		}

		// return the content address to client
		w.WriteHeader(http.StatusOK)
		w.Header().Add("content-type", "text/plain")
		fmt.Fprintln(w, file.Ref())

		// upload the file to all providers asynchronously
		go func() {
			store.Range(func(addr string, provider *provider.Provider) bool {
				if err = provider.Upload(file); err != nil {
					log.Printf("Upload to %q failed: %s", addr, file.Ref())
				}
				return true // iterate whole range, despite errors
			})

		}()

	}
}
