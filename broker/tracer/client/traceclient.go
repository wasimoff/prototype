package main

// go run tracer/client/traceclient.go | bat --language csv

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
	"wasmoff/broker/scheduler"
	"wasmoff/broker/tracer"
)

// fixed broker url and payload for now
const BROKER_RUN string = "http://localhost:4080/api/broker/v1/run"
const PAYLOAD string = `{ "bin": "tsp.wasm", "trace": true, "exec": [{ "args": [ "rand", "8" ] }] }`

func main() {
	client := http.Client{Timeout: 10 * time.Second}
	// Single(&client)
	Averaged(&client, 64)
}

func Averaged(client *http.Client, n int) {
	// run n traces
	traces := make([][]TraceEvent, n)
	for i := 0; i < n; i++ {
		traces[i] = run(client, []byte(PAYLOAD))
	}
	// check if all traces have the same events
	for _, trace := range traces {
		for i, event := range trace {
			if event.Label != traces[0][i].Label {
				panic("traces didn't log the same events")
			}
		}
	}
	// calculate averages
	averages := make([]TraceEvent, len(traces[0]))
	for x, event := range traces[0] {
		averages[x].Label = event.Label
		step := 0.0
		for i := 0; i < len(traces); i++ {
			step += traces[i][x].Step
		}
		averages[x].Step = (step / float64(len(traces)))
	}
	// encode averages to stdout as csv
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	w.Comma = ';'
	w.Write([]string{"step_ms", "label"})
	for _, event := range averages {
		w.Write([]string{
			fmt.Sprintf("%.3f", event.Step),
			event.Label,
		})
	}
}

func Single(client *http.Client) {
	trace := run(client, []byte(PAYLOAD))

	// encode trace to stdout as csv
	w := csv.NewWriter(os.Stdout)
	defer w.Flush()
	w.Comma = ';'
	w.Write([]string{"unixmicro", "delta_ms", "step_ms", "label"})
	for _, event := range trace {
		w.Write([]string{
			fmt.Sprint(event.UnixMicro),
			fmt.Sprintf("%.3f", event.Delta),
			fmt.Sprintf("%.3f", event.Step),
			event.Label,
		})
	}
}

// run a traced request and calculate steps and deltas
func run(client *http.Client, payload []byte) (result []TraceEvent) {

	// start time
	start := tracer.Now("client: start")

	// encode payload and post
	response, err := client.Post(BROKER_RUN, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		panic(fmt.Errorf("tsp trace failed: %w", err))
	}
	defer response.Body.Close()

	// response received
	end := tracer.Now("client: response received")

	// decode the response body
	var results []scheduler.ExecResult
	json.NewDecoder(response.Body).Decode(&results)
	btrace := results[0].Result.Trace

	// concatenate traces into one
	trace := make([]tracer.Event, 0, len(btrace)+2)
	trace = append(trace, start)
	trace = append(trace, btrace...)
	trace = append(trace, end)

	// print trace to stdout
	// fmt.Printf("%#v\n", trace)
	// json.NewEncoder(os.Stdout).Encode(trace)

	// calculate steps and deltas
	result = make([]TraceEvent, len(trace))
	t0 := trace[0].Time
	for i, event := range trace {
		r := &result[i]
		r.UnixMicro = event.Time
		r.Label = event.Label
		r.Delta = float64(event.Time-t0) / 1000
		if i == 0 {
			r.Step = 0.0
		} else {
			r.Step = float64(event.Time-trace[i-1].Time) / 1000
		}
	}
	return

}

type TraceEvent struct {
	UnixMicro int64   // the timestamp in unix epoch microseconds
	Delta     float64 // delta from beginning of trace in milliseconds
	Step      float64 // step from previous event in milliseconds
	Label     string  // event label
}
