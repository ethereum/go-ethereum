// Copyright 2015 The go-ethereum Authors
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

// Contains the Whisper protocol Topic element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#topics.

package whisper5

import "github.com/ethereum/go-ethereum/crypto"

func NewTopic(data []byte) TopicType {
	prefix := [4]byte{}
	copy(prefix[:], crypto.Keccak256(data)[:4])
	return TopicType(prefix)
}

// NewTopicFromString creates a topic using the binary data contents of the specified string.
func NewTopicFromString(data string) TopicType {
	return NewTopic([]byte(data))
}

// String converts a topic byte array to a string representation.
func (self *TopicType) String() string {
	return string(self[:])
}
