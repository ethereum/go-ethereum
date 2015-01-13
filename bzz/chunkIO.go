package bzz

import (
	"bytes"
	"errors"
	"io"
)

type Bounded interface {
	Size() int64
}

type Resizeable interface {
	Bounded
	Resize(int64) error
}

type Sliced interface {
	Slice(int64, int64) []byte
}

// Size, Seek, Read, ReadAt and WriteTo
type SectionReader interface {
	Bounded
	Sliced
	io.Seeker
	io.Reader
	io.ReaderAt
	io.WriterTo
}

// Size, Seek, Write, WriteAt and ReaderFrom
type SectionWriter interface {
	Bounded
	Sliced
	io.Seeker
	io.Writer
	io.WriterAt
	io.ReaderFrom
}

// ChunkReader implements SectionReader on a section
// of an underlying ReaderAt.
type ChunkReader struct {
	r     io.ReaderAt
	base  int64
	off   int64
	limit int64
}

// ChunkWriter implements SectionWriter on a section
// of an underlying WriterAt.
type ChunkWriter struct {
	w     io.WriterAt
	base  int64
	off   int64
	limit int64
}

type SectionReadWriter struct {
	Bounded
	Sliced
	io.Seeker
	io.Reader
	io.ReaderAt
	io.WriterTo
	io.Writer
	io.WriterAt
	io.ReaderFrom
}

// NewChunkReader returns a ChunkReader that reads from r
// starting at offset off and stops with EOF after n bytes.
func NewChunkReader(r io.ReaderAt, off int64, n int64) *ChunkReader {
	return &ChunkReader{r: r, base: off, off: off, limit: off + n}
}

func NewChunkReaderFromBytes(b []byte) *ChunkReader {
	return NewChunkReader(bytes.NewReader(b), 0, int64(len(b)))
}

// NewChunkWriter returns a ChunkWriter that writes to w
// starting at offset off and stops with EOF if write would go past off+n
func NewChunkWriter(w io.WriterAt, off int64, n int64) *ChunkWriter {
	return &ChunkWriter{w: w, base: off, off: off, limit: off + n}
}

func NewChunkWriterFromBytes(b []byte) *ChunkWriter {
	return NewChunkWriter(NewByteSliceWriter(b), 0, int64(len(b)))
}

// the write equivalent of bytes.NewReader(b)
func NewByteSliceWriter(b []byte) *ByteSliceWriter {
	return &ByteSliceWriter{b: b, off: 0, limit: int64(len(b))}
}

type ByteSliceWriter struct {
	b     []byte
	off   int64
	limit int64
}

func (self *ByteSliceWriter) Slice(from, to int64) (slice []byte) {
	dpaLogger.DebugDetailf("bottom line %v:%v  (%v-%v)", from, to, self.off, self.limit)
	if from >= 0 && to <= self.limit {
		slice = self.b[from:to]
	}
	return
}

func (self *ByteSliceWriter) WriteAt(b []byte, off int64) (n int, err error) {
	if off < 0 || off >= self.limit {
		return 0, io.ErrShortWrite
	}
	if n = int(self.limit - off); len(b) > n {
		err = io.ErrShortWrite
	} else {
		n = len(b)
	}
	copy(self.b[off:], b)
	return
}

func (self *ByteSliceWriter) Size() (size int64) {
	return self.limit
}

var errUnableToResize = errors.New("unable to resize")

func (self *ByteSliceWriter) Resize(size int64) (err error) {
	if self.Size() != 0 {
		err = errUnableToResize
	} else {
		self.b = make([]byte, size)
		self.limit = size
	}
	return
}

// Size returns the size of the section in bytes.
func (s *ChunkReader) Size() int64 { return s.limit - s.base }
func (s *ChunkWriter) Size() int64 { return s.limit - s.base }

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

func (s *ChunkWriter) Seek(offset int64, whence int) (int64, error) {
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

func (self *ChunkWriter) Resize(size int64) (err error) {
	err = errUnableToResize
	if self.Size() >= size {
		err = nil
	} else {
		if self.Size() == 0 {
			if ws, ok := self.w.(Resizeable); ok {
				err = ws.Resize(size)
				if err == nil {
					self.limit = size
				}
			}
		}
	}
	return
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

func (s *ChunkWriter) Write(p []byte) (n int, err error) {
	if s.off >= s.limit {
		return 0, io.EOF
	}
	if max := s.limit - s.off; int64(len(p)) > max {
		p = p[0:max]
	}
	n, err = s.w.WriteAt(p, s.off)
	s.off += int64(n)
	return
}

func (s *ChunkReader) Slice(from, to int64) []byte {
	dpaLogger.DebugDetailf("%v %v %v", s.base, s.off, s.limit)
	if sl, ok := s.r.(Sliced); ok {
		dpaLogger.DebugDetailf("%v-%v", s.base+from, s.base+to)
		return sl.Slice(s.base+from, s.base+to)
	}
	return nil
}

func (s *ChunkWriter) Slice(from, to int64) (b []byte) {
	dpaLogger.DebugDetailf("base: %v %v %v", s.base, s.off, s.limit)
	if sl, ok := s.w.(Sliced); ok {
		dpaLogger.DebugDetailf("%v-%v", s.base+from, s.base+to)
		b = sl.Slice(s.base+from, s.base+to)
	}
	return
}

//
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
	return s.r.ReadAt(p, off)
}

//
func (s *ChunkWriter) WriteAt(p []byte, off int64) (n int, err error) {
	if off < 0 || off >= s.limit-s.base {
		return 0, io.EOF
	}
	off += s.base
	if max := s.limit - off; int64(len(p)) > max {
		p = p[0:max]
		n, err = s.w.WriteAt(p, off)
		return n, err
	}
	return s.w.WriteAt(p, off)
}

func (s *ChunkWriter) ReadFrom(r io.Reader) (n int64, err error) {
	var m int
	// if byte slice is available
	if slice := s.Slice(s.off-s.base, s.limit-s.base); slice != nil {
		dpaLogger.DebugDetailf("readfrom %v + %v-%v", s.base, s.off-s.base, s.limit-s.base)
		m, err = r.Read(slice)
		if err != nil {
			dpaLogger.Debugf("%v (m%v)", err, m)
		}
		dpaLogger.DebugDetailf("read slice %x", slice)
	} else {
		b := make([]byte, s.limit-s.off)
		_, err = r.Read(b)
		m, err = s.Write(b)
	}
	n = int64(m)
	return
}

func (r *ChunkReader) WriteTo(w io.Writer) (n int64, err error) {
	var b []byte
	var m int
	if b := r.Slice(r.off, r.limit); b == nil {
		// if slices not available we do it with extra allocation
		b = make([]byte, r.limit-r.off)
		m, err = r.Read(b)
		if err != nil {
			return
		}
	}
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
