package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"wasimoff/broker/net/transport"
	wasimoff "wasimoff/proto/v1"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	brokerUrl = "http://localhost:4080" // default broker base URL
	verbose   = false                   // be more verbose
	readstdin = false                   // read stdin for exec
	websock   = false                   // use websocket to send tasks
)

func init() {
	// get the Broker URL from env
	if url, ok := os.LookupEnv("BROKER"); ok {
		brokerUrl = strings.TrimRight(url, "/")
	}
}

func main() {

	// commandline parser
	flag.StringVar(&brokerUrl, "broker", brokerUrl, "URL to the Broker to use")
	upload := flag.String("upload", "", "Upload a file (wasm or zip) to the Broker and receive its ref")
	exec := flag.Bool("exec", false, "Execute an uploaded binary by passing all non-flag args")
	run := flag.String("run", "", "Run a prepared JSON job file")
	runpy := flag.String("runpy", "", "Run a Python script file with Pyodide")
	flag.BoolVar(&verbose, "verbose", verbose, "Be more verbose and print raw messages for -exec")
	flag.BoolVar(&readstdin, "stdin", readstdin, "Read and send stdin when using -exec (not streamed)")
	flag.BoolVar(&websock, "ws", websock, "Use a WebSocket to send tasks")
	flag.Parse()

	switch true {

	// upload a file, optionally take another argument as name alias
	case *upload != "":
		alias := flag.Arg(0)
		UploadFile(*upload, alias)

	// execute an ad-hoc command, as if you were to run it locally
	case *exec:
		envs := []string{} // TODO: read os.Environ?
		args := flag.Args()
		Execute(args, envs)

	// execute a prepared JSON job
	case *run != "":
		RunJsonFile(*run)

	// execute a python script task
	case *runpy != "":
		RunPythonScript(*runpy)

	// no command specified
	default:
		fmt.Fprintln(os.Stderr, "ERR: at least one of -upload, -exec or -run must be used")
		flag.Usage()
		os.Exit(2)
	}

}

// upload a local file to the Broker
func UploadFile(filename, name string) {

	// read the file
	buf, err := os.ReadFile(filename)
	if err != nil {
		log.Fatal("reading file: ", err)
	}

	// detect the mediatype from buf
	mt := mimetype.Detect(buf)

	// reuse basename as name if it's empty
	if name == "" {
		name = filepath.Base(filename)
	}

	// upload to the broker
	resp, err := http.Post(
		brokerUrl+"/api/storage/upload?name="+name, mt.String(), bytes.NewBuffer(buf))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// print the response and exit depending on statusCode
	body, _ := io.ReadAll(resp.Body)
	fmt.Fprint(os.Stdout, string(body))
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintln(os.Stderr, resp.Status)
		os.Exit(1)
	}
	os.Exit(0)

}

// execute an ad-hoc command by constructing configuration
func Execute(args, envs []string) {
	if len(args) == 0 {
		log.Fatal("need at least one argument")
	}

	// construct an ad-hoc job
	job := &wasimoff.Client_Job_Wasip1Request{
		Tasks: []*wasimoff.Task_Wasip1_Params{{
			Binary: &wasimoff.File{Ref: proto.String(args[0])},
			Args:   args,
			Envs:   envs,
		}},
	}

	// optionally read stdin
	if readstdin {
		stdin, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR: failed reading stdin:", err)
			os.Exit(1)
		}
		job.Tasks[0].Stdin = stdin
	}

	// dump as JSON and run the job
	if verbose {
		js, _ := protojson.Marshal(job)
		log.Println("run:", string(js))
	}
	results := RunJob(job)

	// there should be exactly one result, print it
	task := results[0]
	if task.GetError() != "" {
		fmt.Fprintln(os.Stderr, "ERR:", task.GetError())
		os.Exit(1)
	} else {
		r := task.GetOk()
		if verbose {
			js, _ := protojson.Marshal(r)
			log.Println("result:", string(js))
		}
		if len(r.GetStderr()) != 0 {
			fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m", string(r.GetStderr()))
		}
		fmt.Fprint(os.Stdout, string(r.GetStdout()))
		os.Exit(int(r.GetStatus()))
	}

}

// run a prepared job configuration from file
func RunJsonFile(config string) {

	// read the file
	buf, err := os.ReadFile(config)
	if err != nil {
		log.Fatal("reading file: ", err)
	}

	// decode with protojson and report any errors locally
	job := &wasimoff.Client_Job_Wasip1Request{}
	if err = protojson.Unmarshal(buf, job); err != nil {
		log.Fatal("unmarshal job: ", err)
	}

	// run the job
	results := RunJob(job)

	// print all task results
	for i, task := range results {
		if task.GetError() != "" {
			fmt.Fprintf(os.Stderr, "[task %d FAIL] %s\n", i, task.GetError())
		} else {
			r := task.GetOk()
			fmt.Fprintf(os.Stderr, "[task %d => exit:%d]\n", i, *r.Status)
			if r.Artifacts != nil {
				fmt.Fprintf(os.Stderr, "artifact: %s\n", base64.StdEncoding.EncodeToString(r.Artifacts.GetBlob()))
			}
			if len(r.GetStderr()) != 0 {
				fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(r.GetStderr()))
			}
			fmt.Fprintln(os.Stdout, string(r.GetStdout()))
		}
	}

}

