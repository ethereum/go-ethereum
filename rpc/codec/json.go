package codec

import (
	"encoding/json"
	"net"

	"github.com/ethereum/go-ethereum/rpc/shared"
)

const (
	MAX_RESPONSE_SIZE = 64 * 1024
)

// Json serialization support
type JsonCodec struct {
	c net.Conn
	d *json.Decoder
	e *json.Encoder
}

// Create new JSON coder instance
func NewJsonCoder(conn net.Conn) ApiCoder {
	return &JsonCodec{
		c: conn,
		d: json.NewDecoder(conn),
		e: json.NewEncoder(conn),
	}
}

// Serialize obj to JSON and write it to conn
func (self *JsonCodec) ReadRequest() (*shared.Request, error) {
	req := shared.Request{}
	err := self.d.Decode(&req)
	if err == nil {
		return &req, nil
	}
	return nil, err
}

func (self *JsonCodec) ReadResponse() (interface{}, error) {
	var err error
	buf := make([]byte, MAX_RESPONSE_SIZE)
	n, _ := self.c.Read(buf)

	var failure shared.ErrorResponse
	if err = json.Unmarshal(buf[:n], &failure); err == nil && failure.Error != nil {
		return failure, nil
	}

	var success shared.SuccessResponse
	if err = json.Unmarshal(buf[:n], &success); err == nil {
		return success, nil
	}

	return nil, err
}

// Encode response to encoded form in underlying stream
func (self *JsonCodec) Decode(data []byte, msg interface{}) error {
	return json.Unmarshal(data, msg)
}

func (self *JsonCodec) Encode(msg interface{}) ([]byte, error) {
	return json.Marshal(msg)
}

// Parse JSON data from conn to obj
func (self *JsonCodec) WriteResponse(res interface{}) error {
	return self.e.Encode(&res)
}

// Close decoder and encoder
func (self *JsonCodec) Close() {
	self.c.Close()
}
