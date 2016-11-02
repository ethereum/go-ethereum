package simulations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger/glog"
)

// Journal is an instance of a guaranteed no-loss subscription using event.TypeMux
// Network components POST events to the TypeMux, which then is read by the journal
// Each journal belongs to a subscription
type Journal struct {
	Id      string
	lock    sync.Mutex
	counter int
	cursor  int
	quitc   chan bool
	Events  []*event.Event
}

// NewJournal constructor
// Journal can get input events from subscriptions, add event logs
// or scheduled replay of events from another journal
//
// see the Read and TimedRead iterators for use
// the Journal is safe for concurrent reads and writes
func NewJournal() *Journal {
	return &Journal{quitc: make(chan bool)}
}

// Subscribe takes an event.TypeMux and subscibes to types
// and launches a gorourine that appends any new event to the event log
// used for journalling history of a network
// the goroutine terminates when the journal is closed
func (self *Journal) Subscribe(eventer *event.TypeMux, types ...interface{}) {
	glog.V(6).Infof("subscribe")
	sub := eventer.Subscribe(types...)
	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case ev := <-sub.Chan():
				glog.V(6).Infof("appebd ev %v", ev)
				self.append(ev)
			case <-self.quitc:
				return
			}
		}
	}()
}

// AddJournal appends the event log of another journal to the receiver's one
func (self *Journal) AddJournal(j *Journal) {
	self.append(j.Events...)
}

// NewJournalFromJSON decodes a JSON serialised events log
// into a journal struct
// used to replay recorded history
func NewJournalFromJSON(b []byte) (*Journal, error) {
	self := NewJournal()
	err := json.Unmarshal(b, self)
	if err != nil {
		return nil, err
	}
	return self, nil
}

// Replay replays the events of another journal preserving (relative) timing of events
// params:
// * acc: using acceleration factor acc
// * journal: journal to use
// * eventer: where to post the replayed events
func Replay(acc float64, j *Journal, eventer *event.TypeMux) {
	f := func(d interface{}) bool {
		// reposts the data with the eventer (the data receives a new timestamp)
		eventer.Post(d)
		return true
	}
	j.TimedRead(acc, f)
}

// Snapshot creates a snapshot out of the journal
// this is simply done by reading the event log backwards and mark the last action
// on a node/connection ignoring all earlier mentions
// TODO: implmented
func Snapshot(conf *SnapshotConfig, j *Journal) (*Journal, error) {
	return nil, fmt.Errorf("snapshot not implemented")
}

func (self *Journal) Close() {
	close(self.quitc)
}

func (self *Journal) append(evs ...*event.Event) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.Events = append(self.Events, evs...)
	self.counter++
}

func (self *Journal) NewEntries() int {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.counter - self.cursor
}

func (self *Journal) WaitEntries(n int) {
	for self.NewEntries() < n {
		glog.V(6).Infof(".")
	}
}

func (self *Journal) Read(f func(*event.Event) bool) (read int, err error) {
	self.lock.Lock()
	defer self.lock.Unlock()
	glog.V(6).Infof("read out of %v", len(self.Events))
	ok := true
	for self.cursor < len(self.Events) && ok {
		glog.V(6).Infof("read %v", read)
		read++
		ok = f(self.Events[self.cursor])
		self.cursor++
		select {
		case <-self.quitc:
			break
		default:
		}
	}
	self.reset(self.cursor)
	return read, nil
}

// TimedRead reads the events but blocks for intervals that correspond to
// the original time intervals,
// NOTE: the events' timestamps are supposed to be strictly ordered otherwise
// the call panics.
// acc is an acceleration factor
func (self *Journal) TimedRead(acc float64, f func(interface{}) bool) (read int, err error) {
	var lastEvent time.Time
	timer := time.NewTimer(0)
	var data interface{}
	h := func(ev *event.Event) bool {
		// wait for the interval time passes event time
		if ev.Time.Before(lastEvent) {
			panic("events not ordered")
		}
		interval := ev.Time.Sub(lastEvent)
		glog.V(6).Infof("reset timer to interval %v", interval)
		timer.Reset(time.Duration(acc) * interval)
		lastEvent = ev.Time
		data = ev.Data
		return false
	}
	var n int
	for {
		// Read blocks for the iteration. need to read one event at a time so that
		// waiting for the timer to go off does not block concurrent access to the journal
		n, err = self.Read(h)
		if read > 0 {
			select {
			case <-self.quitc:
				break
			case <-timer.C:
			}
		}
		read += n
		if err != nil || n == 0 || !f(data) {
			break
		}
	}
	return read, err
}

func (self *Journal) Reset(n int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	self.reset(n)
}

func (self *Journal) reset(n int) {
	length := len(self.Events)
	if length == 0 {
		return
	}
	if n >= length-1 {
		n = length - 1
	}
	glog.V(6).Infof("cursor reset from %v to %v/%v (%v)", self.cursor, n, len(self.Events), self.counter)
	self.Events = self.Events[self.cursor:]
	self.cursor = 0
}

func (self *Journal) Counter() int {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.counter
}

// type History()

func (self *Journal) Cursor() int {
	self.lock.Lock()
	defer self.lock.Unlock()
	return self.cursor
}

type SnapshotConfig struct {
	Id string
}

type JournalPlayConfig struct {
	Id      string
	SpeedUp float64
	Journal *Journal
	Events  []string
}

