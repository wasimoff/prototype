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
	"path"
	"strings"
	"sync"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/net/transport"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// default URL to use for the brokerUrl
var brokerUrl = "http://localhost:4080"

// be more verbose
var verbose bool

// read stdin for exec
var readstdin bool

func init() {
	// get the Broker URL from env
	if url, ok := os.LookupEnv("BROKER"); ok {
		brokerUrl = strings.TrimRight(url, "/")
	}
}

func main() {

	// commandline parser
	flag.StringVar(&brokerUrl, "broker", brokerUrl, "URL to the Broker to use")
	upload := flag.String("upload", "", "Upload a file (wasm or zip) to the Broker and receive it ref")
	exec := flag.String("exec", "", "Execute an uploaded binary; separate further app arguments with '--'")
	run := flag.String("run", "", "Run a prepared JSON job file")
	flag.BoolVar(&verbose, "verbose", false, "Be more verbose and print raw messages for -exec")
	flag.BoolVar(&readstdin, "stdin", false, "Read and send stdin when using -exec (not streamed)")
	flag.Parse()

	switch true {

	// upload a file, optionally take another argument as name alias
	case *upload != "":
		alias := flag.Arg(0)
		UploadFile(*upload, alias)

	// execute an ad-hoc command, as if you were to run it locally
	case *exec != "":
		envs := []string{} // TODO: read os.Environ?
		args := append([]string{*exec}, flag.Args()...)
		Execute(args, envs)

	// execute a prepared JSON job
	case *run != "":
		RunJsonFile(*run)

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
		name = path.Base(filename)
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

	// TODO: websocket testing
	ctx, cancel := context.WithCancel(context.Background())
	wt, err := transport.DialWebSocketTransport(ctx, brokerUrl+"/api/client/ws")
	if err != nil {
		log.Printf("ERR: websocket: %s", err)
	}
	messenger := transport.NewMessengerInterface(wt)

	wg := sync.WaitGroup{}
	for i := 0; i < 32; i++ {
		wg.Add(1)
		go func() {
			res := pb.WasiTaskResult{}
			err = messenger.RequestSync(ctx, &pb.WasiTaskArgs{
				Binary: &pb.File{Ref: proto.String(args[0])},
				Args:   args,
				Envs:   envs,
				Stdin:  []byte{},
			}, &res)
			if err != nil {
				log.Printf("client socket err: %s", err)
			} else {
				log.Printf("socketed task done: %s", protojson.Format(&res))
			}
			wg.Done()
		}()
	}
	wg.Wait()
	cancel()

	// construct an ad-hoc job
	job := &pb.OffloadWasiJobRequest{
		Tasks: []*pb.WasiTaskArgs{{
			Binary: &pb.File{Ref: proto.String(args[0])},
			Args:   args,
			Envs:   envs,
			Stdin:  []byte{},
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
		r := task.GetResult()
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
	job := &pb.OffloadWasiJobRequest{}
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
			r := task.GetResult()
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
func RunJob(job *pb.OffloadWasiJobRequest) []*pb.ExecuteWasiResponse {

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
	response := &pb.OffloadWasiJobResponse{}
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
	if f := response.GetFailure(); f != "" {
		log.Fatal("job failed: ", f)
	}

	return response.Tasks
}
