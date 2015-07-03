package codec

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	READ_TIMEOUT      = 60 // in seconds
	MAX_REQUEST_SIZE  = 1024 * 1024
	MAX_RESPONSE_SIZE = 1024 * 1024
)

// Json serialization support
type JsonCodec struct {
	c net.Conn
	d *json.Decoder
}

// Create new JSON coder instance
func NewJsonCoder(conn net.Conn) ApiCoder {
	return &JsonCodec{
		c: conn,
		d: json.NewDecoder(conn),
	}
}

// Read incoming request and parse it to RPC request
func (self *JsonCodec) ReadRequest() (requests []*shared.Request, isBatch bool, err error) {
	deadline := time.Now().Add(READ_TIMEOUT * time.Second)
	if err := self.c.SetDeadline(deadline); err != nil {
		return nil, false, err
	}

	var incoming json.RawMessage
	err = self.d.Decode(&incoming)
	if err == nil {
		isBatch = incoming[0] == '['
		if isBatch {
			requests = make([]*shared.Request, 0)
			err = json.Unmarshal(incoming, &requests)
		} else {
			requests = make([]*shared.Request, 1)
			var singleRequest shared.Request
			if err = json.Unmarshal(incoming, &singleRequest); err == nil {
				requests[0] = &singleRequest
			}
		}
		return
	}

	self.c.Close()
	return nil, false, err
}

func (self *JsonCodec) ReadResponse() (interface{}, error) {
	bytesInBuffer := 0
	buf := make([]byte, MAX_RESPONSE_SIZE)

	deadline := time.Now().Add(READ_TIMEOUT * time.Second)
	if err := self.c.SetDeadline(deadline); err != nil {
		return nil, err
	}

	for {
		n, err := self.c.Read(buf[bytesInBuffer:])
		if err != nil {
			return nil, err
		}
		bytesInBuffer += n

		var failure shared.ErrorResponse
		if err = json.Unmarshal(buf[:bytesInBuffer], &failure); err == nil && failure.Error != nil {
			return failure, fmt.Errorf(failure.Error.Message)
		}

		var success shared.SuccessResponse
		if err = json.Unmarshal(buf[:bytesInBuffer], &success); err == nil {
			return success, nil
		}
	}

	self.c.Close()
	return nil, fmt.Errorf("Unable to read response")
}

// Decode data
func (self *JsonCodec) Decode(data []byte, msg interface{}) error {
	return json.Unmarshal(data, msg)
}

// Encode message
func (self *JsonCodec) Encode(msg interface{}) ([]byte, error) {
	return json.Marshal(msg)
}

// Parse JSON data from conn to obj
func (self *JsonCodec) WriteResponse(res interface{}) error {
	data, err := json.Marshal(res)
	if err != nil {
		self.c.Close()
		return err
	}

	bytesWritten := 0

	for bytesWritten < len(data) {
		n, err := self.c.Write(data[bytesWritten:])
		if err != nil {
			self.c.Close()
			return err
		}
		bytesWritten += n
	}

	return nil
}

// Close decoder and encoder
func (self *JsonCodec) Close() {
	self.c.Close()
}
