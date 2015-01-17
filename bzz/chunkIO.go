//
package bzz

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"time"
)

type Bounded interface {
	Size() int64
}

type Sliced interface {
	Slice(int64, int64) (b []byte, err error)
}

// Size, Seek, Read, ReadAt
type SectionReader interface {
	Bounded
	io.Seeker
	io.Reader
	io.ReaderAt
}

type LazySectionReader interface {
	SectionReader
	GetTimeout() time.Time
	SetTimeout(time.Time) error
}

// ChunkReader implements SectionReader on a section
// of an underlying ReaderAt.
type ChunkReader struct {
	r     io.ReaderAt
	base  int64
	off   int64
	limit int64
}

// NewChunkReader returns a ChunkReader that reads from r
// starting at offset off and stops with EOF after n bytes.
func NewChunkReader(r io.ReaderAt, off int64, n int64) *ChunkReader {
	return &ChunkReader{r: r, base: off, off: off, limit: off + n}
}

// ByteSliceReader just extends byte.Reader to make base slice accessible
type ByteSliceReader struct {
	*bytes.Reader
	base []byte
}

func NewByteSliceReader(b []byte) *ByteSliceReader {
	return &ByteSliceReader{
		base:   b,
		Reader: bytes.NewReader(b),
	}
}

// ByteSliceReader implements the Sliced interface
func (self *ByteSliceReader) Slice(from, to int64) (b []byte, err error) {
	if from < 0 || to >= int64(self.Len()) {
		err = io.EOF
	} else {
		b = self.base[from:to]
	}
	return
}

// NewChunkReaderFromBytes is a convenience shortcut to get a SectionReader over a byte slice
func NewChunkReaderFromBytes(b []byte) *ChunkReader {
	return NewChunkReader(NewByteSliceReader(b), 0, int64(len(b)))
}

/*
The following is adapted from io.SectionReader
*/

func (s *ChunkReader) Size() int64 { return s.limit - s.base }

var errWhence = errors.New("Seek: invalid whence")
var errOffset = errors.New("Seek: invalid offset")

func (s *ChunkReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errWhence
	case 0:
		offset += s.base
	case 1:
		offset += s.off
	case 2:
		offset += s.limit
	}
	if offset < s.base {
		return 0, errOffset
	}
	s.off = offset
	return offset - s.base, nil
}

func (s *ChunkReader) Read(p []byte) (n int, err error) {
	if s.off >= s.limit {
		return 0, io.EOF
	}
	if max := s.limit - s.off; int64(len(p)) > max {
		p = p[0:max]
	}
	n, err = s.r.ReadAt(p, s.off)
	s.off += int64(n)
	return
}

func (s *ChunkReader) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= s.limit-s.base {
		return 0, io.EOF
	}
	off += s.base
	if max := s.limit - off; int64(len(p)) > max {
		p = p[0:max]
		n, err = s.r.ReadAt(p, off)
		if err == nil {
			err = io.EOF
		}
		return n, err
	}
	n, err = s.r.ReadAt(p, off)
	return
}

// added methods to that ChunkReader implements the Sliced interface
func (s *ChunkReader) Slice(from, to int64) (b []byte, err error) {
	if from < 0 || to >= s.Size() {
		err = io.EOF
	} else {
		if sl, ok := s.r.(Sliced); ok {
			b, err = sl.Slice(s.base+from, s.base+to)
		} else {
			err = errors.New("not sliceable base")
		}
	}
	return
}

// added method so that ChunkReader implements the io.WriterTo interface
// WriteTo method is used by io.Copy
// This is so that we avoid one extra step of allocation (if the underlying initial Reader implements Sliced
func (r *ChunkReader) WriteTo(w io.Writer) (n int64, err error) {
	var b []byte
	var m int
	// if b, _ := r.Slice(r.off-r.base, r.limit-r.base); b == nil {
	// if slices not available we do it with extra allocation
	b = make([]byte, r.limit-r.off)
	m, err = r.Read(b)
	if err != nil {
		return
	}
	// }
	m, err = w.Write(b)
	if m > len(b) {
		panic("bytes.Reader.WriteTo: invalid Write count")
	}
	r.off = r.base + int64(m)
	n = int64(m)
	if m != len(b) && err == nil {
		err = io.ErrShortWrite
	}
	// w
	return
}

// LazyChunkReader implements LazySectionReader
type LazyChunkReader struct {
	sections []LazySectionReader
	readerFs [](func() LazySectionReader)
	size     int64
	treeSize int64
	off      int64
}

func (self *LazyChunkReader) GetTimeout() (t time.Time) {
	return
}

func (self *LazyChunkReader) SetTimeout(t time.Time) error {
	return nil
}

func (self *LazyChunkReader) Size() (n int64) {
	return self.size
}

func (self *LazyChunkReader) Read(b []byte) (read int, err error) {
	read, err = self.ReadAt(b, self.off)
	self.off += int64(read)
	return
}

func (s *LazyChunkReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	default:
		return 0, errWhence
	case 0:
		offset += 0
	case 1:
		offset += s.off
	case 2:
		offset += s.size
	}
	if offset < 0 {
		return 0, errOffset
	}
	s.off = offset
	return offset, nil
}

func (self *LazyChunkReader) ReadAt(b []byte, off int64) (read int, err error) {
	if off < 0 || off >= self.size {
		err = io.EOF
		return
	}
	want := len(b)
	got := int(self.size - off)
	if want > got {
		err = io.EOF
	}
	var index int
	index = int(off / self.treeSize)
	off -= int64(index) * self.treeSize
	var limit int
	var reader LazySectionReader

	for i := index; i < len(self.sections) && want > 0; i++ {
		if reader = self.sections[i]; reader == nil {
			reader = self.readerFs[i]() // this hooks into the go routines and does a wg.Done once the needed chunks are fetched and processed
			self.sections[i] = reader   // memoize this reader
		}
		if want > int(reader.Size()-off) {
			limit = int(reader.Size() - off)
		} else {
			limit = want
		}
		if got, err = reader.ReadAt(b[read:read+limit], off); err != nil {
			return
		}
		read += got
		want -= got
		off = 0
	}
	return
}

type EmbeddedReader struct {
	LazySectionReader
	lock sync.Mutex
}

type NotReadyReader struct {
	initF func()
	r     LazySectionReader
}

func (self *NotReadyReader) Size() (n int64) {
	self.initF()
	return self.r.Size()
}

func (self *NotReadyReader) Read(b []byte) (read int, err error) {
	self.initF()
	return self.r.Read(b)
}

func (self *NotReadyReader) ReadAt(b []byte, off int64) (read int, err error) {
	self.initF()
	return self.r.ReadAt(b, off)
}

func (self *NotReadyReader) Seek(offset int64, whence int) (int64, error) {
	self.initF()
	return self.r.Seek(offset, whence)
}

// just a wrapper to make a SectionReader conform to LazySectionReader
// with dummy implementation
type lazyReader struct {
	SectionReader
}

func LazyReader(r SectionReader) (l LazySectionReader) {
	return &lazyReader{
		SectionReader: r,
	}
}

func (self *lazyReader) GetTimeout() (t time.Time) {
	return
}

func (self *lazyReader) SetTimeout(t time.Time) error {
	return nil
}
