package ethutil

import "testing"

func TestReactorAdd(t *testing.T) {
	engine := NewReactorEngine()
	ch := make(chan React)
	engine.Subscribe("test", ch)
	if len(engine.patterns) != 1 {
		t.Error("Expected patterns to be 1, got", len(engine.patterns))
	}
}

func TestReactorEvent(t *testing.T) {
	engine := NewReactorEngine()

	// Buffer 1, so it doesn't block for this test
	ch := make(chan React, 1)
	engine.Subscribe("test", ch)
	engine.Post("test", "hello")

	value := <-ch
	if val, ok := value.Resource.(string); ok {
		if val != "hello" {
			t.Error("Expected Resource to be 'hello', got", val)
		}
	} else {
		t.Error("Unable to cast")
	}
}
