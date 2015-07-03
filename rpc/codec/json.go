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

var (
	// No new requests in buffer
	EmptyRequestQueueError = fmt.Errorf("No incoming requests")
	// Next request in buffer isn't yet complete
	IncompleteRequestError = fmt.Errorf("Request incomplete")
)

// Json serialization support
type JsonCodec struct {
	c                net.Conn
	reqBuffer        []byte
	bytesInReqBuffer int
	reqLastPos       int
}

// Create new JSON coder instance
func NewJsonCoder(conn net.Conn) ApiCoder {
	return &JsonCodec{
		c:                conn,
		reqBuffer:        make([]byte, MAX_REQUEST_SIZE),
		bytesInReqBuffer: 0,
		reqLastPos:       0,
	}
}

// Indication if the next request in the buffer is a batch request
func (self *JsonCodec) isNextBatchReq() (bool, error) {
	for i := 0; i < self.bytesInReqBuffer; i++ {
		switch self.reqBuffer[i] {
		case 0x20, 0x09, 0x0a, 0x0d: // allow leading whitespace (JSON whitespace RFC4627)
			continue
		case 0x7b: // single req
			return false, nil
		case 0x5b: // batch req
			return true, nil
		default:
			return false, &json.InvalidUnmarshalError{}
		}
	}

	return false, EmptyRequestQueueError
}

// remove parsed request from buffer
func (self *JsonCodec) resetReqbuffer(pos int) {
	copy(self.reqBuffer, self.reqBuffer[pos:self.bytesInReqBuffer])
	self.reqLastPos = 0
	self.bytesInReqBuffer -= pos
}

// parse request in buffer
func (self *JsonCodec) nextRequest() (requests []*shared.Request, isBatch bool, err error) {
	if isBatch, err := self.isNextBatchReq(); err == nil {
		if isBatch {
			requests = make([]*shared.Request, 0)
			for ; self.reqLastPos <= self.bytesInReqBuffer; self.reqLastPos++ {
				if err = json.Unmarshal(self.reqBuffer[:self.reqLastPos], &requests); err == nil {
					self.resetReqbuffer(self.reqLastPos)
					return requests, true, nil
				}
			}
			return nil, true, IncompleteRequestError
		} else {
			request := shared.Request{}
			for ; self.reqLastPos <= self.bytesInReqBuffer; self.reqLastPos++ {
				if err = json.Unmarshal(self.reqBuffer[:self.reqLastPos], &request); err == nil {
					requests := make([]*shared.Request, 1)
					requests[0] = &request
					self.resetReqbuffer(self.reqLastPos)
					return requests, false, nil
				}
			}
			return nil, true, IncompleteRequestError
		}
	} else {
		return nil, false, err
	}
}

// Serialize obj to JSON and write it to conn
func (self *JsonCodec) ReadRequest() (requests []*shared.Request, isBatch bool, err error) {
	if self.bytesInReqBuffer != 0 {
		req, batch, err := self.nextRequest()
		if err == nil {
			return req, batch, err
		}

		if err != IncompleteRequestError {
			return nil, false, err
		}
	}

	// no/incomplete request in buffer -> read more data first
	deadline := time.Now().Add(READ_TIMEOUT * time.Second)
	if err := self.c.SetDeadline(deadline); err != nil {
		return nil, false, err
	}

	var retErr error
	for {
		n, err := self.c.Read(self.reqBuffer[self.bytesInReqBuffer:])
		if err != nil {
			retErr = err
			break
		}

		self.bytesInReqBuffer += n

		requests, isBatch, err := self.nextRequest()
		if err == nil {
			return requests, isBatch, nil
		}

		if err == IncompleteRequestError || err == EmptyRequestQueueError {
			continue // need more data
		}

		retErr = err
		break
	}

	self.c.Close()
	return nil, false, retErr
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
