package ethreact

import (
	"github.com/ethereum/eth-go/ethlog"
	"sync"
)

var logger = ethlog.NewLogger("REACTOR")

type EventHandler struct {
	lock  sync.RWMutex
	name  string
	chans []chan Event
}

// Post the Event with the reactor resource on the channels
// currently subscribed to the event
func (e *EventHandler) Post(event Event) {
	e.lock.RLock()
	defer e.lock.RUnlock()

	// if we want to preserve order pushing to subscibed channels
	// dispatching should be syncrounous
	// this means if subscribed event channel is blocked (closed or has fixed capacity)
	// the reactor dispatch will be blocked, so we need to mitigate by skipping
	// rogue blocking subscribers
	for i, ch := range e.chans {
		select {
		case ch <- event:
		default:
			logger.Warnln("subscribing channel %d to event %s blocked. skipping", i, event.Name)
		}
	}
}

// Add a subscriber to this event
func (e *EventHandler) Add(ch chan Event) {
	e.lock.Lock()
	defer e.lock.Unlock()

	e.chans = append(e.chans, ch)
}

// Remove a subscriber
func (e *EventHandler) Remove(ch chan Event) int {
	e.lock.Lock()
	defer e.lock.Unlock()

	for i, c := range e.chans {
		if c == ch {
			e.chans = append(e.chans[:i], e.chans[i+1:]...)
		}
	}
	return len(e.chans)
}

// Basic reactor resource
type Event struct {
	Resource interface{}
	Name     string
}

// The reactor basic engine. Acts as bridge
// between the events and the subscribers/posters
type ReactorEngine struct {
	lock            sync.RWMutex
	eventChannel    chan Event
	eventHandlers   map[string]*EventHandler
	quit            chan bool
	shutdownChannel chan bool
	running         bool
	drained         bool
}

func New() *ReactorEngine {
	return &ReactorEngine{
		eventHandlers:   make(map[string]*EventHandler),
		eventChannel:    make(chan Event),
		quit:            make(chan bool, 1),
		shutdownChannel: make(chan bool, 1),
	}
}

func (reactor *ReactorEngine) Start() {
	reactor.lock.Lock()
	defer reactor.lock.Unlock()
	if !reactor.running {
		go func() {
		out:
			for {
				select {
				case <-reactor.quit:
					break out
				case event := <-reactor.eventChannel:
					// needs to be called syncronously to keep order of events
					reactor.dispatch(event)
				default:
					reactor.drained = true
				}
			}
			reactor.lock.Lock()
			defer reactor.lock.Unlock()
			reactor.running = false
			logger.Infoln("stopped")
			close(reactor.shutdownChannel)
		}()
		reactor.running = true
		logger.Infoln("started")
	}
}

func (reactor *ReactorEngine) Stop() {
	reactor.lock.RLock()
	if reactor.running {
		reactor.quit <- true
	}
	reactor.lock.RUnlock()
	<-reactor.shutdownChannel
}

func (reactor *ReactorEngine) Flush() {
	for !reactor.drained {
	}
}

// Subscribe a channel to the specified event
func (reactor *ReactorEngine) Subscribe(event string, eventChannel chan Event) {
	reactor.lock.Lock()
	defer reactor.lock.Unlock()

	eventHandler := reactor.eventHandlers[event]
	// Create a new event handler if one isn't available
	if eventHandler == nil {
		eventHandler = &EventHandler{name: event}
		reactor.eventHandlers[event] = eventHandler
	}
	// Add the events channel to reactor event handler
	eventHandler.Add(eventChannel)
	logger.Debugln("added new subscription to %s", event)
}

func (reactor *ReactorEngine) Unsubscribe(event string, eventChannel chan Event) {
	reactor.lock.Lock()
	defer reactor.lock.Unlock()

	eventHandler := reactor.eventHandlers[event]
	if eventHandler != nil {
		len := eventHandler.Remove(eventChannel)
		if len == 0 {
			reactor.eventHandlers[event] = nil
		}
		logger.Debugln("removed subscription to %s", event)
	}
}

func (reactor *ReactorEngine) Post(event string, resource interface{}) {
	reactor.lock.Lock()
	defer reactor.lock.Unlock()

	if reactor.running {
		reactor.drained = false
		reactor.eventChannel <- Event{Resource: resource, Name: event}
	}
}

func (reactor *ReactorEngine) dispatch(event Event) {
	name := event.Name
	eventHandler := reactor.eventHandlers[name]
	// if no subscriptions to this event type - no event handler created
	// then noone to notify
	if eventHandler != nil {
		// needs to be called syncronously
		eventHandler.Post(event)
	}
}
