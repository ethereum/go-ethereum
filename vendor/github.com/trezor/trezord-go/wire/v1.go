package wire

import (
	"encoding/binary"
	"errors"
	"io"

	"github.com/trezor/trezord-go/memorywriter"
)

const (
	repMarker = '?'
	repMagic  = '#'
	packetLen = 64
)

type Message struct {
	Kind uint16
	Data []byte

	Log *memorywriter.MemoryWriter
}

func (m *Message) WriteTo(w io.Writer) (int64, error) {
	m.Log.Log("start")

	var (
		rep  [packetLen]byte
		kind = m.Kind
		size = uint32(len(m.Data))
	)
	// pack header
	rep[0] = repMarker
	rep[1] = repMagic
	rep[2] = repMagic
	binary.BigEndian.PutUint16(rep[3:], kind)
	binary.BigEndian.PutUint32(rep[5:], size)

	m.Log.Log("actually writing")

	var (
		written = 0 // number of written bytes
		offset  = 9 // just after the header
	)
	for written < len(m.Data) {
		n := copy(rep[offset:], m.Data[written:])
		written += n
		offset += n
		if offset >= len(rep) {
			_, err := w.Write(rep[:])
			if err != nil {
				return int64(written), err
			}
			offset = 1 // just after the marker
		}
	}
	if offset != 1 {
		for offset < len(rep) {
			rep[offset] = 0x00
			offset++
		}
		_, err := w.Write(rep[:])
		if err != nil {
			return int64(written), err
		}
	}

	return int64(written), nil
}

var (
	ErrMalformedMessage = errors.New("malformed wire format")
)

func ReadFrom(r io.Reader, mw *memorywriter.MemoryWriter) (*Message, error) {
	mw.Log("start")
	var (
		rep  [packetLen]byte
		read = 0 // number of read bytes
	)
	n, err := r.Read(rep[:])
	if err != nil {
		return nil, err
	}

	// skip all the previous messages in the bus
	for rep[0] != repMarker || rep[1] != repMagic || rep[2] != repMagic {
		mw.Log("detected previous message, skipping")
		n, err = r.Read(rep[:])
		if err != nil {
			return nil, err
		}
	}
	read += n

	mw.Log("actual reading started")

	// parse header
	var (
		kind = binary.BigEndian.Uint16(rep[3:])
		size = binary.BigEndian.Uint32(rep[5:])
		data = make([]byte, 0, size)
	)
	data = append(data, rep[9:]...) // read data after header

	for uint32(len(data)) < size {
		n, err := r.Read(rep[:])
		if err != nil {
			return nil, err
		}
		if rep[0] != repMarker {
			return nil, ErrMalformedMessage
		}
		read += n
		data = append(data, rep[1:]...) // read data after marker
	}
	data = data[:size]

	mw.Log("actual reading finished")

	return &Message{
		Kind: kind,
		Data: data,

		Log: mw,
	}, nil
}
