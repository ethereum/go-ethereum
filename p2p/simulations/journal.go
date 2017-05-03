package simulations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

// Journal is an instance of a guaranteed no-loss subscription to network related events
// (using event.TypeMux). Network components POST events to the TypeMux, which then is
// read by the journal. Each journal belongs to a subscription.
type Journal struct {
	Id      string
	lock    sync.Mutex
	counter int
	cursor  int
	quitc   chan bool
	Events  []*event.TypeMuxEvent
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
	log.Info("subscribe")
	sub := eventer.Subscribe(types...)
	go func() {
		defer sub.Unsubscribe()
		for {
			select {
			case ev := <-sub.Chan():
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

func (self *Journal) append(evs ...*event.TypeMuxEvent) {
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
		time.Sleep(10 * time.Millisecond)
	}
}

func (self *Journal) Read(f func(*event.TypeMuxEvent) bool) (read int) {
	self.lock.Lock()
	defer self.lock.Unlock()
	ok := true
	for self.cursor < len(self.Events) && ok {
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
	return read
}

// TimedRead reads the events but blocks for intervals that correspond to
// the original time intervals,
// NOTE: the events' timestamps are supposed to be strictly ordered otherwise
// the call panics.
// acc is an acceleration factor
func (self *Journal) TimedRead(acc float64, f func(interface{}) bool) (read int) {
	var lastEvent time.Time
	timer := time.NewTimer(0)
	var data interface{}
	h := func(ev *event.TypeMuxEvent) bool {
		// wait for the interval time passes event time
		if ev.Time.Before(lastEvent) {
			panic("events not ordered")
		}
		interval := ev.Time.Sub(lastEvent)
		log.Trace(fmt.Sprintf("reset timer to interval %v", interval))
		timer.Reset(time.Duration(acc) * interval)
		lastEvent = ev.Time
		data = ev.Data
		return false
	}
	var n int
	for {
		// Read blocks for the iteration. need to read one event at a time so that
		// waiting for the timer to go off does not block concurrent access to the journal
		n = self.Read(h)
		if read > 0 && n > 0 {
			select {
			case <-self.quitc:
				break
			case <-timer.C:
			}
		}
		read += n
		if n == 0 || !f(data) {
			log.Trace(fmt.Sprintf("timed read ends (read %v entries)", read))
			break
		}
	}
	return read
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
	log.Trace(fmt.Sprintf("cursor reset from %v to %v/%v (%v)", self.cursor, n, len(self.Events), self.counter))
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

func ConnLabel(source, target *adapters.NodeId) string {
	var first, second *adapters.NodeId
	if bytes.Compare(source.Bytes(), target.Bytes()) > 0 {
		first = target
		second = source
	} else {
		first = source
		second = target
	}
	return fmt.Sprintf("%v-%v", first, second)
}
