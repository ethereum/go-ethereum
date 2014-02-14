package ethwire

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/ethereum/eth-go/ethutil"
	"net"
	"time"
)

// Message:
// [4 bytes token] RLP([TYPE, DATA])
// Refer to http://wiki.ethereum.org/index.php/Wire_Protocol

// The magic token which should be the first 4 bytes of every message.
var MagicToken = []byte{34, 64, 8, 145}

type MsgType byte

const (
	MsgHandshakeTy  = 0x00
	MsgDiscTy       = 0x01
	MsgPingTy       = 0x02
	MsgPongTy       = 0x03
	MsgGetPeersTy   = 0x10
	MsgPeersTy      = 0x11
	MsgTxTy         = 0x12
	MsgBlockTy      = 0x13
	MsgGetChainTy   = 0x14
	MsgNotInChainTy = 0x15

	MsgTalkTy = 0xff
)

var msgTypeToString = map[MsgType]string{
	MsgHandshakeTy:  "Handshake",
	MsgDiscTy:       "Disconnect",
	MsgPingTy:       "Ping",
	MsgPongTy:       "Pong",
	MsgGetPeersTy:   "Get peers",
	MsgPeersTy:      "Peers",
	MsgTxTy:         "Transactions",
	MsgBlockTy:      "Blocks",
	MsgGetChainTy:   "Get chain",
	MsgNotInChainTy: "Not in chain",
}

func (mt MsgType) String() string {
	return msgTypeToString[mt]
}

type Msg struct {
	Type MsgType // Specifies how the encoded data should be interpreted
	//Data []byte
	Data *ethutil.Value
}

func NewMessage(msgType MsgType, data interface{}) *Msg {
	return &Msg{
		Type: msgType,
		Data: ethutil.NewValue(data),
	}
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

// The basic message reader waits for data on the given connection, decoding
// and doing a few sanity checks such as if there's a data type and
// unmarhals the given data
func ReadMessages(conn net.Conn) (msgs []*Msg, err error) {
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
		conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
		// Create a new temporarily buffer
		b := make([]byte, 1440)
		// Wait for a message from this peer
		n, _ := conn.Read(b)
		if err != nil && n == 0 {
			if err.Error() != "EOF" {
				fmt.Println("err now", err)
				return nil, err
			} else {
				fmt.Println("IOF NOW")
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
	msg, remaining, done, err := ReadMessage(buff)
	for ; done != true; msg, remaining, done, err = ReadMessage(remaining) {
		//log.Println("rx", msg)

		if msg != nil {
			msgs = append(msgs, msg)
		}
	}

	return
}

// The basic message writer takes care of writing data over the given
// connection and does some basic error checking
func WriteMessage(conn net.Conn, msg *Msg) error {
	var pack []byte

	// Encode the type and the (RLP encoded) data for sending over the wire
	encoded := ethutil.NewValue(append([]interface{}{byte(msg.Type)}, msg.Data.Slice()...)).Encode()
	payloadLength := ethutil.NumberToBytes(uint32(len(encoded)), 32)

	// Write magic token and payload length (first 8 bytes)
	pack = append(MagicToken, payloadLength...)
	pack = append(pack, encoded...)
	//fmt.Printf("payload %v (%v) %q\n", msg.Type, conn.RemoteAddr(), encoded)

	// Write to the connection
	_, err := conn.Write(pack)
	if err != nil {
		return err
	}

	return nil
}
