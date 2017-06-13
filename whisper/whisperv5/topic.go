// Copyright 2016 The go-ethereum Authors
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

// Contains the Whisper protocol Topic element.

package whisperv5

import (
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// Topic represents a cryptographically secure, probabilistic partial
// classifications of a message, determined as the first (left) 4 bytes of the
// SHA3 hash of some arbitrary data given by the original author of the message.
type TopicType [TopicLength]byte

func BytesToTopic(b []byte) (t TopicType) {
	sz := TopicLength
	if x := len(b); x < TopicLength {
		sz = x
	}
	for i := 0; i < sz; i++ {
		t[i] = b[i]
	}
	return t
}

// String converts a topic byte array to a string representation.
func (topic *TopicType) String() string {
	return string(common.ToHex(topic[:]))
}

func (t *TopicType) MarshalJSON() ([]byte, error) {
	return json.Marshal(hexutil.Bytes(t[:]))
}

// UnmarshalJSON parses a hex representation to a topic.
func (t *TopicType) UnmarshalJSON(input []byte) error {
	var data hexutil.Bytes
	if err := json.Unmarshal(input, &data); err != nil {
		return err
	}
	if len(data) != TopicLength {
		return fmt.Errorf("unmarshalJSON failed: topic must be exactly %d bytes(%d)", TopicLength, len(input))
	}
	*t = BytesToTopic(data)
	return nil
}
