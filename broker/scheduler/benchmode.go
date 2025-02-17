package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"
	"wasimoff/broker/net/pb"
	"wasimoff/broker/provider"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func tspbench(store *provider.ProviderStore, parallel int) {

	// wait for required binary upload
	bin := "tsp.wasm"
	args := []string{"tsp.wasm", "rand", "10"}
	log.Printf("BENCHMODE: please upload %q binary", bin)
	binary := pb.File{Ref: &bin}
	for {
		if store.Storage.Get(bin) != nil {
			// file uploaded
			log.Printf("BENCHMODE: required binary uploaded, let's go ...")
			err := store.Storage.ResolvePbFile(&binary) // ! <-- this one is important
			if err != nil {
				panic(err)
			}
			break
		}
		time.Sleep(time.Second)
	}

	// use "tickets" to limit the number of concurrent tasks in-flight
	tickets := make(chan struct{}, parallel)
	for len(tickets) < cap(tickets) {
		tickets <- struct{}{}
	}

	// receive finished tasks to tick the throughput counter and reinsert ticket
	doneChan := make(chan *provider.AsyncTask, parallel)
	go func() {
		for t := range doneChan {
			if t.Error == nil {
				// store.RateTick()
			}
			tickets <- struct{}{}
		}
	}()

	// loop forever with incrementing index
	for i := 0; ; i++ {
		<-tickets
		taskQueue <- provider.NewAsyncTask(
			context.Background(),
			&pb.Task_Request{
				Info: &pb.Task_Metadata{
					Id: proto.String(fmt.Sprintf("benchmode/%d", i)),
				},
				Parameters: &pb.Task_Request_Wasip1{
					Wasip1: &pb.Task_Wasip1_Params{
						Binary: &binary,
						Args:   args,
					},
				},
			},
			&pb.Task_Response{},
			doneChan,
		)
	}
}

func pytest(parallel int) {

	// use "tickets" to limit the number of concurrent tasks in-flight
	tickets := make(chan struct{}, parallel)
	for len(tickets) < cap(tickets) {
		tickets <- struct{}{}
	}

	// receive finished tasks to tick the throughput counter and reinsert ticket
	doneChan := make(chan *provider.AsyncTask, parallel)
	go func() {
		for t := range doneChan {
			if t.Error != nil {
				fmt.Printf("ERR: %s\n", t.Error)
			} else {
				if t.Response.GetError() != "" {
					fmt.Printf("Pytest ERR: %s\n", t.Response.GetError())
				} else {
					fmt.Printf("Pytest: %s\n", prototext.Format(t.Response.GetPyodide().GetOk()))
				}
			}
			tickets <- struct{}{}
		}
	}()

	// loop forever with incrementing index
	for i := 0; ; i++ {
		<-tickets
		taskQueue <- provider.NewAsyncTask(
			context.Background(),
			&pb.Task_Request{
				Info: &pb.Task_Metadata{
					Id: proto.String(fmt.Sprintf("pytest/%d", i)),
				},
				Parameters: &pb.Task_Request_Pyodide{
					Pyodide: &pb.Task_Pyodide_Params{
						Script:   proto.String("import numpy as np; mat = np.random.rand(5,5); print(mat); mat.mean()"),
						Packages: []string{"numpy"},
					},
				},
			},
			&pb.Task_Response{},
			doneChan,
		)
	}
}
