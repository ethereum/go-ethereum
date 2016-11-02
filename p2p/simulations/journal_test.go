package simulations

import (
	// "encoding/json"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/event"
	// "github.com/ethereum/go-ethereum/p2p/discover"
	// "github.com/ethereum/go-ethereum/logger/glog"
)

func testEvents(intervals ...int) (events []*event.Event) {
	t := time.Now()
	for i, interval := range intervals {
		t = t.Add(time.Duration(interval) * time.Millisecond)
		events = append(events, &event.Event{
			Time: t,
			Data: interface{}(&Entry{
				Type:   "node",
				Action: "Off",
				Object: interface{}(i),
			}),
		})
	}
	return events
}

func TestTimedRead(t *testing.T) {
	// keys := []string{
	// 	"aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80aa7cca80",
	// 	"f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3f5ae22c3",
	// }
	// var ids []*discover.NodeID
	// for _, key := range keys {
	// 	id := discover.MustHexID(key)
	// 	ids = append(ids, &id)
	// }

	j := NewJournal()
	intervals := []int{100, 200, 300, 300, 100, 200}
	j.Events = testEvents(intervals...)
	var newTimes []time.Time
	var i int
	acc := 0.5
	length := 4
	f := func(data interface{}) bool {
		_ = data.(*Entry)
		newTimes = append(newTimes, time.Now())
		i++
		return i <= length
	}
	start := time.Now()
	read, err := j.TimedRead(acc, f)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
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

// func TestReplay(t *testing.T) {
// }
