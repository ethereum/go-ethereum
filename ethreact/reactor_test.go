package ethreact

import (
	"fmt"
	"testing"
)

func TestReactorAdd(t *testing.T) {
	reactor := New()
	ch := make(chan Event)
	reactor.Subscribe("test", ch)
	if reactor.eventHandlers["test"] == nil {
		t.Error("Expected new eventHandler to be created")
	}
	reactor.Unsubscribe("test", ch)
	if reactor.eventHandlers["test"] != nil {
		t.Error("Expected eventHandler to be removed")
	}
}

func TestReactorEvent(t *testing.T) {
	var name string
	reactor := New()
	// Buffer the channel, so it doesn't block for this test
	cap := 20
	ch := make(chan Event, cap)
	reactor.Subscribe("even", ch)
	reactor.Subscribe("odd", ch)
	reactor.Post("even", "disappears") // should not broadcast if engine not started
	reactor.Start()
	for i := 0; i < cap; i++ {
		if i%2 == 0 {
			name = "even"
		} else {
			name = "odd"
		}
		reactor.Post(name, i)
	}
	reactor.Post("test", cap) // this should not block
	i := 0
	reactor.Flush()
	close(ch)
	for event := range ch {
		fmt.Printf("%d: %v", i, event)
		if i%2 == 0 {
			name = "even"
		} else {
			name = "odd"
		}
		if val, ok := event.Resource.(int); ok {
			if i != val || event.Name != name {
				t.Error("Expected event %d to be of type %s and resource %d, got ", i, name, i, val)
			}
		} else {
			t.Error("Unable to cast")
		}
		i++
	}
	if i != cap {
		t.Error("excpected exactly %d events, got ", i)
	}
	reactor.Stop()
}
