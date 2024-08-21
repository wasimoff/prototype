package scheduler

// TODO: check out https://blog.questionable.services/article/http-handler-error-handling-revisited/

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"strings"
	"wasimoff/broker/provider"
	"wasimoff/broker/storage"
	"wasimoff/broker/tracer"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"golang.org/x/exp/slices"
)

// Use a single Validator instance to cache struct info
var validate *validator.Validate

func init() {
	validate = validator.New()
}

// Validator (`validate:""` tags) README:
// https://pkg.go.dev/github.com/go-playground/validator/v10#section-readme

// The overall RunConfiguration in a single request.
type RunConfiguration struct {
	// The run ID is generated internally for reference purposes.
	RunID uuid.UUID `json:"-"`
	// Metadata about the requestor.
	Requestor *Requestor `json:"-"`
	// The filename of the binary to use for all of these runs.
	Binary string `json:"bin" validate:"required,filepath"`
	// Global environment variables that are the same for all runs.
	//! no deduplication with task envs is performed yet
	Environ []string `json:"envs" validate:"dive,required,contains=="`
	// An array of parametrized runs on the common binary.
	Exec []ParametrizedRun `json:"exec" validate:"required,min=1,dive"`
	// Enable timestamp tracing of executions.
	Trace bool `json:"trace"`
}

// A single parametrized run that should be sent to the providers.
type ParametrizedRun struct {
	Args     []string `json:"args" validate:"required_without=Envs"`
	Envs     []string `json:"envs" validate:"required_without=Args,dive,required,contains=="`
	Stdin    string   `json:"stdin"`
	LoadFs   []string `json:"loadfs"`
	Datafile string   `json:"datafile"`
}

// Metadata about the requesting client that is populated by the handler.
type Requestor struct {
	RemoteAddr string
}

// The ExecHandler returns a HTTP handler, which accepts run configurations for
// existing WASM binaries and dispatches them to available providers. Upon task
// completion, the results are returned to the HTTP requester.
func ExecHandler(selector Scheduler) http.HandlerFunc {

	// create a queue for the tasks and start the dispatcher
	queue := make(chan *Task, 10)
	go Dispatcher(queue, selector)

	// return the http handler to register as a route
	return func(w http.ResponseWriter, r *http.Request) {

		// track the start time of request
		trace := new(tracer.Trace)
		trace.Now("broker: request received")

		// the request body should be a run configuration
		run, err := decodeRunConfiguration(r.Header.Get("content-type"), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		run.Requestor = &Requestor{r.RemoteAddr}
		if run.Trace {
			// trace not supported for more than one task
			if len(run.Exec) != 1 {
				http.Error(w, "trace is only supported for a single task", http.StatusBadRequest)
				return
			}
			trace.Now("broker: configuration decoded")
		} else {
			trace = nil
		}
		log.Printf("Offloading Request by %q: %v\n", run.Requestor.RemoteAddr, run)

		// early exit if there are no providers
		err = selector.Ok()
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}

		// compute all the tasks of a request
		tasks := DispatchTasks(run, trace, queue)
		// for i, task := range tasks {
		// 	log.Printf("task[%d]: %v, result: %v", i, task, task.Result)
		// }

		// send the result back
		w.Header().Set("content-type", "application/json")
		json.NewEncoder(w).Encode(FormatExecResults(tasks))
	}
}

// Decode the `RunConfiguration` from a request and check for required fields.
func decodeRunConfiguration(contentType string, body io.ReadCloser) (*RunConfiguration, error) {

	// parse the content-type
	mt, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to parse content-type: %w", err)
	}

	// nothing but JSON implemented yet
	if mt != "application/json" {
		return nil, fmt.Errorf("no config parsers besides JSON are implemented yet")
	}

	// parse the config
	var run RunConfiguration
	json.NewDecoder(body).Decode(&run)

	// check for required fields
	err = validate.Struct(run)
	if err != nil {
		return nil, fmt.Errorf("failed to validate run configuration: %w", err)
	}

	// assign the uuid and return
	run.RunID = uuid.New()
	return &run, nil

}

// The UploadHandler returns a HTTP handler, which takes the POSTed body
// and uploads it to the available providers. The providers get marked
// having this file "available" for selection during task execution.
func UploadHandler(store *provider.ProviderStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// need a name as query parameter
		name := r.URL.Query().Get("name")
		if name == "" {
			http.Error(w, "need a name in query parameter", http.StatusBadRequest)
			return
		}

		// persist this file to upload to new providers as they connect
		persist := func(q string) bool {
			// mostly true by default, except when falsy value
			return !slices.Contains[[]string]([]string{"n", "no", "false"}, strings.ToLower(q))
		}(r.URL.Query().Get("persist"))

		// read the expected binary or return a "Bad Request"
		bytes, err := uploadedFile(r.Header.Get("content-type"), r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// convert bytes and name to *File and maybe insert in storage
		file := storage.NewFile(name, bytes)
		if persist {
			store.Storage.Insert(file)
		}

		// early-exit if no providers available
		if store.Size() == 0 {
			http.Error(w, "provider list is empty", http.StatusServiceUnavailable)
			return
		}

		// otherwise upload to all available
		errMap := make(map[string]string)
		hasErr := false
		store.Range(func(addr string, provider *provider.Provider) bool {
			log.Printf("upload %q to %q", name, addr)
			err = provider.Upload(file)
			if err != nil {
				hasErr = true
				errMap[addr] = err.Error()
			} else {
				errMap[addr] = "OK"
			}
			return true // iterate whole range, despite errors
		})

		// return result
		if hasErr {
			w.Header().Set("content-type", "application/json")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(errMap)
		} else {
			fmt.Fprintf(w, "Upload OK, %d bytes", len(bytes))
		}
	}
}

// Just read the entire body into a byte buffer.
func uploadedFile(contentType string, body io.ReadCloser) ([]byte, error) {

	//! this check was removed to be able to upload any kind of file for now
	// // the request body should contain a WebAssembly binary
	// mt, _, err := mime.ParseMediaType(contentType)
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to parse content-type: %w", err)
	// }
	// if mt != "application/wasm" {
	// 	return nil, fmt.Errorf("binary must be a WASM file")
	// }

	// read the binary
	wasm, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("failed to read entire request body: %w", err)
	}
	return wasm, nil
}
