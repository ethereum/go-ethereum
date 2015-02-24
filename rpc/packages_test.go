package rpc

import (
	"sync"
	"testing"
	"time"
)

func TestFilterClose(t *testing.T) {
	t.Skip()
	api := &EthereumApi{
		logs:     make(map[int]*logFilter),
		messages: make(map[int]*whisperFilter),
		quit:     make(chan struct{}),
	}

	filterTickerTime = 1
	api.logs[0] = &logFilter{}
	api.messages[0] = &whisperFilter{}
	var wg sync.WaitGroup
	wg.Add(1)
	go api.start()
	go func() {
		select {
		case <-time.After(500 * time.Millisecond):
			api.stop()
			wg.Done()
		}
	}()
	wg.Wait()
	if len(api.logs) != 0 {
		t.Error("expected logs to be empty")
	}

	if len(api.messages) != 0 {
		t.Error("expected messages to be empty")
	}
}
