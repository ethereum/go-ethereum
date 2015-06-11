package codec

import (
	"net"
	"strconv"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

type Codec int

// (de)serialization support for rpc interface
type ApiCoder interface {
	// Parse message to request from underlying stream
	ReadRequest() (*shared.Request, error)
	// Parse response message from underlying stream
	ReadResponse() (interface{}, error)
	// Encode response to encoded form in underlying stream
	WriteResponse(interface{}) error
	// Decode single message from data
	Decode([]byte, interface{}) error
	// Encode msg to encoded form
	Encode(msg interface{}) ([]byte, error)
	// close the underlying stream
	Close()
}

// supported codecs
const (
	JSON Codec = iota
	nCodecs
)

var (
	// collection with supported coders
	coders = make([]func(net.Conn) ApiCoder, nCodecs)
)

// create a new coder instance
func (c Codec) New(conn net.Conn) ApiCoder {
	switch c {
	case JSON:
		return NewJsonCoder(conn)
	}

	panic("codec: request for codec #" + strconv.Itoa(int(c)) + " is unavailable")
}
