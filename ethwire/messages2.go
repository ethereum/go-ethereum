package ethwire

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/ethutil"
)

// The connection object allows you to set up a connection to the Ethereum network.
// The Connection object takes care of all encoding and sending objects properly over
// the network.
type Connection struct {
	conn            net.Conn
	nTimeout        time.Duration
	pendingMessages Messages
}

// Create a new connection to the Ethereum network
func New(conn net.Conn) *Connection {
	return &Connection{conn: conn, nTimeout: 500}
}

// Read, reads from the network. It will block until the next message is received.
func (self *Connection) Read() *Msg {
	if len(self.pendingMessages) == 0 {
		self.readMessages()
	}

	ret := self.pendingMessages[0]
	self.pendingMessages = self.pendingMessages[1:]

	return ret

}

// Write to the Ethereum network specifying the type of the message and
// the data. Data can be of type RlpEncodable or []interface{}. Returns
// nil or if something went wrong an error.
func (self *Connection) Write(typ MsgType, v ...interface{}) error {
	var pack []byte

	slice := [][]interface{}{[]interface{}{byte(typ)}}
	for _, value := range v {
		if encodable, ok := value.(ethutil.RlpEncodeDecode); ok {
			slice = append(slice, encodable.RlpValue())
		} else if raw, ok := value.([]interface{}); ok {
			slice = append(slice, raw)
		} else {
			panic(fmt.Sprintf("Unable to 'write' object of type %T", value))
		}
	}

	// Encode the type and the (RLP encoded) data for sending over the wire
	encoded := ethutil.NewValue(slice).Encode()
	payloadLength := ethutil.NumberToBytes(uint32(len(encoded)), 32)

	// Write magic token and payload length (first 8 bytes)
	pack = append(MagicToken, payloadLength...)
	pack = append(pack, encoded...)

	// Write to the connection
	_, err := self.conn.Write(pack)
	if err != nil {
		return err
	}

	return nil
}

func (self *Connection) readMessage(data []byte) (msg *Msg, remaining []byte, done bool, err error) {
	if len(data) == 0 {
		return nil, nil, true, nil
	}

	if len(data) <= 8 {
		return nil, remaining, false, errors.New("Invalid message")
	}

	// Check if the received 4 first bytes are the magic token
	if bytes.Compare(MagicToken, data[:4]) != 0 {
		return nil, nil, false, fmt.Errorf("MagicToken mismatch. Received %v", data[:4])
	}

	messageLength := ethutil.BytesToNumber(data[4:8])
	remaining = data[8+messageLength:]
	if int(messageLength) > len(data[8:]) {
		return nil, nil, false, fmt.Errorf("message length %d, expected %d", len(data[8:]), messageLength)
	}

	message := data[8 : 8+messageLength]
	decoder := ethutil.NewValueFromBytes(message)
	// Type of message
	t := decoder.Get(0).Uint()
	// Actual data
	d := decoder.SliceFrom(1)

	msg = &Msg{
		Type: MsgType(t),
		Data: d,
	}

	return
}

// The basic message reader waits for data on the given connection, decoding
// and doing a few sanity checks such as if there's a data type and
// unmarhals the given data
func (self *Connection) readMessages() (err error) {
	// The recovering function in case anything goes horribly wrong
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ethwire.ReadMessage error: %v", r)
		}
	}()

	// Buff for writing network message to
	//buff := make([]byte, 1440)
	var buff []byte
	var totalBytes int
	for {
		// Give buffering some time
		self.conn.SetReadDeadline(time.Now().Add(self.nTimeout * time.Millisecond))
		// Create a new temporarily buffer
		b := make([]byte, 1440)
		// Wait for a message from this peer
		n, _ := self.conn.Read(b)
		if err != nil && n == 0 {
			if err.Error() != "EOF" {
				fmt.Println("err now", err)
				return err
			} else {
				break
			}

			// Messages can't be empty
		} else if n == 0 {
			break
		}

		buff = append(buff, b[:n]...)
		totalBytes += n
	}

	// Reslice buffer
	buff = buff[:totalBytes]
	msg, remaining, done, err := self.readMessage(buff)
	for ; done != true; msg, remaining, done, err = self.readMessage(remaining) {
		//log.Println("rx", msg)

		if msg != nil {
			self.pendingMessages = append(self.pendingMessages, msg)
		}
	}

	return
}

func ReadMessage(data []byte) (msg *Msg, remaining []byte, done bool, err error) {
	if len(data) == 0 {
		return nil, nil, true, nil
	}

	if len(data) <= 8 {
		return nil, remaining, false, errors.New("Invalid message")
	}

	// Check if the received 4 first bytes are the magic token
	if bytes.Compare(MagicToken, data[:4]) != 0 {
		return nil, nil, false, fmt.Errorf("MagicToken mismatch. Received %v", data[:4])
	}

	messageLength := ethutil.BytesToNumber(data[4:8])
	remaining = data[8+messageLength:]
	if int(messageLength) > len(data[8:]) {
		return nil, nil, false, fmt.Errorf("message length %d, expected %d", len(data[8:]), messageLength)
	}

	message := data[8 : 8+messageLength]
	decoder := ethutil.NewValueFromBytes(message)
	// Type of message
	t := decoder.Get(0).Uint()
	// Actual data
	d := decoder.SliceFrom(1)

	msg = &Msg{
		Type: MsgType(t),
		Data: d,
	}

	return
}

func bufferedRead(conn net.Conn) ([]byte, error) {
	return nil, nil
}