func NewJournalPlayersController(eventer *event.TypeMux) Controller {
	self := NewResourceContoller(
		&ResourceHandlers{
			// POST /o/players/
			Create: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					conf := msg.(*JournalPlayConfig)
					go Replay(conf.SpeedUp, conf.Journal, eventer)
					c := NewJournalPlayerController(conf)
					parent.SetResource(conf.Id, c)
					return nil, nil
				},
				Type: reflect.TypeOf(&JournalPlayConfig{}),
			},
		})
	return self
}

func NewJournalPlayerController(conf *JournalPlayConfig) Controller {
	self := NewResourceContoller(
		&ResourceHandlers{
			// GET /0/players/<playerId>
			Retrieve: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					return nil, fmt.Errorf("info about journal player not implemented")
				},
			},
			// DELETE /0/players/<playerId>
			Destroy: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					conf.Journal.Close() // terminate Replay-> TimedRead routine
					parent.DeleteResource(conf.Id)
					return nil, nil
				},
			},
		})
	return self
}

type MockerConfig struct {
	// TODO: frequency/volume etc.
	Id             string
	UpdateInterval time.Duration
}

func NewMockersController(eventer *event.TypeMux) Controller {
	self := NewResourceContoller(
		&ResourceHandlers{
			// Create: n.StartNode, NodeConfig
			Create: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					conf := msg.(*MockerConfig)
					ticker := time.NewTicker(conf.UpdateInterval)
					go MockEvents(eventer, ticker.C)
					c := NewMockerController(conf, ticker)
					parent.SetResource(conf.Id, c)
					return nil, nil
				},
				Type: reflect.TypeOf(&MockerConfig{}),
			},
		})
	return self
}

func NewMockerController(conf *MockerConfig, ticker *time.Ticker) Controller {
	self := NewResourceContoller(
		&ResourceHandlers{
			// GET /0/mockevents/<mockerId>
			Retrieve: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					return nil, fmt.Errorf("info about mocker not implemented")
				},
			},
			// DELETE /0/mockevents/<mockerId>
			Destroy: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					ticker.Stop() //terminate MockEvents routine
					parent.DeleteResource(conf.Id)
					return nil, nil
				},
			},
		})
	return self
}

// deltas: changes in the number of cumulative actions: non-negative integers.
// base unit is the fixed minimal interval  between two measurements (time quantum)
// acceleration : to slow down you just set the base unit higher.
// to speed up: skip x number of base units
// frequency: given as the (constant or average) number of base units between measurements
// if resolution is expressed as the inverse of frequency  = preserved information
// setting the acceleration
// beginning of the record (lifespan) of the network is index 0
// acceleration means that snapshots are rarer so the same span can be generated by the journal
// then update logs can be compressed (toonly one state transition per affected node)
// epoch, epochcount

type Delta struct {
	On  int
	Off int
}

func oneOutOf(n int) int {
	t := rand.Intn(n)
	if t == 0 {
		return 1
	}
	return 0
}

func deltas(i int) (d []*Delta) {
	if i == 0 {
		return []*Delta{
			&Delta{10, 0},
			&Delta{20, 0},
		}
	}
	return []*Delta{
		&Delta{oneOutOf(10), oneOutOf(10)},
		&Delta{oneOutOf(2), oneOutOf(2)},
	}
}

// MockEvents generates random connectivity events and posts them
// to the eventer
// The journal using the eventer can then be read to visualise or
// drive connections
func MockEvents(eventer *event.TypeMux, ticker <-chan time.Time) {
	ids := RandomNodeIDs(100)
	var onNodes []*SimNode
	offNodes := ids
	var onConns []*SimConn

	n := 0
	for _ = range ticker {
		ds := deltas(n)
		for i := 0; len(offNodes) > 0 && i < ds[0].On; i++ {
			c := rand.Intn(len(offNodes))
			sn := &SimNode{ID: offNodes[c]}
			err := eventer.Post(&Entry{
				Type:   "Node",
				Action: "On",
				Object: sn,
			})
			if err != nil {
				panic(err.Error())
			}
			onNodes = append(onNodes, sn)
			offNodes = append(offNodes[0:c], offNodes[c+1:]...)
		}
		for i := 0; len(onNodes) > 0 && i < ds[0].Off; i++ {
			c := rand.Intn(len(onNodes))
			sn := onNodes[c]
			err := eventer.Post(&Entry{
				Type:   "Node",
				Action: "Off",
				Object: sn,
			})
			if err != nil {
				panic(err.Error())
			}
			onNodes = append(onNodes[0:c], onNodes[c+1:]...)
			offNodes = append(offNodes, sn.ID)
		}
		for i := 0; len(onNodes) > 1 && i < ds[1].On; i++ {
			caller := onNodes[rand.Intn(len(onNodes))].ID
			callee := onNodes[rand.Intn(len(onNodes))].ID
			if bytes.Compare(caller[:], callee[:]) >= 0 {
				i--
				continue
			}
			sc := &SimConn{
				Caller: caller,
				Callee: callee,
			}
			err := eventer.Post(&Entry{
				Type:   "Conn",
				Action: "On",
				Object: sc,
			})
			if err != nil {
				panic(err.Error())
			}
			onConns = append(onConns, sc)
		}
		for i := 0; len(onConns) > 0 && i < ds[1].Off; i++ {
			c := rand.Intn(len(onConns))
			err := eventer.Post(&Entry{
				Type:   "Conn",
				Action: "Off",
				Object: onConns[c],
			})
			if err != nil {
				panic(err.Error())
			}
			onConns = append(onConns[0:c], onConns[c+1:]...)
		}
		n++
	}

}
