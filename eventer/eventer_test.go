package eventer

import "testing"

func TestChannel(t *testing.T) {
	eventer := New(nil)

	c := make(Channel, 1)
	eventer.RegisterChannel("test", c)
	eventer.Post("test", "hello world")

	res := <-c

	if res.Data.(string) != "hello world" {
		t.Error("Expected event with data 'hello world'. Got", res.Data)
	}
}

func TestFunction(t *testing.T) {
	eventer := New(nil)

	var data string
	eventer.RegisterFunc("test", func(ev Event) {
		data = ev.Data.(string)
	})
	eventer.Post("test", "hello world")

	if data != "hello world" {
		t.Error("Expected event with data 'hello world'. Got", data)
	}
}

func TestRegister(t *testing.T) {
	eventer := New(nil)

	c := eventer.Register("test")
	eventer.Post("test", "hello world")

	res := <-c

	if res.Data.(string) != "hello world" {
		t.Error("Expected event with data 'hello world'. Got", res.Data)
	}
}

func TestOn(t *testing.T) {
	eventer := New(nil)

	c := make(Channel, 1)
	eventer.On("test", c)

	var data string
	eventer.On("test", func(ev Event) {
		data = ev.Data.(string)
	})
	eventer.Post("test", "hello world")

	res := <-c
	if res.Data.(string) != "hello world" {
		t.Error("Expected channel event with data 'hello world'. Got", res.Data)
	}

	if data != "hello world" {
		t.Error("Expected function event with data 'hello world'. Got", data)
	}
}
