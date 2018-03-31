// Package multistream implements a simple stream router for the
// multistream-select protocoli. The protocol is defined at
// https://github.com/multiformats/multistream-select
package multistream

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"sync"
)

// ErrTooLarge is an error to signal that an incoming message was too large
var ErrTooLarge = errors.New("incoming message was too large")

// ProtocolID identifies the multistream protocol itself and makes sure
// the multistream muxers on both sides of a channel can work with each other.
const ProtocolID = "/multistream/1.0.0"

// HandlerFunc is a user-provided function used by the MultistreamMuxer to
// handle a protocol/stream.
type HandlerFunc func(protocol string, rwc io.ReadWriteCloser) error

// Handler is a wrapper to HandlerFunc which attaches a name (protocol) and a
// match function which can optionally be used to select a handler by other
// means than the name.
type Handler struct {
	MatchFunc func(string) bool
	Handle    HandlerFunc
	AddName   string
}

// MultistreamMuxer is a muxer for multistream. Depending on the stream
// protocol tag it will select the right handler and hand the stream off to it.
type MultistreamMuxer struct {
	handlerlock sync.Mutex
	handlers    []Handler
}

// NewMultistreamMuxer creates a muxer.
func NewMultistreamMuxer() *MultistreamMuxer {
	return new(MultistreamMuxer)
}

func writeUvarint(w io.Writer, i uint64) error {
	varintbuf := make([]byte, 16)
	n := binary.PutUvarint(varintbuf, i)
	_, err := w.Write(varintbuf[:n])
	if err != nil {
		return err
	}
	return nil
}

func delimWriteBuffered(w io.Writer, mes []byte) error {
	bw := bufio.NewWriter(w)
	err := delimWrite(bw, mes)
	if err != nil {
		return err
	}

	return bw.Flush()
}

func delimWrite(w io.Writer, mes []byte) error {
	err := writeUvarint(w, uint64(len(mes)+1))
	if err != nil {
		return err
	}

	_, err = w.Write(mes)
	if err != nil {
		return err
	}

	_, err = w.Write([]byte{'\n'})
	if err != nil {
		return err
	}
	return nil
}

// Ls is a Multistream muxer command which returns the list of handler names
// available on a muxer.
func Ls(rw io.ReadWriter) ([]string, error) {
	err := delimWriteBuffered(rw, []byte("ls"))
	if err != nil {
		return nil, err
	}

	n, err := binary.ReadUvarint(&byteReader{rw})
	if err != nil {
		return nil, err
	}

	var out []string
	for i := uint64(0); i < n; i++ {
		val, err := lpReadBuf(rw)
		if err != nil {
			return nil, err
		}
		out = append(out, string(val))
	}

	return out, nil
}

func fulltextMatch(s string) func(string) bool {
	return func(a string) bool {
		return a == s
	}
}

// AddHandler attaches a new protocol handler to the muxer.
func (msm *MultistreamMuxer) AddHandler(protocol string, handler HandlerFunc) {
	msm.AddHandlerWithFunc(protocol, fulltextMatch(protocol), handler)
}

// AddHandlerWithFunc attaches a new protocol handler to the muxer with a match.
// If the match function returns true for a given protocol tag, the protocol
// will be selected even if the handler name and protocol tags are different.
func (msm *MultistreamMuxer) AddHandlerWithFunc(protocol string, match func(string) bool, handler HandlerFunc) {
	msm.handlerlock.Lock()
	msm.removeHandler(protocol)
	msm.handlers = append(msm.handlers, Handler{
		MatchFunc: match,
		Handle:    handler,
		AddName:   protocol,
	})
	msm.handlerlock.Unlock()
}

// RemoveHandler removes the handler with the given name from the muxer.
func (msm *MultistreamMuxer) RemoveHandler(protocol string) {
	msm.handlerlock.Lock()
	defer msm.handlerlock.Unlock()

	msm.removeHandler(protocol)
}

func (msm *MultistreamMuxer) removeHandler(protocol string) {
	for i, h := range msm.handlers {
		if h.AddName == protocol {
			msm.handlers = append(msm.handlers[:i], msm.handlers[i+1:]...)
			return
		}
	}
}

// Protocols returns the list of handler-names added to this this muxer.
func (msm *MultistreamMuxer) Protocols() []string {
	var out []string
	msm.handlerlock.Lock()
	for _, h := range msm.handlers {
		out = append(out, h.AddName)
	}
	msm.handlerlock.Unlock()
	return out
}

// ErrIncorrectVersion is an error reported when the muxer protocol negotiation
// fails because of a ProtocolID mismatch.
var ErrIncorrectVersion = errors.New("client connected with incorrect version")

func (msm *MultistreamMuxer) findHandler(proto string) *Handler {
	msm.handlerlock.Lock()
	defer msm.handlerlock.Unlock()

	for _, h := range msm.handlers {
		if h.MatchFunc(proto) {
			return &h
		}
	}

	return nil
}

