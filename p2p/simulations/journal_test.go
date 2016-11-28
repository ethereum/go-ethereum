package simulations

import (
	"encoding/json"
	"io/ioutil"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/p2p/adapters"
)

func testEvents(intervals ...int) (events []*event.Event) {
	t := time.Now()
	for _, interval := range intervals {
		t = t.Add(time.Duration(interval) * time.Millisecond)
		events = append(events, &event.Event{
			Time: t,
			Data: interface{}(&NodeEvent{
				Type:   "node",
				Action: "down",
			}),
		})
	}
	return events
}

func TestTimedRead(t *testing.T) {
	j := NewJournal()
	intervals := []int{100, 200, 300, 300, 100, 200}
	j.Events = testEvents(intervals...)
	var newTimes []time.Time
	var i int
	acc := 0.5
	length := 4
	f := func(data interface{}) bool {
		_ = data.(*NodeEvent)
		newTimes = append(newTimes, time.Now())
		i++
		return i <= length
	}
	start := time.Now()
	read := j.TimedRead(acc, f)
	if read != 5 {
		t.Fatalf("incorrect number of events read: expected 5, got %v", read)
	}
	for i, ti := range newTimes {
		expInt := time.Duration(acc*float64(intervals[i])) * time.Millisecond
		gotInt := ti.Sub(start)
		if gotInt-expInt > 1*time.Millisecond {
			t.Fatalf("journal timed read incorrect interval: expected %v ,got %v", expInt, gotInt)
		}
		start = ti
	}
}

func testIDs() (ids []*adapters.NodeId) {

	keys := []string{
		"aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80",
		"f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3",
	}
	for _, key := range keys {
		id := adapters.NewNodeIdFromHex(key)
		ids = append(ids, id)
	}
	return ids
}

func testJournal(ids []*adapters.NodeId) *Journal {
	eventer := &event.TypeMux{}
	journal := NewJournal()
	journal.Subscribe(eventer, ConnectivityEvents...)
	mockNewNodes(eventer, ids)
	journal.WaitEntries(len(ids))
	return journal
}

func TestSubscribe(t *testing.T) {
	ids := testIDs()
	journal := testJournal(ids)
	for i, ev := range journal.Events {
		id := ev.Data.(*NodeEvent).node.Id
		if id != ids[i] {
			t.Fatalf("incorrect id: expected %v, got %v", id, ids[i])
		}
	}
}

func loadTestJournal(t *testing.T) ([]byte, *Journal) {
	b, err := ioutil.ReadFile("./testjournal.json")
	if err != nil {
		t.Fatalf("unexpected error reading test journal json: %v", err)
	}
	journal, err := NewJournalFromJSON(b)
	if err != nil {
		t.Fatalf("unexpected error decoding journal json: %v", err)
	}
	return b, journal
}

func TestLoadSave(t *testing.T) {
	b, j := loadTestJournal(t)

	jo, err := json.MarshalIndent(j, "", " ")
	if err != nil {
		t.Fatalf("unexpected error encoding journal for %v: %v", j, err)
	}
	expJSON := string(b)
	gotJSON := string(jo)
	if expJSON != gotJSON {
		t.Fatalf("incorrect json for journal: expected %v, got %v", expJSON, gotJSON)
	}
}

func TestReplay(t *testing.T) {
	_, jo := loadTestJournal(t)
	eventer := &event.TypeMux{}

	journal := NewJournal()
	journal.Subscribe(eventer, ConnectivityEvents...)

	Replay(0, jo, eventer)
	for i, ev := range jo.Events {
		exp := ev.Data.(*NodeEvent).String()
		got := journal.Events[i].Data.(*NodeEvent).String()
		if exp != got {
			t.Fatalf("incorrent replayed journal entry at pos %v: expected %v, got %v", i, exp, got)
		}
	}
	// ids := RandomNodeIds(7)
	// ticker := time.NewTicker(1000 * time.Microsecond)
	// go MockEvents(eventer, ids, ticker.C)
	// journal.WaitEntries(20)
	// // eventer.Stop()	// eventer = &event.TypeMux{}
	// journal = NewJournal()
	// journal.Subscribe(eventer, &Entry{})

	// func TestReplay(t *testing.T) {
	// }
}
