package eventer

// Basic receiver interface.
type Receiver interface {
	Send(Event)
}

// Receiver as channel
type Channel chan Event

func (self Channel) Send(ev Event) {
	self <- ev
}

// Receiver as function
type Function func(ev Event)

func (self Function) Send(ev Event) {
	self(ev)
}

type Event struct {
	Type string
	Data interface{}
}

type Channels map[string][]Receiver

type EventMachine struct {
	channels Channels
}

func New() *EventMachine {
	return &EventMachine{
		channels: make(Channels),
	}
}

func (self *EventMachine) add(typ string, r Receiver) {
	self.channels[typ] = append(self.channels[typ], r)
}

// Generalised methods for the known receiver types
// * Channel
// * Function
func (self *EventMachine) On(typ string, r interface{}) {
	if eventFunc, ok := r.(func(Event)); ok {
		self.RegisterFunc(typ, eventFunc)
	} else if eventChan, ok := r.(Channel); ok {
		self.RegisterChannel(typ, eventChan)
	} else {
		panic("Invalid type for EventMachine::On")
	}
}

func (self *EventMachine) RegisterChannel(typ string, c Channel) {
	self.add(typ, c)
}

func (self *EventMachine) RegisterFunc(typ string, f Function) {
	self.add(typ, f)
}

func (self *EventMachine) Register(typ string) Channel {
	c := make(Channel, 1)
	self.add(typ, c)

	return c
}

func (self *EventMachine) Post(typ string, data interface{}) {
	if self.channels[typ] != nil {
		ev := Event{typ, data}
		for _, receiver := range self.channels[typ] {
			// Blocking is OK. These are internals and need to be handled
			receiver.Send(ev)
		}
	}
}
