// Copyright 2018 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package whisperv6

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/rlp"
	"io/ioutil"
	"time"

	"github.com/ethereum/go-ethereum/p2p"
	inet "github.com/libp2p/go-libp2p-net"
)

// LibP2PStream is a wrapper used to implement the MsgReadWriter
// interface for libp2p's streams.
type LibP2PStream struct {
	stream inet.Stream
}

// serializableP2PMsg is the serializable version of p2p.Msg
type serializableP2PMsg struct {
	Code       uint64
	Size       uint32
	Payload    []byte
	ReceivedAt time.Time
}

// ReadMsg implements the MsgReadWriter interface to read messages
// from lilbp2p streams.
func (stream *LibP2PStream) ReadMsg() (p2p.Msg, error) {
	var m serializableP2PMsg
	raw, err := rlp.NewStream(bufio.NewReader(stream.stream), 0).Raw()
	if err != nil {
		return p2p.Msg{}, fmt.Errorf("Error during RLP decoding of message: %v", err)
	}
	err = rlp.DecodeBytes(raw, &m)
	msg := p2p.Msg{
		Code:       m.Code,
		Size:       m.Size,
		Payload:    bytes.NewReader(m.Payload),
		ReceivedAt: m.ReceivedAt,
	}
	return msg, err
}

// WriteMsg implements the MsgReadWriter interface to write messages
// to lilbp2p streams.
func (stream *LibP2PStream) WriteMsg(msg p2p.Msg) error {
	payload, err := ioutil.ReadAll(msg.Payload)
	if err != nil {
		return fmt.Errorf("Error reading payload: %v", err)
	}
	m := serializableP2PMsg{msg.Code, msg.Size, payload, msg.ReceivedAt}

	data, err := rlp.EncodeToBytes(m)
	if err != nil {
		return fmt.Errorf("Error encoding to RLP: %v", err)
	}

	nbytes, err := stream.stream.Write(data)
	if err != nil {
		return err
	}

	if nbytes != len(data) {
		return fmt.Errorf("Invalid size written in libp2p stream: wrote %d bytes, was expecting %d bytes", nbytes, msg.Size)
	}

	return nil
}
