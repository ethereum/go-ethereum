// Package ethwire provides low level access to the Ethereum network and allows
// you to broadcast data over the network.
package ethwire

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/ethereum/eth-go/ethutil"
)

// Connection interface describing the methods required to implement the wire protocol.
type Conn interface {
	Write(typ MsgType, v ...interface{}) error
	Read() *Msg
}

// The magic token which should be the first 4 bytes of every message and can be used as separator between messages.
var MagicToken = []byte{34, 64, 8, 145}

type MsgType byte

const (
	// Values are given explicitly instead of by iota because these values are
	// defined by the wire protocol spec; it is easier for humans to ensure
	// correctness when values are explicit.
	MsgHandshakeTy = 0x00
	MsgDiscTy      = 0x01
	MsgPingTy      = 0x02
	MsgPongTy      = 0x03
	MsgGetPeersTy  = 0x04
	MsgPeersTy     = 0x05

	MsgStatusTy         = 0x10
	MsgGetTxsTy         = 0x11
	MsgTxTy             = 0x12
	MsgGetBlockHashesTy = 0x13
	MsgBlockHashesTy    = 0x14
	MsgGetBlocksTy      = 0x15
	MsgBlockTy          = 0x16
)

var msgTypeToString = map[MsgType]string{
	MsgHandshakeTy:      "Handshake",
	MsgDiscTy:           "Disconnect",
	MsgPingTy:           "Ping",
	MsgPongTy:           "Pong",
	MsgGetPeersTy:       "Get peers",
	MsgStatusTy:         "Status",
	MsgPeersTy:          "Peers",
	MsgTxTy:             "Transactions",
	MsgBlockTy:          "Blocks",
	MsgGetTxsTy:         "Get Txs",
	MsgGetBlockHashesTy: "Get block hashes",
	MsgBlockHashesTy:    "Block hashes",
	MsgGetBlocksTy:      "Get blocks",
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

type Messages []*Msg

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

	var (
		buff      []byte
		messages  [][]byte
		msgLength int
	)

	for {
		// Give buffering some time
		conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
		// Create a new temporarily buffer
		b := make([]byte, 1440)
		n, _ := conn.Read(b)
		if err != nil && n == 0 {
			if err.Error() != "EOF" {
				fmt.Println("err now", err)
				return nil, err
			} else {
				break
			}
		}

		if n == 0 && len(buff) == 0 {
			continue
		}

		buff = append(buff, b[:n]...)
		if msgLength == 0 {
			// Check if the received 4 first bytes are the magic token
			if bytes.Compare(MagicToken, buff[:4]) != 0 {
				return nil, fmt.Errorf("MagicToken mismatch. Received %v", buff[:4])
			}

			// Read the length of the message
			msgLength = int(ethutil.BytesToNumber(buff[4:8]))

			// Remove the token and length
			buff = buff[8:]
		}

		if len(buff) >= msgLength {
			messages = append(messages, buff[:msgLength])
			buff = buff[msgLength:]
			msgLength = 0

			if len(buff) == 0 {
				break
			}
		}
	}

	for _, m := range messages {
		decoder := ethutil.NewValueFromBytes(m)
		// Type of message
		t := decoder.Get(0).Uint()
		// Actual data
		d := decoder.SliceFrom(1)

		msgs = append(msgs, &Msg{Type: MsgType(t), Data: d})
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
