package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"wasimoff/broker/net/pb"

	"github.com/gabriel-vasile/mimetype"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// default URL to use for the brokerUrl
var brokerUrl = "http://localhost:4080"

const apiPrefix = "/api/broker/v1"

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
		brokerUrl+apiPrefix+"/upload?name="+name, mt.String(), bytes.NewBuffer(buf))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// print the response and exit depending on statusCode
	body, _ := io.ReadAll(resp.Body)
	fmt.Print(string(body))
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
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
	job := &pb.OffloadWasiJobRequest{
		Tasks: []*pb.WasiTaskArgs{{
			Binary: &pb.File{Ref: proto.String(args[0])},
			Args:   args,
			Envs:   envs,
			// Stdin: os.Stdin, // TODO: detect if stdin is a terminal, else read from it
		}},
	}

	// dump as JSON and run the job
	js, _ := protojson.Marshal(job)
	log.Println("run:", string(js))
	results := RunJob(job)

	// there should be exactly one result, print it
	task := results[0]
	if task.GetError() != "" {
		fmt.Fprintln(os.Stderr, "ERR:", task.GetError())
	} else {
		r := task.GetResult()
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
			fmt.Printf("[task %d FAIL] %s\n", i, task.GetError())
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
		brokerUrl+apiPrefix+"/run", "application/protobuf", bytes.NewBuffer(jobpb))
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
