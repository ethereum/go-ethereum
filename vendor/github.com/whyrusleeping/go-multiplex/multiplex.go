package multiplex

import (
	"bufio"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	logging "github.com/ipfs/go-log"
	mpool "github.com/libp2p/go-msgio/mpool"
)

var log = logging.Logger("multiplex")

var MaxMessageSize = 1 << 20

// Max time to block waiting for a slow reader to read from a stream before
// resetting it. Preferably, we'd have some form of back-pressure mechanism but
// we don't have that in this protocol.
var ReceiveTimeout = 5 * time.Second

// ErrShutdown is returned when operating on a shutdown session
var ErrShutdown = errors.New("session shut down")

// ErrTwoInitiators is returned when both sides think they're the initiator
var ErrTwoInitiators = errors.New("two initiators")

// ErrInvalidState is returned when the other side does something it shouldn't.
// In this case, we close the connection to be safe.
var ErrInvalidState = errors.New("received an unexpected message from the peer")

// +1 for initiator
const (
	NewStream = 0
	Message   = 1
	Close     = 3
	Reset     = 5
)

type Multiplex struct {
	con       net.Conn
	buf       *bufio.Reader
	nextID    uint64
	initiator bool

	closed       chan struct{}
	shutdown     chan struct{}
	shutdownErr  error
	shutdownLock sync.Mutex

	wrLock sync.Mutex

	nstreams chan *Stream
	hdrBuf   []byte

	channels map[uint64]*Stream
	chLock   sync.Mutex
}

func NewMultiplex(con net.Conn, initiator bool) *Multiplex {
	mp := &Multiplex{
		con:       con,
		initiator: initiator,
		buf:       bufio.NewReader(con),
		channels:  make(map[uint64]*Stream),
		closed:    make(chan struct{}),
		shutdown:  make(chan struct{}),
		nstreams:  make(chan *Stream, 16),
		hdrBuf:    make([]byte, 20),
	}

	go mp.handleIncoming()

	return mp
}

func (mp *Multiplex) newStream(id uint64, name string, initiator bool) *Stream {
	var i uint64
	if initiator {
		i = 1
	}
	return &Stream{
		id:        id,
		name:      name,
		initiator: i,
		dataIn:    make(chan []byte, 8),
		reset:     make(chan struct{}),
		mp:        mp,
	}
}

func (m *Multiplex) Accept() (*Stream, error) {
	select {
	case s, ok := <-m.nstreams:
		if !ok {
			return nil, errors.New("multiplex closed")
		}
		return s, nil
	case <-m.closed:
		return nil, m.shutdownErr
	}
}

func (mp *Multiplex) Close() error {
	mp.closeNoWait()

	// Wait for the receive loop to finish.
	<-mp.closed

	return nil
}

func (mp *Multiplex) closeNoWait() {
	mp.shutdownLock.Lock()
	select {
	case <-mp.shutdown:
	default:
		mp.con.Close()
		close(mp.shutdown)
	}
	mp.shutdownLock.Unlock()
}

func (mp *Multiplex) IsClosed() bool {
	select {
	case <-mp.closed:
		return true
	default:
		return false
	}
}

func (mp *Multiplex) sendMsg(header uint64, data []byte, dl time.Time) error {
	mp.wrLock.Lock()
	defer mp.wrLock.Unlock()
	if !dl.IsZero() {
		if err := mp.con.SetWriteDeadline(dl); err != nil {
			return err
		}
	}
	n := binary.PutUvarint(mp.hdrBuf, header)
	n += binary.PutUvarint(mp.hdrBuf[n:], uint64(len(data)))
	_, err := mp.con.Write(mp.hdrBuf[:n])
	if err != nil {
		return err
	}

	if len(data) != 0 {
		_, err = mp.con.Write(data)
		if err != nil {
			return err
		}
	}
	if !dl.IsZero() {
		if err := mp.con.SetWriteDeadline(time.Time{}); err != nil {
			return err
		}
	}

	return nil
}

func (mp *Multiplex) nextChanID() (out uint64) {
	if mp.initiator {
		out = mp.nextID + 1
	} else {
		out = mp.nextID
	}
	mp.nextID += 2
	return
}

func (mp *Multiplex) NewStream() (*Stream, error) {
	return mp.NewNamedStream("")
}

