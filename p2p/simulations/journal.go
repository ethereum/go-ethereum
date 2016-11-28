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
	"github.com/ethereum/go-ethereum/p2p/adapters"
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
		time.Sleep(10 * time.Millisecond)
	}
}

func (self *Journal) Read(f func(*event.Event) bool) (read int) {
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
			glog.V(6).Infof("timed read ends (read %v entries)", read)
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
					return empty, nil
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
					return empty, nil
				},
			},
		})
	return self
}

type MockerConfig struct {
	// TODO: frequency/volume etc.
	Id             string
	NodeCount      int
	UpdateInterval time.Duration
}

func NewMockersController(eventer *event.TypeMux) Controller {
	self := NewResourceContoller(
		&ResourceHandlers{
			// Create: n.StartNode, NodeConfig
			Create: &ResourceHandler{
				Handle: func(msg interface{}, parent *ResourceController) (interface{}, error) {
					conf := msg.(*MockerConfig)
					if conf.NodeCount == 0 {
						conf.NodeCount = 100
					}
					ids := RandomNodeIds(conf.NodeCount)
					if conf.UpdateInterval == 0 {
						conf.UpdateInterval = 1 * time.Second
					}
					ticker := time.NewTicker(conf.UpdateInterval)
					go MockEvents(eventer, ids, ticker.C)
					c := NewMockerController(conf, ticker)
					if len(conf.Id) == 0 {
						conf.Id = fmt.Sprintf("%d", parent.id)
					}
					glog.V(6).Infof("new mocker controller on %v", conf.Id)
					if parent != nil {
						parent.SetResource(conf.Id, c)
					}
					parent.id++
					return empty, nil
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
					return empty, nil
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

// MockEvents generates random connectivity events and posts them
// to the eventer
// The journal using the eventer can then be read to visualise or
// drive connections
func MockEvents(eventer *event.TypeMux, ids []*adapters.NodeId, ticker <-chan time.Time) {

	var onNodes []*Node
	offNodes := ids
	onConnsMap := make(map[string]int)
	var onConns []*Conn
	connsMap := make(map[string]int)
	var conns []*Conn
	// ids := RandomNodeIds(100)
	switchonRate := 5
	dropoutRate := 100
	newConnCount := 1 // new connection per node per tick
	connFailRate := 100
	disconnRate := 100 // fraction of all connections
	nodesTarget := len(ids) / 2
	degreeTarget := 8
	convergenceRate := 5
	rounds := 0
	for _ = range ticker {
		glog.V(6).Infof("rates: %v/%v, %v (%v/%v)", switchonRate, dropoutRate, newConnCount, connFailRate, disconnRate)
		// here switchon rate will depend
		nodesUp := len(offNodes) / switchonRate
		missing := nodesTarget - len(onNodes)
		if missing > 0 {
			if nodesUp < missing {
				nodesUp += (missing-nodesUp)/convergenceRate + 1
			}
		}

		nodesDown := len(onNodes) / dropoutRate

		connsUp := len(onNodes) * newConnCount
		connsUp = connsUp - connsUp/connFailRate
		missing = nodesTarget*degreeTarget/2 - len(onConns)
		if missing < connsUp {
			connsUp = missing
			if connsUp < 0 {
				connsUp = 0
			}
		}
		connsDown := len(onConns) / disconnRate
		glog.V(6).Infof("Nodes Up: %v, Down: %v [ON: %v/%v]\nConns Up: %v, Down: %v [ON: %v/%v(%v)]", nodesUp, nodesDown, len(onNodes), len(onNodes)+len(offNodes), connsUp, connsDown, len(onConns), len(conns)-len(onConns), len(conns))

		for i := 0; len(onNodes) > 0 && i < nodesDown; i++ {
			c := rand.Intn(len(onNodes))
			sn := onNodes[c]
			err := eventer.Post(&NodeEvent{
				Type:   "node",
				Action: "down",
				node:   sn,
			})
			if err != nil {
				panic(err.Error())
			}
			onNodes = append(onNodes[0:c], onNodes[c+1:]...)
			offNodes = append(offNodes, sn.Id)
		}
		for i := 0; len(offNodes) > 0 && i < nodesUp; i++ {
			c := rand.Intn(len(offNodes))
			sn := &Node{Id: offNodes[c]}
			err := eventer.Post(&NodeEvent{
				Type:   "node",
				Action: "up",
				node:   sn,
			})
			if err != nil {
				panic(err.Error())
			}
			onNodes = append(onNodes, sn)
			offNodes = append(offNodes[0:c], offNodes[c+1:]...)
		}
		var found bool
		var sc *Conn
		for i := 0; len(onNodes) > 1 && i < connsUp; i++ {
			sc = nil
			n := rand.Intn(len(onNodes) - 1)
			m := n + 1 + rand.Intn(len(onNodes)-n-1)
			for i := m; i < len(onNodes); i++ {
				lab := ConnLabel(onNodes[n].Id, onNodes[i].Id)
				var j int
				j, found = onConnsMap[lab]
				if found {
					continue
				}
				j, found = connsMap[lab]
				if found {
					sc = conns[j]
					break
				}
				caller := onNodes[n].Id
				callee := onNodes[i].Id

				sc := &Conn{
					One:   caller,
					Other: callee,
				}
				connsMap[lab] = len(conns)
				conns = append(conns, sc)
				break
			}

			if sc == nil {
				i--
				continue
			}
			lab := ConnLabel(sc.One, sc.Other)
			onConnsMap[lab] = len(onConns)
			onConns = append(onConns, sc)
			err := eventer.Post(&ConnEvent{
				Type:   "conn",
				Action: "up",
				conn:   sc,
			})
			if err != nil {
				panic(err.Error())
			}
		}

		for i := 0; len(onConns) > 0 && i < connsDown; i++ {
			c := rand.Intn(len(onConns))
			conn := onConns[c]
			onConns = append(onConns[0:c], onConns[c+1:]...)
			lab := ConnLabel(conn.One, conn.Other)
			delete(onConnsMap, lab)
			err := eventer.Post(&ConnEvent{
				Type:   "conn",
				Action: "down",
				conn:   conn,
			})
			if err != nil {
				panic(err.Error())
			}
		}
		rounds++
	}
}
