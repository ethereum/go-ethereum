package msgio

import (
	"encoding/binary"
	"io"
	"sync"

	mpool "github.com/libp2p/go-msgio/mpool"
)

// varintWriter is the underlying type that implements the Writer interface.
type varintWriter struct {
	W io.Writer

	lbuf []byte      // for encoding varints
	lock sync.Locker // for threadsafe writes
}

// NewVarintWriter wraps an io.Writer with a varint msgio framed writer.
// The msgio.Writer will write the length prefix of every message written
// as a varint, using https://golang.org/pkg/encoding/binary/#PutUvarint
func NewVarintWriter(w io.Writer) WriteCloser {
	return &varintWriter{
		W:    w,
		lbuf: make([]byte, binary.MaxVarintLen64),
		lock: new(sync.Mutex),
	}
}

func (s *varintWriter) Write(msg []byte) (int, error) {
	err := s.WriteMsg(msg)
	if err != nil {
		return 0, err
	}
	return len(msg), nil
}

func (s *varintWriter) WriteMsg(msg []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	length := uint64(len(msg))
	n := binary.PutUvarint(s.lbuf, length)
	if _, err := s.W.Write(s.lbuf[:n]); err != nil {
		return err
	}
	_, err := s.W.Write(msg)
	return err
}

func (s *varintWriter) Close() error {
	if c, ok := s.W.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

// varintReader is the underlying type that implements the Reader interface.
type varintReader struct {
	R  io.Reader
	br io.ByteReader // for reading varints.

	lbuf []byte
	next int
	pool *mpool.Pool
	lock sync.Locker
	max  int // the maximal message size (in bytes) this reader handles
}

// NewVarintReader wraps an io.Reader with a varint msgio framed reader.
// The msgio.Reader will read whole messages at a time (using the length).
// Varints read according to https://golang.org/pkg/encoding/binary/#ReadUvarint
// Assumes an equivalent writer on the other side.
func NewVarintReader(r io.Reader) ReadCloser {
	return NewVarintReaderWithPool(r, mpool.ByteSlicePool)
}

// NewVarintReaderWithPool wraps an io.Reader with a varint msgio framed reader.
// The msgio.Reader will read whole messages at a time (using the length).
// Varints read according to https://golang.org/pkg/encoding/binary/#ReadUvarint
// Assumes an equivalent writer on the other side. It uses a given mpool.Pool
func NewVarintReaderWithPool(r io.Reader, p *mpool.Pool) ReadCloser {
	if p == nil {
		panic("nil pool")
	}
	return &varintReader{
		R:    r,
		br:   &simpleByteReader{R: r},
		lbuf: make([]byte, binary.MaxVarintLen64),
		next: -1,
		pool: p,
		lock: new(sync.Mutex),
		max:  defaultMaxSize,
	}
}

// NextMsgLen reads the length of the next msg into s.lbuf, and returns it.
// WARNING: like Read, NextMsgLen is destructive. It reads from the internal
// reader.
func (s *varintReader) NextMsgLen() (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.nextMsgLen()
}

func (s *varintReader) nextMsgLen() (int, error) {
	if s.next == -1 {
		length, err := binary.ReadUvarint(s.br)
		if err != nil {
			return 0, err
		}
		s.next = int(length)
	}
	return s.next, nil
}

func (s *varintReader) Read(msg []byte) (int, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	length, err := s.nextMsgLen()
	if err != nil {
		return 0, err
	}

	if length > len(msg) {
		return 0, io.ErrShortBuffer
	}
	_, err = io.ReadFull(s.R, msg[:length])
	s.next = -1 // signal we've consumed this msg
	return length, err
}

func (s *varintReader) ReadMsg() ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	length, err := s.nextMsgLen()
	if err != nil {
		return nil, err
	}

	if length > s.max {
		return nil, ErrMsgTooLarge
	}

	msgb := s.pool.Get(uint32(length))
	if msgb == nil {
		return nil, io.ErrShortBuffer
	}
	msg := msgb.([]byte)[:length]
	_, err = io.ReadFull(s.R, msg)
	s.next = -1 // signal we've consumed this msg
	return msg, err
}

func (s *varintReader) ReleaseMsg(msg []byte) {
	s.pool.Put(uint32(cap(msg)), msg)
}

func (s *varintReader) Close() error {
	if c, ok := s.R.(io.Closer); ok {
		return c.Close()
	}
	return nil
}

type simpleByteReader struct {
	R   io.Reader
	buf []byte
}

func (r *simpleByteReader) ReadByte() (c byte, err error) {
	if r.buf == nil {
		r.buf = make([]byte, 1)
	}

	if _, err := io.ReadFull(r.R, r.buf); err != nil {
		return 0, err
	}
	return r.buf[0], nil
}
