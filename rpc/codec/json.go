package codec

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	READ_TIMEOUT      = 15 // read timeout in seconds
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

// Serialize obj to JSON and write it to conn
func (self *JsonCodec) ReadRequest() (requests []*shared.Request, isBatch bool, err error) {

	deadline := time.Now().Add(READ_TIMEOUT * time.Second)
	if err := self.c.SetDeadline(deadline); err != nil {
		return nil, false, err
	}

	for {
		var err error
		singleRequest := shared.Request{}
		if err = self.d.Decode(&singleRequest); err == nil {
			requests := make([]*shared.Request, 1)
			requests[0] = &singleRequest
			return requests, false, nil
		}

		fmt.Printf("err %T %v\n", err)

		if opErr, ok := err.(*net.OpError); ok {
			if opErr.Timeout() {
				break
			}
		}

		requests = make([]*shared.Request, 0)
		if err = self.d.Decode(&requests); err == nil {
			return requests, true, nil
		}

		if opErr, ok := err.(*net.OpError); ok {
			if opErr.Timeout() {
				break
			}
		}
	}

	self.c.Close() // timeout
	return nil, false, fmt.Errorf("Timeout reading request")
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
