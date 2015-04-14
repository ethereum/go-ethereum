// Contains some common utility functions for testing.

package whisper

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"

	"github.com/ethereum/go-ethereum/p2p"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

// randomNodeID generates and returns a random P2P discovery node id for the
// whisper tests.
func randomNodeID() (id discover.NodeID) {
	for i := range id {
		id[i] = byte(rand.Intn(255))
	}
	return id
}

// randomNodeName generates and returns a random P2P node name for the whisper
// tests.
func randomNodeName() string {
	return common.MakeName(fmt.Sprintf("whisper-go-test-%3d", rand.Intn(999)), "1.0")
}

// whisperCaps returns the node capabilities for running the whisper sub-protocol.
func whisperCaps() []p2p.Cap {
	return []p2p.Cap{
		p2p.Cap{
			Name:    protocolName,
			Version: uint(protocolVersion),
		},
	}
}

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
