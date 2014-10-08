package eventer

import (
	"math/rand"
	"testing"
	"time"
)

func TestChannel(t *testing.T) {
	eventer := New()

	c := make(Channel, 1)
	eventer.RegisterChannel("test", c)
	eventer.Post("test", "hello world")

	res := <-c

	if res.Data.(string) != "hello world" {
		t.Error("Expected event with data 'hello world'. Got", res.Data)
	}
}

func TestFunction(t *testing.T) {
	eventer := New()

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
	eventer := New()

	c := eventer.Register("test")
	eventer.Post("test", "hello world")

	res := <-c

	if res.Data.(string) != "hello world" {
		t.Error("Expected event with data 'hello world'. Got", res.Data)
	}
}

func TestOn(t *testing.T) {
	eventer := New()

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

func TestConcurrentUsage(t *testing.T) {
	rand.Seed(time.Now().Unix())
	eventer := New()
	stop := make(chan struct{})
	recv := make(chan int)
	poster := func() {
		for {
			select {
			case <-stop:
				return
			default:
				eventer.Post("test", "hi")
			}
		}
	}
	listener := func(i int) {
		time.Sleep(time.Duration(rand.Intn(99)) * time.Millisecond)
		c := eventer.Register("test")
		// wait for the first event
		<-c
		recv <- i
		// keep receiving to prevent deadlock
		for {
			select {
			case <-stop:
				return
			case <-c:
			}
		}
	}

	nlisteners := 200
	go poster()
	for i := 0; i < nlisteners; i++ {
		go listener(i)
	}
	// wait until everyone has been served
	for i := 0; i < nlisteners; i++ {
		<-recv
	}
	close(stop)
}