// run a prepared job configuration from proto message
func RunJob(job *wasimoff.Client_Job_Wasip1Request) []*wasimoff.Task_Wasip1_Result {

	// short-circuit to alternative function, when we should be using websocket
	if websock {
		return RunJobOnWebSocket(job)
	}

	// (re)marshal as binary
	jobpb, err := proto.Marshal(job)
	if err != nil {
		log.Fatal("can't remarshal: ", err)
	}

	// send the request
	resp, err := http.Post(
		brokerUrl+"/api/client/run", "application/protobuf", bytes.NewBuffer(jobpb))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// wait and read the full response
	body, _ := io.ReadAll(resp.Body)
	response := &wasimoff.Client_Job_Wasip1Response{}
	if err := proto.Unmarshal(body, response); err != nil {
		log.Println("can't unmarshal response: ", err)
		fmt.Fprintln(os.Stderr, string(body))
		os.Exit(1)
	}

	// print error if HTTP status isn't OK
	if resp.StatusCode != http.StatusOK {
		log.Print("http error:", resp.Status)
		fmt.Fprintln(os.Stderr, string(body))
		os.Exit(1)
	}

	// print overall failures
	if response.Error != nil {
		log.Fatal("job failed: ", response.GetError())
	}

	return response.GetTasks()
}

// alternatively, run a job by sending each task over websocket
func RunJobOnWebSocket(job *wasimoff.Client_Job_Wasip1Request) []*wasimoff.Task_Wasip1_Result {

	// open a websocket to the broker
	socket, err := transport.DialWebSocketTransport(context.TODO(), brokerUrl+"/api/client/ws")
	if err != nil {
		log.Printf("ERR: opening websocket: %s", err)
	}
	// wrap it in a messenger for RPC
	messenger := transport.NewMessengerInterface(socket)
	defer messenger.Close(nil)

	// chan and list to collect responses
	ntasks := len(job.GetTasks())
	done := make(chan *transport.PendingCall, ntasks)
	responses := make([]*wasimoff.Task_Wasip1_Result, ntasks)

	// submit all tasks
	for i, task := range job.GetTasks() {
		task.InheritNil(job.Parent)
		if verbose {
			log.Printf("websocket: submit task %d", i)
		}

		// store index in context
		ctx := context.WithValue(context.TODO(), ctxJobIndex{}, i)

		// assemble wrapped task and fire it off
		tr := &wasimoff.Task_Request{
			Parameters: &wasimoff.Task_Request_Wasip1{
				Wasip1: task,
			},
		}
		messenger.SendRequest(ctx, tr, &wasimoff.Task_Response{}, done)
	}

	// wait for all responses
	for ntasks > 0 {
		call := <-done
		ntasks -= 1
		i := call.Context.Value(ctxJobIndex{}).(int)
		if verbose {
			log.Printf("websocket: received result %d: err=%v", i, call.Error)
		}

		if call.Error != nil {
			responses[i].Result = &wasimoff.Task_Wasip1_Result_Error{
				Error: call.Error.Error(),
			}
		} else {
			if resp, ok := call.Response.(*wasimoff.Task_Response); ok {
				responses[i].Result = resp.GetWasip1().Result
			} else {
				responses[i].Result = &wasimoff.Task_Wasip1_Result_Error{
					Error: "failed to parse the response as pb.Task_Response",
				}
			}
		}
	}

	return responses
}

// run a python script from file
func RunPythonScript(script string) {

	// read the file
	buf, err := os.ReadFile(script)
	if err != nil {
		log.Fatal("reading file: ", err)
	}
	script = string(buf)

	// prepare a request using this file
	request := &wasimoff.Task_Request{
		Parameters: &wasimoff.Task_Request_Pyodide{
			Pyodide: &wasimoff.Task_Pyodide_Params{
				Script: &script,
			},
		},
	}

	// open a websocket to the broker
	socket, err := transport.DialWebSocketTransport(context.TODO(), brokerUrl+"/api/client/ws")
	if err != nil {
		log.Printf("ERR: opening websocket: %s", err)
	}
	// wrap it in a messenger for RPC
	messenger := transport.NewMessengerInterface(socket)
	defer messenger.Close(nil)

	// send the request
	response := &wasimoff.Task_Response{}
	messenger.RequestSync(context.TODO(), request, response)

	// print all task results
	if response.GetError() != "" {
		fmt.Fprintf(os.Stderr, "[task py FAIL] %s\n", response.GetError())
	} else {
		if response.GetPyodide().GetError() != "" {
			fmt.Fprintf(os.Stderr, "[task py FAIL] %s\n", response.GetPyodide().GetError())
			return
		}
		r := response.GetPyodide().GetOk()
		fmt.Fprintf(os.Stderr, "# Pyodide v%s: https://pyodide.org/en/%s/usage/packages-in-pyodide.html\n", r.GetVersion(), r.GetVersion())
		if len(r.GetStderr()) != 0 {
			fmt.Fprintf(os.Stderr, "\033[31m%s\033[0m\n", string(r.GetStderr()))
		}
		fmt.Fprintln(os.Stdout, string(r.GetStdout()))
		if r.Pickle != nil {
			fmt.Fprintf(os.Stderr, "\nresult pickle: %s\n", base64.StdEncoding.EncodeToString(r.GetPickle()))
		}
	}

}

// typed key to store index in context
type ctxJobIndex struct{}
