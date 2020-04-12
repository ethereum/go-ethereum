package splunk

import (
	"sync"
	"time"
)

const (
	bufferSize       = 10000
	defaultInterval  = 5 * time.Second
	defaultThreshold = 100
	defaultRetries   = 0
)

type Writer struct {
	Client *Client
	// How often the write buffer should be flushed to splunk
	FlushInterval time.Duration
	// How many Write()'s before buffer should be flushed to splunk
	FlushThreshold int
	// Max number of retries we should do when we flush the buffer
	MaxRetries int
	dataChan   chan *message
	errors     chan error
	once       sync.Once
}

type message struct {
	data      string
	writtenAt time.Time
}

func (w *Writer) initialize() {
	w.once.Do(func() {
		w.dataChan = make(chan *message, bufferSize)
		w.errors = make(chan error, bufferSize)
		go w.listen()
	})
}

func (w *Writer) Write(b []byte) (int, error) {
	w.initialize()
	w.dataChan <- &message{
		data:      string(b),
		writtenAt: time.Now(),
	}
	return len(b), nil
}

func (w *Writer) Errors() <-chan error {
	w.initialize()
	return w.errors
}

func (w *Writer) listen() {
	if w.FlushInterval <= 0 {
		w.FlushInterval = defaultInterval
	}
	if w.FlushThreshold == 0 {
		w.FlushThreshold = defaultThreshold
	}
	if w.MaxRetries == 0 {
		w.MaxRetries = defaultRetries
	}
	ticker := time.NewTicker(w.FlushInterval)
	buffer := make([]*message, 0)

	flush := func() {
		go w.send(buffer, w.MaxRetries)
		buffer = make([]*message, 0)
	}
	for {
		select {
		case <-ticker.C:
			if len(buffer) > 0 {
				flush()
			}
		case d := <-w.dataChan:
			buffer = append(buffer, d)
			if len(buffer) > w.FlushThreshold {
				flush()
			}
		}
	}
}

func (w *Writer) send(messages []*message, retries int) {
	events := make([]*Event, len(messages))
	for i, m := range messages {
		events[i] = w.Client.NewEventWithTime(m.writtenAt.Unix(), m.data, w.Client.Source, w.Client.SourceType, w.Client.Index)
	}
	err := w.Client.LogEvents(events)
	if err != nil {
		for i := 0; i < retries; i++ {
			err = w.Client.LogEvents(events)
			if err == nil {
				return
			}
		}
		select {
		case w.errors <- err:
		default:
		}
	}
}
