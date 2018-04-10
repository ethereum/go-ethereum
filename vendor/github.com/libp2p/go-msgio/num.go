package msgio

import (
	"encoding/binary"
	"io"
)

// NBO is NetworkByteOrder
var NBO = binary.BigEndian

// WriteLen writes a length to the given writer.
func WriteLen(w io.Writer, l int) error {
	ul := uint32(l)
	return binary.Write(w, NBO, &ul)
}

// ReadLen reads a length from the given reader.
// if buf is non-nil, it reuses the buffer. Ex:
//    l, err := ReadLen(r, nil)
//    _, err := ReadLen(r, buf)
func ReadLen(r io.Reader, buf []byte) (int, error) {
	if len(buf) < 4 {
		buf = make([]byte, 4)
	}
	buf = buf[:4]

	if _, err := io.ReadFull(r, buf); err != nil {
		return 0, err
	}

	n := int(NBO.Uint32(buf))
	return n, nil
}