// NegotiateLazy performs protocol selection and returns
// a multistream, the protocol used, the handler and an error. It is lazy
// because the write-handshake is performed on a subroutine, allowing this
// to return before that handshake is completed.
func (msm *MultistreamMuxer) NegotiateLazy(rwc io.ReadWriteCloser) (Multistream, string, HandlerFunc, error) {
	pval := make(chan string, 1)
	writeErr := make(chan error, 1)
	defer close(pval)

	lzc := &lazyServerConn{
		con: rwc,
	}

	started := make(chan struct{})
	go lzc.waitForHandshake.Do(func() {
		close(started)

		defer close(writeErr)

		if err := delimWriteBuffered(rwc, []byte(ProtocolID)); err != nil {
			lzc.werr = err
			writeErr <- err
			return
		}

		for proto := range pval {
			if err := delimWriteBuffered(rwc, []byte(proto)); err != nil {
				lzc.werr = err
				writeErr <- err
				return
			}
		}
	})
	<-started

	line, err := ReadNextToken(rwc)
	if err != nil {
		return nil, "", nil, err
	}

	if line != ProtocolID {
		rwc.Close()
		return nil, "", nil, ErrIncorrectVersion
	}

loop:
	for {
		// Now read and respond to commands until they send a valid protocol id
		tok, err := ReadNextToken(rwc)
		if err != nil {
			rwc.Close()
			return nil, "", nil, err
		}

		switch tok {
		case "ls":
			select {
			case pval <- "ls":
			case err := <-writeErr:
				rwc.Close()
				return nil, "", nil, err
			}
		default:
			h := msm.findHandler(tok)
			if h == nil {
				select {
				case pval <- "na":
				case err := <-writeErr:
					rwc.Close()
					return nil, "", nil, err
				}
				continue loop
			}

			select {
			case pval <- tok:
			case <-writeErr:
				// explicitly ignore this error. It will be returned to any
				// writers and if we don't plan on writing anything, we still
				// want to complete the handshake
			}

			// hand off processing to the sub-protocol handler
			return lzc, tok, h.Handle, nil
		}
	}
}

// Negotiate performs protocol selection and returns the protocol name and
// the matching handler function for it (or an error).
func (msm *MultistreamMuxer) Negotiate(rwc io.ReadWriteCloser) (string, HandlerFunc, error) {
	// Send our protocol ID
	err := delimWriteBuffered(rwc, []byte(ProtocolID))
	if err != nil {
		return "", nil, err
	}

	line, err := ReadNextToken(rwc)
	if err != nil {
		return "", nil, err
	}

	if line != ProtocolID {
		rwc.Close()
		return "", nil, ErrIncorrectVersion
	}

loop:
	for {
		// Now read and respond to commands until they send a valid protocol id
		tok, err := ReadNextToken(rwc)
		if err != nil {
			return "", nil, err
		}

		switch tok {
		case "ls":
			err := msm.Ls(rwc)
			if err != nil {
				return "", nil, err
			}
		default:
			h := msm.findHandler(tok)
			if h == nil {
				err := delimWriteBuffered(rwc, []byte("na"))
				if err != nil {
					return "", nil, err
				}
				continue loop
			}

			err := delimWriteBuffered(rwc, []byte(tok))
			if err != nil {
				return "", nil, err
			}

			// hand off processing to the sub-protocol handler
			return tok, h.Handle, nil
		}
	}

}

// Ls implements the "ls" command which writes the list of
// supported protocols to the given Writer.
func (msm *MultistreamMuxer) Ls(w io.Writer) error {
	buf := new(bytes.Buffer)
	msm.handlerlock.Lock()
	err := writeUvarint(buf, uint64(len(msm.handlers)))
	if err != nil {
		return err
	}

	for _, h := range msm.handlers {
		err := delimWrite(buf, []byte(h.AddName))
		if err != nil {
			msm.handlerlock.Unlock()
			return err
		}
	}
	msm.handlerlock.Unlock()
	ll := make([]byte, 16)
	nw := binary.PutUvarint(ll, uint64(buf.Len()))

	r := io.MultiReader(bytes.NewReader(ll[:nw]), buf)

	_, err = io.Copy(w, r)
	return err
}

// Handle performs protocol negotiation on a ReadWriteCloser
// (i.e. a connection). It will find a matching handler for the
// incoming protocol and pass the ReadWriteCloser to it.
func (msm *MultistreamMuxer) Handle(rwc io.ReadWriteCloser) error {
	p, h, err := msm.Negotiate(rwc)
	if err != nil {
		return err
	}
	return h(p, rwc)
}

// ReadNextToken extracts a token from a ReadWriter. It is used during
// protocol negotiation and returns a string.
func ReadNextToken(rw io.ReadWriter) (string, error) {
	tok, err := ReadNextTokenBytes(rw)
	if err != nil {
		return "", err
	}

	return string(tok), nil
}

// ReadNextTokenBytes extracts a token from a ReadWriter. It is used
// during protocol negotiation and returns a byte slice.
func ReadNextTokenBytes(rw io.ReadWriter) ([]byte, error) {
	data, err := lpReadBuf(rw)
	switch err {
	case nil:
		return data, nil
	case ErrTooLarge:
		err := delimWriteBuffered(rw, []byte("messages over 64k are not allowed"))
		if err != nil {
			return nil, err
		}
		return nil, ErrTooLarge
	default:
		return nil, err
	}
}

func lpReadBuf(r io.Reader) ([]byte, error) {
	br, ok := r.(io.ByteReader)
	if !ok {
		br = &byteReader{r}
	}

	length, err := binary.ReadUvarint(br)
	if err != nil {
		return nil, err
	}

	if length > 64*1024 {
		return nil, ErrTooLarge
	}

	buf := make([]byte, length)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	if len(buf) == 0 || buf[length-1] != '\n' {
		return nil, errors.New("message did not have trailing newline")
	}

	// slice off the trailing newline
	buf = buf[:length-1]

	return buf, nil

}

// byteReader implements the ByteReader interface that ReadUVarint requires
type byteReader struct {
	io.Reader
}

func (br *byteReader) ReadByte() (byte, error) {
	var b [1]byte
	n, err := br.Read(b[:])
	if n == 1 {
		return b[0], nil
	}
	if err == nil {
		if n != 0 {
			panic("read more bytes than buffer size")
		}
		err = io.ErrNoProgress
	}
	return 0, err
}