func (mp *Multiplex) NewNamedStream(name string) (*Stream, error) {
	mp.chLock.Lock()

	// We could call IsClosed but this is faster (given that we already have
	// the lock).
	if mp.channels == nil {
		return nil, ErrShutdown
	}

	sid := mp.nextChanID()
	header := (sid << 3) | NewStream

	if name == "" {
		name = fmt.Sprint(sid)
	}
	s := mp.newStream(sid, name, true)
	mp.channels[sid] = s
	mp.chLock.Unlock()

	err := mp.sendMsg(header, []byte(name), time.Time{})
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (mp *Multiplex) cleanup() {
	mp.closeNoWait()
	mp.chLock.Lock()
	defer mp.chLock.Unlock()
	for _, msch := range mp.channels {
		msch.clLock.Lock()
		if !msch.closedRemote {
			msch.closedRemote = true
			// Cancel readers
			close(msch.reset)
		}
		msch.closedLocal = true
		msch.clLock.Unlock()
	}
	// Don't remove this nil assignment. We check if this is nil to check if
	// the connection is closed when we already have the lock (faster than
	// checking if the stream is closed).
	mp.channels = nil
	if mp.shutdownErr == nil {
		mp.shutdownErr = ErrShutdown
	}
	close(mp.closed)
}

func (mp *Multiplex) handleIncoming() {
	defer mp.cleanup()

	recvTimeout := time.NewTimer(0)
	defer recvTimeout.Stop()

	if !recvTimeout.Stop() {
		<-recvTimeout.C
	}

	for {
		ch, tag, err := mp.readNextHeader()
		if err != nil {
			mp.shutdownErr = err
			return
		}

		b, err := mp.readNext()
		if err != nil {
			mp.shutdownErr = err
			return
		}

		mp.chLock.Lock()
		msch, ok := mp.channels[ch]
		mp.chLock.Unlock()

		// Why the +1s? 1 is added to the tag if the message is sent by an initiator.
		// While we don't actually care about which side is the
		// initiator (why should we?) it's the spec...
		switch tag {
		case NewStream:
			if mp.initiator == ((ch & 0x1) == 1) {
				mp.shutdownErr = ErrTwoInitiators
				return
			}

			if ok {
				log.Debugf("received NewStream message for existing stream: %d", ch)
				mp.shutdownErr = ErrInvalidState
				return
			}

			name := string(b)
			msch = mp.newStream(ch, name, false)
			mp.chLock.Lock()
			mp.channels[ch] = msch
			mp.chLock.Unlock()
			select {
			case mp.nstreams <- msch:
			case <-mp.shutdown:
				return
			}

		case Reset, Reset + 1:
			if !ok {
				// This is *ok*. We forget the stream on reset.
				continue
			}
			msch.clLock.Lock()

			// Honestly, this check should never be true... It means we've leaked.
			// However, this is an error on *our* side so we shouldn't just bail.
			if msch.closedLocal && msch.closedRemote {
				msch.clLock.Unlock()
				log.Errorf("leaked a completely closed stream")
				continue
			}

			if !msch.closedRemote {
				close(msch.reset)
			}
			msch.closedRemote = true
			msch.closedLocal = true
			msch.clLock.Unlock()

			mp.chLock.Lock()
			delete(mp.channels, ch)
			mp.chLock.Unlock()
		case Close, Close + 1:
			if !ok {
				continue
			}

			msch.clLock.Lock()

			if msch.closedRemote {
				msch.clLock.Unlock()
				// Technically a bug on the other side. We
				// should consider killing the connection.
				continue
			}

			close(msch.dataIn)
			msch.closedRemote = true

			cleanup := msch.closedLocal

			msch.clLock.Unlock()

			if cleanup {
				mp.chLock.Lock()
				delete(mp.channels, ch)
				mp.chLock.Unlock()
			}
		case Message, Message + 1:
			if !ok {
				// This is a perfectly valid case when we reset
				// and forget about the stream.
				log.Debugf("message for non-existant stream, dropping data: %d", ch)
				// Guess initiator status based on the tag.
				var initiator uint64
				if tag == Message {
					initiator = 1
				}
				go mp.sendMsg(ch<<3|Reset+initiator, nil, time.Time{})
				continue
			}

			msch.clLock.Lock()
			remoteClosed := msch.closedRemote
			msch.clLock.Unlock()
			if remoteClosed {
				log.Errorf("Received data from remote after stream was closed by them. (len = %d)", len(b))
				go mp.sendMsg(ch<<3|Reset+msch.initiator, nil, time.Time{})
				continue
			}

			recvTimeout.Reset(ReceiveTimeout)
			select {
			case msch.dataIn <- b:
			case <-msch.reset:
			case <-recvTimeout.C:
				log.Warningf("timed out receiving message into stream queue.")
				// Do not do this asynchronously. Otherwise, we
				// could drop a message, then receive a message,
				// then reset.
				msch.Reset()
				continue
			case <-mp.shutdown:
				return
			}
			if !recvTimeout.Stop() {
				<-recvTimeout.C
			}
		default:
			log.Debugf("message with unknown header on stream %s", ch)
			if ok {
				msch.Reset()
			}
		}
	}
}

func (mp *Multiplex) readNextHeader() (uint64, uint64, error) {
	h, err := binary.ReadUvarint(mp.buf)
	if err != nil {
		return 0, 0, err
	}

	// get channel ID
	ch := h >> 3

	rem := h & 7

	return ch, rem, nil
}

func (mp *Multiplex) readNext() ([]byte, error) {
	// get length
	l, err := binary.ReadUvarint(mp.buf)
	if err != nil {
		return nil, err
	}

	if l > uint64(MaxMessageSize) {
		return nil, fmt.Errorf("message size too large!")
	}

	if l == 0 {
		return nil, nil
	}

	buf := mpool.ByteSlicePool.Get(uint32(l)).([]byte)[:l]
	n, err := io.ReadFull(mp.buf, buf)
	if err != nil {
		return nil, err
	}

	return buf[:n], nil
}
