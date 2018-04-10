package msgio

import (
	"io"

	mpool "github.com/libp2p/go-msgio/mpool"
)

// Chan is a msgio duplex channel. It is used to have a channel interface
// around a msgio.Reader or Writer.
type Chan struct {
	MsgChan   chan []byte
	ErrChan   chan error
	CloseChan chan bool
}

// NewChan constructs a Chan with a given buffer size.
func NewChan(chanSize int) *Chan {
	return &Chan{
		MsgChan:   make(chan []byte, chanSize),
		ErrChan:   make(chan error, 1),
		CloseChan: make(chan bool, 2),
	}
}

// ReadFrom wraps the given io.Reader with a msgio.Reader, reads all
// messages, ands sends them down the channel.
func (s *Chan) ReadFrom(r io.Reader) {
	s.readFrom(NewReader(r))
}

// ReadFromWithPool wraps the given io.Reader with a msgio.Reader, reads all
// messages, ands sends them down the channel. Uses given Pool
func (s *Chan) ReadFromWithPool(r io.Reader, p *mpool.Pool) {
	s.readFrom(NewReaderWithPool(r, p))
}

// ReadFrom wraps the given io.Reader with a msgio.Reader, reads all
// messages, ands sends them down the channel.
func (s *Chan) readFrom(mr Reader) {
	// single reader, no need for Mutex
	mr.(*reader).lock = new(nullLocker)

Loop:
	for {
		buf, err := mr.ReadMsg()
		if err != nil {
			if err == io.EOF {
				break Loop // done
			}

			// unexpected error. tell the client.
			s.ErrChan <- err
			break Loop
		}

		select {
		case <-s.CloseChan:
			break Loop // told we're done
		case s.MsgChan <- buf:
			// ok seems fine. send it away
		}
	}

	close(s.MsgChan)
	// signal we're done
	s.CloseChan <- true
}

// WriteTo wraps the given io.Writer with a msgio.Writer, listens on the
// channel and writes all messages to the writer.
func (s *Chan) WriteTo(w io.Writer) {
	// new buffer per message
	// if bottleneck, cycle around a set of buffers
	mw := NewWriter(w)

	// single writer, no need for Mutex
	mw.(*writer).lock = new(nullLocker)
Loop:
	for {
		select {
		case <-s.CloseChan:
			break Loop // told we're done

		case msg, ok := <-s.MsgChan:
			if !ok { // chan closed
				break Loop
			}

			if err := mw.WriteMsg(msg); err != nil {
				if err != io.EOF {
					// unexpected error. tell the client.
					s.ErrChan <- err
				}

				break Loop
			}
		}
	}

	// signal we're done
	s.CloseChan <- true
}

// Close the Chan
func (s *Chan) Close() {
	s.CloseChan <- true
}

// nullLocker conforms to the sync.Locker interface but does nothing.
type nullLocker struct{}

func (l *nullLocker) Lock()   {}
func (l *nullLocker) Unlock() {}
