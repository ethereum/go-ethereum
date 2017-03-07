// Copyright 2014 The go-ethereum Authors
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

/*
Package whisper implements the Whisper PoC-1.

(https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec)

Whisper combines aspects of both DHTs and datagram messaging systems (e.g. UDP).
As such it may be likened and compared to both, not dissimilar to the
matter/energy duality (apologies to physicists for the blatant abuse of a
fundamental and beautiful natural principle).

Whisper is a pure identity-based messaging system. Whisper provides a low-level
(non-application-specific) but easily-accessible API without being based upon
or prejudiced by the low-level hardware attributes and characteristics,
particularly the notion of singular endpoints.
*/
package whisper5

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

const (
	statusCode   = 0x00
	messagesCode = 0x01

	protocolVersion    uint64 = 5
	protocolVersionStr        = "5.0"
	protocolName              = "shh"

	signatureFlag = byte(1 << 7)
	paddingFlag   = byte(1 << 6)

	signatureLength = 65
	maxPadLength    = 256 // must not exceed 256
	aesKeyLength    = 32
	saltLength      = 12
	kdfIterations   = 4096
	msgMaxLength    = 0xFFFF

	expirationCycle   = 800 * time.Millisecond
	transmissionCycle = 300 * time.Millisecond

	DefaultTTL = 50 * time.Second
	DefaultPoW = 50 * time.Millisecond
)

// Topic represents a cryptographically secure, probabilistic partial
// classifications of a message, determined as the first (left) 4 bytes of the
// SHA3 hash of some arbitrary data given by the original author of the message.
type TopicType [4]byte

func BytesToTopic(b []byte) (t TopicType) {
	sz := 4
	if x := len(b); x < 4 {
		sz = x
	}
	for i := 0; i < sz; i++ {
		t[i] = b[i]
	}
	return t
}

func HashToTopic(h common.Hash) (t TopicType) {
	for i := 0; i < 4; i++ {
		t[i] = h[i]
	}
	return t
}
