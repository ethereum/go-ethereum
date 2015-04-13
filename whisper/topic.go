// Contains the Whisper protocol Topic element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#topics.

package whisper

import "github.com/ethereum/go-ethereum/crypto"

// Topic represents a cryptographically secure, probabilistic partial
// classifications of a message, determined as the first (left) 4 bytes of the
// SHA3 hash of some arbitrary data given by the original author of the message.
type Topic [4]byte

// NewTopic creates a topic from the 4 byte prefix of the SHA3 hash of the data.
func NewTopic(data []byte) Topic {
	prefix := [4]byte{}
	copy(prefix[:], crypto.Sha3(data)[:4])
	return Topic(prefix)
}

// String converts a topic byte array to a string representation.
func (self *Topic) String() string {
	return string(self[:])
}

// TopicSet represents a hash set to check if a topic exists or not.
type TopicSet map[string]struct{}

// NewTopicSet creates a topic hash set from a slice of topics.
func NewTopicSet(topics []Topic) TopicSet {
	set := make(map[string]struct{})
	for _, topic := range topics {
		set[topic.String()] = struct{}{}
	}
	return TopicSet(set)
}
