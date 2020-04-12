package splunk

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestWriter_Write(t *testing.T) {
	numWrites := 1000
	numMessages := 0
	lock := sync.Mutex{}
	notify := make(chan bool, numWrites)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := ioutil.ReadAll(r.Body)
		split := strings.Split(string(b), "\n")
		num := 0
		// Since we batch our logs up before we send them:
		// Increment our messages counter by one for each JSON object we got in this response
		// We don't know how many responses we'll get, we only care about the number of messages
		for _, line := range split {
			if strings.HasPrefix(line, "{") {
				num++
				notify <- true
			}
		}
		lock.Lock()
		numMessages = numMessages + num
		lock.Unlock()
	}))

	// Create a writer that's flushing constantly. We want this test to run
	// quickly
	writer := Writer{
		Client:        NewClient(server.Client(), server.URL, "", "", "", ""),
		FlushInterval: 1 * time.Millisecond,
	}
	// Send a bunch of messages in separate goroutines to make sure we're properly
	// testing Writer's concurrency promise
	for i := 0; i < numWrites; i++ {
		go writer.Write([]byte(fmt.Sprintf("%d", i)))
	}
	// To notify our test we've collected everything we need.
	doneChan := make(chan bool)
	go func() {
		for i := 0; i < numWrites; i++ {
			// Do nothing, just loop through to the next one
			<-notify
		}
		doneChan <- true
	}()
	select {
	case <-doneChan:
		// Do nothing, we're good
	case <-time.After(1 * time.Second):
		t.Errorf("Timed out waiting for messages")
	}
	// We may have received more than numWrites amount of messages, check that case
	if numMessages != numWrites {
		t.Errorf("Didn't get the right number of messages, expected %d, got %d", numWrites, numMessages)
	}
}

func TestWriter_Errors(t *testing.T) {
	numMessages := 1000
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "bad request")
	}))
	writer := Writer{
		Client: NewClient(server.Client(), server.URL, "", "", "", ""),
		// Will flush after the last message is sent
		FlushThreshold: numMessages - 1,
		// Don't let the flush interval cause raciness
		FlushInterval: 5 * time.Minute,
	}
	for i := 0; i < numMessages; i++ {
		_, _ = writer.Write([]byte("some data"))
	}
	select {
	case <-writer.Errors():
		// good to go, got our error
	case <-time.After(1 * time.Second):
		t.Errorf("Timed out waiting for error, should have gotten 1 error")
	}
}
