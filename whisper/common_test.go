// Contains some common utility functions for testing.

package whisper

import (
	"bytes"
	"io/ioutil"

	"github.com/ethereum/go-ethereum/p2p"
)

// bufMsgPipe creates a buffered message pipe between two endpoints.
func bufMsgPipe() (*p2p.MsgPipeRW, *p2p.MsgPipeRW) {
	A, midA := p2p.MsgPipe()
	midB, B := p2p.MsgPipe()

	go copyMsgPipe(midA, midB)
	go copyMsgPipe(midB, midA)

	return A, B
}

// copyMsgPipe copies messages from the src pipe to the dest.
func copyMsgPipe(dst, src *p2p.MsgPipeRW) {
	defer dst.Close()
	for {
		msg, err := src.ReadMsg()
		if err != nil {
			return
		}
		data, err := ioutil.ReadAll(msg.Payload)
		if err != nil {
			return
		}
		msg.Payload = bytes.NewReader(data)
		if err := dst.WriteMsg(msg); err != nil {
			return
		}
	}
}
