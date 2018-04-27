package multihash

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
)

// Reader is an io.Reader wrapper that exposes a function
// to read a whole multihash, parse it, and return it.
type Reader interface {
	io.Reader

	ReadMultihash() (Multihash, error)
}

// Writer is an io.Writer wrapper that exposes a function
// to write a whole multihash.
type Writer interface {
	io.Writer

	WriteMultihash(Multihash) error
}

// NewReader wraps an io.Reader with a multihash.Reader
func NewReader(r io.Reader) Reader {
	return &mhReader{r}
}

// NewWriter wraps an io.Writer with a multihash.Writer
func NewWriter(w io.Writer) Writer {
	return &mhWriter{w}
}

type mhReader struct {
	r io.Reader
}

func (r *mhReader) Read(buf []byte) (n int, err error) {
	return r.r.Read(buf)
}

func (r *mhReader) ReadByte() (byte, error) {
	if br, ok := r.r.(io.ByteReader); ok {
		return br.ReadByte()
	}
	b := make([]byte, 1)
	_, err := r.r.Read(b)
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (r *mhReader) ReadMultihash() (Multihash, error) {
	code, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}

	length, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	if length > math.MaxInt32 {
		return nil, errors.New("digest too long, supporting only <= 2^31-1")
	}

	pre := make([]byte, 2*binary.MaxVarintLen64)
	spot := pre
	n := binary.PutUvarint(spot, code)
	spot = pre[n:]
	n += binary.PutUvarint(spot, length)

	buf := make([]byte, int(length)+n)
	copy(buf, pre[:n])

	if _, err := io.ReadFull(r.r, buf[n:]); err != nil {
		return nil, err
	}

	return Cast(buf)
}

type mhWriter struct {
	w io.Writer
}

func (w *mhWriter) Write(buf []byte) (n int, err error) {
	return w.w.Write(buf)
}

func (w *mhWriter) WriteMultihash(m Multihash) error {
	_, err := w.w.Write([]byte(m))
	return err
}
