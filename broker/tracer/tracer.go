package tracer

// This package is used to trace the duration of steps in the whole
// offloading process, including communications in the javascript frontend.

import "time"

type Trace struct {
	Events []Event `json:"events" msgpack:"events"`
}

func (t *Trace) Now(label string) {
	t.Events = append(t.Events, Now(label))
}

type Event struct {
	Time  int64  `json:"time" msgpack:"time"`
	Label string `json:"label" msgpack:"label"`
}

func Now(label string) Event {
	return Event{
		Time:  time.Now().UnixMicro(),
		Label: label,
	}
}
