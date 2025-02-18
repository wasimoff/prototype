package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"sync/atomic"
	"time"
	"wasimoff/broker/net/transport"
	wasimoff "wasimoff/proto/v1"

	"google.golang.org/protobuf/proto"
)

var broker string = "http://localhost:4080"
var tspn int = 10
var parallel int = 32

func main() {

	flag.IntVar(&tspn, "tsp", tspn, "tsp rand N")
	flag.IntVar(&parallel, "p", parallel, "how many parallel tasks to have in-flight")
	flag.StringVar(&broker, "wasimoff", broker, "URL to the Wasimoff Broker")
	flag.Parse()

	// open a websocket to the broker
	messenger, err := transport.DialWasimoff(context.TODO(), broker)
	if err != nil {
		log.Fatalf("ERR: %s", err)
	}
	defer messenger.Close(nil)

	// construct task structure once
	task := &wasimoff.Task_Wasip1_Params{
		Binary: &wasimoff.File{Ref: proto.String("tsp.wasm")},
		Args:   []string{"tsp.wasm", "rand", fmt.Sprintf("%d", tspn)},
	}

	// use "tickets" to limit the number of concurrent tasks in-flight
	tickets := make(chan struct{}, parallel)
	for len(tickets) < cap(tickets) {
		tickets <- struct{}{}
	}

	// count completed requests
	counter := atomic.Uint64{}

	for {
		<-tickets
		go func() {

			result := &wasimoff.Task_Wasip1_Result{}
			err := messenger.RequestSync(context.TODO(), task, result)
			if err != nil {
				time.Sleep(time.Second)
			}

			tickets <- struct{}{}
			c := counter.Add(1)
			fmt.Printf("\rrequests: %12d", c)

		}()
	}

}
