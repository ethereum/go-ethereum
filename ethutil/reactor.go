package ethutil

import (
	"sync"
)

type ReactorEvent struct {
	mut   sync.Mutex
	event string
	chans []chan React
}

// Post the specified reactor resource on the channels
// currently subscribed
func (e *ReactorEvent) Post(react React) {
	e.mut.Lock()
	defer e.mut.Unlock()

	for _, ch := range e.chans {
		go func(ch chan React) {
			ch <- react
		}(ch)
	}
}

// Add a subscriber to this event
func (e *ReactorEvent) Add(ch chan React) {
	e.mut.Lock()
	defer e.mut.Unlock()

	e.chans = append(e.chans, ch)
}

// Remove a subscriber
func (e *ReactorEvent) Remove(ch chan React) {
	e.mut.Lock()
	defer e.mut.Unlock()

	for i, c := range e.chans {
		if c == ch {
			e.chans = append(e.chans[:i], e.chans[i+1:]...)
		}
	}
}

// Basic reactor resource
type React struct {
	Resource interface{}
}

// The reactor basic engine. Acts as bridge
// between the events and the subscribers/posters
type ReactorEngine struct {
	patterns map[string]*ReactorEvent
}

func NewReactorEngine() *ReactorEngine {
	return &ReactorEngine{patterns: make(map[string]*ReactorEvent)}
}

// Subscribe a channel to the specified event
func (reactor *ReactorEngine) Subscribe(event string, ch chan React) {
	ev := reactor.patterns[event]
	// Create a new event if one isn't available
	if ev == nil {
		ev = &ReactorEvent{event: event}
		reactor.patterns[event] = ev
	}

	// Add the channel to reactor event handler
	ev.Add(ch)
}

func (reactor *ReactorEngine) Unsubscribe(event string, ch chan React) {
	ev := reactor.patterns[event]
	if ev != nil {
		ev.Remove(ch)
	}
}

func (reactor *ReactorEngine) Post(event string, resource interface{}) {
	ev := reactor.patterns[event]
	if ev != nil {
		ev.Post(React{Resource: resource})
	}
}
