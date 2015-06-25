package codec

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	MAX_REQUEST_SIZE  = 1024 * 1024
	MAX_RESPONSE_SIZE = 1024 * 1024
)

// Json serialization support
type JsonCodec struct {
	c             net.Conn
	buffer        []byte
	bytesInBuffer int
}

// Create new JSON coder instance
func NewJsonCoder(conn net.Conn) ApiCoder {
	return &JsonCodec{
		c:             conn,
		buffer:        make([]byte, MAX_REQUEST_SIZE),
		bytesInBuffer: 0,
	}
}

// Serialize obj to JSON and write it to conn
func (self *JsonCodec) ReadRequest() (requests []*shared.Request, isBatch bool, err error) {
	n, err := self.c.Read(self.buffer[self.bytesInBuffer:])
	if err != nil {
		self.bytesInBuffer = 0
		return nil, false, err
	}

	self.bytesInBuffer += n

	singleRequest := shared.Request{}
	err = json.Unmarshal(self.buffer[:self.bytesInBuffer], &singleRequest)
	if err == nil {
		self.bytesInBuffer = 0
		requests := make([]*shared.Request, 1)
		requests[0] = &singleRequest
		return requests, false, nil
	}

	requests = make([]*shared.Request, 0)
	err = json.Unmarshal(self.buffer[:self.bytesInBuffer], &requests)
	if err == nil {
		self.bytesInBuffer = 0
		return requests, true, nil
	}

	return nil, false, err
}

func (self *JsonCodec) ReadResponse() (interface{}, error) {
	bytesInBuffer := 0
	buf := make([]byte, MAX_RESPONSE_SIZE)

	deadline := time.Now().Add(15 * time.Second)
	self.c.SetDeadline(deadline)

	for {
		n, err := self.c.Read(buf[bytesInBuffer:])
		if err != nil {
			return nil, err
		}
		bytesInBuffer += n

		var success shared.SuccessResponse
		if err = json.Unmarshal(buf[:bytesInBuffer], &success); err == nil {
			return success, nil
		}

		var failure shared.ErrorResponse
		if err = json.Unmarshal(buf[:bytesInBuffer], &failure); err == nil && failure.Error != nil {
			return failure, nil
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
