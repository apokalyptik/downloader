package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var (
	eventChan = make(chan Event, 10)
	event     = &Emitter{}
)

type Event struct {
	Type string
	Data interface{}
}

type Emitter struct {
	ctx context.Context
}

func (e *Emitter) handle() {
	for {
		select {
		case data := <-eventChan:
			runtime.EventsEmit(e.ctx, data.Type, data.Data)
		}
	}
}
