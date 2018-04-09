package log

import (
	"fmt"
	"io"
	"sync"
)

// MaxWriterBuffer specifies how big the writer buffer can get before
// killing the writer.
var MaxWriterBuffer = 512 * 1024

var log = Logger("eventlog")

// MirrorWriter implements a WriteCloser which syncs incoming bytes to multiple
// [buffered] WriteClosers. They can be added with AddWriter().
type MirrorWriter struct {
	active   bool
	activelk sync.Mutex

	// channel for incoming writers
	writerAdd chan *writerAdd

	// slices of writer/sync-channel pairs
	writers []*bufWriter

	// synchronization channel for incoming writes
	msgSync chan []byte
}

type writerSync struct {
	w  io.WriteCloser
	br chan []byte
}

// NewMirrorWriter initializes and returns a MirrorWriter.
func NewMirrorWriter() *MirrorWriter {
	mw := &MirrorWriter{
		msgSync:   make(chan []byte, 64), // sufficiently large buffer to avoid callers waiting
		writerAdd: make(chan *writerAdd),
	}

	go mw.logRoutine()

	return mw
}

// Write broadcasts the written bytes to all Writers.
func (mw *MirrorWriter) Write(b []byte) (int, error) {
	mycopy := make([]byte, len(b))
	copy(mycopy, b)
	mw.msgSync <- mycopy
	return len(b), nil
}

// Close closes the MirrorWriter
func (mw *MirrorWriter) Close() error {
	// it is up to the caller to ensure that write is not called during or
	// after close is called.
	close(mw.msgSync)
	return nil
}

func (mw *MirrorWriter) doClose() {
	for _, w := range mw.writers {
		w.writer.Close()
	}
}

func (mw *MirrorWriter) logRoutine() {
	// rebind to avoid races on nilling out struct fields
	msgSync := mw.msgSync
	writerAdd := mw.writerAdd

	defer mw.doClose()

	for {
		select {
		case b, ok := <-msgSync:
			if !ok {
				return
			}

			// write to all writers
			dropped := mw.broadcastMessage(b)

			// consolidate the slice
			if dropped {
				mw.clearDeadWriters()
			}
		case wa := <-writerAdd:
			mw.writers = append(mw.writers, newBufWriter(wa.w))

			mw.activelk.Lock()
			mw.active = true
			mw.activelk.Unlock()
			close(wa.done)
		}
	}
}

// broadcastMessage sends the given message to every writer
// if any writer is killed during the send, 'true' is returned
func (mw *MirrorWriter) broadcastMessage(b []byte) bool {
	var dropped bool
	for i, w := range mw.writers {
		_, err := w.Write(b)
		if err != nil {
			mw.writers[i] = nil
			dropped = true
		}
	}
	return dropped
}

func (mw *MirrorWriter) clearDeadWriters() {
	writers := mw.writers
	mw.writers = nil
	for _, w := range writers {
		if w != nil {
			mw.writers = append(mw.writers, w)
		}
	}
	if len(mw.writers) == 0 {
		mw.activelk.Lock()
		mw.active = false
		mw.activelk.Unlock()
	}
}

type writerAdd struct {
	w    io.WriteCloser
	done chan struct{}
}

// AddWriter attaches a new WriteCloser to this MirrorWriter.
// The new writer will start getting any bytes written to the mirror.
func (mw *MirrorWriter) AddWriter(w io.WriteCloser) {
	wa := &writerAdd{
		w:    w,
		done: make(chan struct{}),
	}
	mw.writerAdd <- wa
	<-wa.done
}

// Active returns if there is at least one Writer
// attached to this MirrorWriter
func (mw *MirrorWriter) Active() (active bool) {
	mw.activelk.Lock()
	active = mw.active
	mw.activelk.Unlock()
	return
}

func newBufWriter(w io.WriteCloser) *bufWriter {
	bw := &bufWriter{
		writer:   w,
		incoming: make(chan []byte, 1),
	}

	go bw.loop()
	return bw
}

// writes incoming messages to a buffer and when it fills
// up, writes them to the writer
type bufWriter struct {
	writer io.WriteCloser

	incoming chan []byte

	deathLock sync.Mutex
	dead      bool
}

var errDeadWriter = fmt.Errorf("writer is dead")

func (bw *bufWriter) Write(b []byte) (int, error) {
	bw.deathLock.Lock()
	dead := bw.dead
	bw.deathLock.Unlock()
	if dead {
		if bw.incoming != nil {
			close(bw.incoming)
			bw.incoming = nil
		}
		return 0, errDeadWriter
	}

	bw.incoming <- b
	return len(b), nil
}

func (bw *bufWriter) die() {
	bw.deathLock.Lock()
	bw.dead = true
	bw.writer.Close()
	bw.deathLock.Unlock()
}

func (bw *bufWriter) loop() {
	bufsize := 0
	bufBase := make([][]byte, 0, 16) // some initial memory
	buffered := bufBase
	nextCh := make(chan []byte)

	var nextMsg []byte

	go func() {
		for b := range nextCh {
			_, err := bw.writer.Write(b)
			if err != nil {
				log.Info("eventlog write error: %s", err)
				bw.die()
				return
			}
		}
	}()

	// collect and buffer messages
	incoming := bw.incoming
	for {
		if nextMsg == nil || nextCh == nil {
			// nextCh == nil implies we are 'dead' and draining the incoming channel
			// until the caller notices and closes it for us
			select {
			case b, ok := <-incoming:
				if !ok {
					return
				}
				nextMsg = b
			}
		}

		select {
		case b, ok := <-incoming:
			if !ok {
				return
			}
			bufsize += len(b)
			buffered = append(buffered, b)
			if bufsize > MaxWriterBuffer {
				// if we have too many messages buffered, kill the writer
				bw.die()
				if nextCh != nil {
					close(nextCh)
				}
				nextCh = nil
				// explicity keep going here to drain incoming
			}
		case nextCh <- nextMsg:
			nextMsg = nil
			if len(buffered) > 0 {
				nextMsg = buffered[0]
				buffered = buffered[1:]
				bufsize -= len(nextMsg)
			}

			if len(buffered) == 0 {
				// reset slice position
				buffered = bufBase[:0]
			}
		}
	}
}
