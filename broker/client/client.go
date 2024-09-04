package main

import (
	"bytes"
	"encoding/base64"
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

var BROKER = "http://localhost:4080"

func init() {
	// get the Broker URL from env
	if url, ok := os.LookupEnv("BROKER"); ok {
		BROKER = strings.TrimRight(url, "/")
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
		BROKER+"/api/broker/v1/upload?name="+name, mt.String(), bytes.NewBuffer(buf))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// print the response and exit depending on statusCode
	body, _ := io.ReadAll(resp.Body)
	fmt.Println(string(body))
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		os.Exit(1)
	}
	os.Exit(0)

}

// run a prepared job configuration from file
func RunJSON(config string) {

	// read the file
	buf, err := os.ReadFile(config)
	if err != nil {
		log.Fatal("reading file: ", err)
	}

	// decode with protojson and report any errors
	job := &pb.OffloadWasiJobRequest{}
	if err = protojson.Unmarshal(buf, job); err != nil {
		log.Fatal("unmarshal job: ", err)
	}

	RunJob(job)
}

// run a prepared job configuration from proto message
func RunJob(job *pb.OffloadWasiJobRequest) {

	// (re)marshal as binary
	jobpb, err := proto.Marshal(job)
	if err != nil {
		log.Fatal("can't remarshal: ", err)
	}

	// send the request
	resp, err := http.Post(
		BROKER+"/api/broker/v1/run", "application/protobuf", bytes.NewBuffer(jobpb))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	ParseResult(resp)
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

	// dump as JSON
	js, _ := protojson.Marshal(job)
	fmt.Println("-->", string(js))

	RunJob(job)
}

// parse a result and print the stdout/stderr as strings
func ParseResult(resp *http.Response) {

	// read the full response
	body, _ := io.ReadAll(resp.Body)
	response := &pb.OffloadWasiJobResponse{}
	if err := proto.Unmarshal(body, response); err != nil {
		log.Println("can't unmarshal response: ", err)
		fmt.Println(string(body))
		os.Exit(1)
	}

	// print failures
	if f := response.GetFailure(); f != "" {
		log.Fatal("job failed: ", f)
	}

	// print task results
	for i, task := range response.Tasks {
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

	// non-zero exit on failures
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		os.Exit(1)
	}
	os.Exit(0)

}

func main() {

	usage := fmt.Sprintf(
		"usage: %s { run config.json, exec args[], upload app.wasm }",
		path.Base(os.Args[0]))
	if len(os.Args) < 2 {
		fmt.Println(usage)
		os.Exit(1)
	}
	command := os.Args[1]
	switch command {

	case "upload":
		if len(os.Args) < 3 {
			log.Fatal("filename required")
		}
		filename := os.Args[2]
		name := ""
		if len(os.Args) > 3 {
			name = os.Args[3]
		}
		UploadFile(filename, name)

	case "run":
		if len(os.Args) < 3 {
			log.Fatal("run configuration required")
		}
		config := os.Args[2]
		RunJSON(config)

	case "exec":
		if len(os.Args) < 3 {
			log.Fatal("first argument is the binary")
		}
		Execute(os.Args[2:], []string{})

	default:
		fmt.Printf("unknown command: %q\n", command)
		fmt.Println(usage)
		os.Exit(1)
	}

}
