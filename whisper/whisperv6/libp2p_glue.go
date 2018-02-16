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
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/p2p"
	inet "github.com/libp2p/go-libp2p-net"
)

type LibP2PStream struct {
	stream inet.Stream
}

const (
	codeLength        = 8
	payloadSizeLength = 4
)

// ReadMsg implements the MsgReadWriter interface to read messages
// from lilbp2p streams.
func (stream *LibP2PStream) ReadMsg() (p2p.Msg, error) {
	codeBytes := make([]byte, codeLength)
	nbytes, err := stream.stream.Read(codeBytes)
	if err != nil {
		return p2p.Msg{}, err
	} else if nbytes != len(codeBytes) {
		return p2p.Msg{}, fmt.Errorf("Invalid message header length: expected %d, got %d", codeLength, nbytes)
	}
	code := binary.LittleEndian.Uint64(codeBytes)

	sizeBytes := make([]byte, payloadSizeLength)
	nbytes, err = stream.stream.Read(sizeBytes)
	if err != nil {
		return p2p.Msg{}, err
	} else if nbytes != len(sizeBytes) {
		return p2p.Msg{}, fmt.Errorf("Invalid message size length: expected %d, got %d", len(sizeBytes), nbytes)
	}
	size := binary.LittleEndian.Uint32(sizeBytes)

	payload := make([]byte, size)
	nbytes, err = stream.stream.Read(payload)
	if err != nil {
		return p2p.Msg{}, err
	} else if nbytes != int(size) {
		return p2p.Msg{}, fmt.Errorf("Invalid message payload length: expected %d, got %d", size, nbytes)
	}

	return p2p.Msg{Code: code, Size: size, Payload: bytes.NewReader(payload)}, nil
}

// WriteMsg implements the MsgReadWriter interface to write messages
// to lilbp2p streams.
func (stream *LibP2PStream) WriteMsg(msg p2p.Msg) error {
	data := make([]byte, msg.Size+codeLength+payloadSizeLength)

	binary.LittleEndian.PutUint64(data[0:codeLength], msg.Code)
	binary.LittleEndian.PutUint32(data[codeLength:codeLength+payloadSizeLength], msg.Size)

	nbytes, err := msg.Payload.Read(data[codeLength+payloadSizeLength:])
	if (nbytes&0xFFFFFFFF) != nbytes || uint32(nbytes) != msg.Size {
		return fmt.Errorf("Invalid size read in libp2p stream: read %d bytes, was expecting %d bytes", nbytes, msg.Size)
	} else if err != nil {
		return err
	}

	nbytes, err = stream.stream.Write(data)

	if err != nil {
		return err
	}

	if nbytes != len(data) {
		return fmt.Errorf("Invalid size written in libp2p stream: wrote %d bytes, was expecting %d bytes", nbytes, msg.Size)
	}

	return nil
}
