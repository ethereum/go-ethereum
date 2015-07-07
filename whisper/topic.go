// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

// Contains the Whisper protocol Topic element. For formal details please see
// the specs at https://github.com/ethereum/wiki/wiki/Whisper-PoC-1-Protocol-Spec#topics.

package whisper

import "github.com/ethereum/go-ethereum/crypto"

// Topic represents a cryptographically secure, probabilistic partial
// classifications of a message, determined as the first (left) 4 bytes of the
// SHA3 hash of some arbitrary data given by the original author of the message.
type Topic [4]byte

// NewTopic creates a topic from the 4 byte prefix of the SHA3 hash of the data.
//
// Note, empty topics are considered the wildcard, and cannot be used in messages.
func NewTopic(data []byte) Topic {
	prefix := [4]byte{}
	copy(prefix[:], crypto.Sha3(data)[:4])
	return Topic(prefix)
}

// NewTopics creates a list of topics from a list of binary data elements, by
// iteratively calling NewTopic on each of them.
func NewTopics(data ...[]byte) []Topic {
	topics := make([]Topic, len(data))
	for i, element := range data {
		topics[i] = NewTopic(element)
	}
	return topics
}

// NewTopicFromString creates a topic using the binary data contents of the
// specified string.
func NewTopicFromString(data string) Topic {
	return NewTopic([]byte(data))
}

// NewTopicsFromStrings creates a list of topics from a list of textual data
// elements, by iteratively calling NewTopicFromString on each of them.
func NewTopicsFromStrings(data ...string) []Topic {
	topics := make([]Topic, len(data))
	for i, element := range data {
		topics[i] = NewTopicFromString(element)
	}
	return topics
}

// String converts a topic byte array to a string representation.
func (self *Topic) String() string {
	return string(self[:])
}

// topicMatcher is a filter expression to verify if a list of topics contained
// in an arriving message matches some topic conditions. The topic matcher is
// built up of a list of conditions, each of which must be satisfied by the
// corresponding topic in the message. Each condition may require: a) an exact
// topic match; b) a match from a set of topics; or c) a wild-card matching all.
//
// If a message contains more topics than required by the matcher, those beyond
// the condition count are ignored and assumed to match.
//
// Consider the following sample topic matcher:
//   sample := {
//     {TopicA1, TopicA2, TopicA3},
//     {TopicB},
//     nil,
//     {TopicD1, TopicD2}
//   }
// In order for a message to pass this filter, it should enumerate at least 4
// topics, the first any of [TopicA1, TopicA2, TopicA3], the second mandatory
// "TopicB", the third is ignored by the filter and the fourth either "TopicD1"
// or "TopicD2". If the message contains further topics, the filter will match
// them too.
type topicMatcher struct {
	conditions []map[Topic]struct{}
}

// newTopicMatcher create a topic matcher from a list of topic conditions.
func newTopicMatcher(topics ...[]Topic) *topicMatcher {
	matcher := make([]map[Topic]struct{}, len(topics))
	for i, condition := range topics {
		matcher[i] = make(map[Topic]struct{})
		for _, topic := range condition {
			matcher[i][topic] = struct{}{}
		}
	}
	return &topicMatcher{conditions: matcher}
}

// newTopicMatcherFromBinary create a topic matcher from a list of binary conditions.
func newTopicMatcherFromBinary(data ...[][]byte) *topicMatcher {
	topics := make([][]Topic, len(data))
	for i, condition := range data {
		topics[i] = NewTopics(condition...)
	}
	return newTopicMatcher(topics...)
}

// newTopicMatcherFromStrings creates a topic matcher from a list of textual
// conditions.
func newTopicMatcherFromStrings(data ...[]string) *topicMatcher {
	topics := make([][]Topic, len(data))
	for i, condition := range data {
		topics[i] = NewTopicsFromStrings(condition...)
	}
	return newTopicMatcher(topics...)
}

// Matches checks if a list of topics matches this particular condition set.
func (self *topicMatcher) Matches(topics []Topic) bool {
	// Mismatch if there aren't enough topics
	if len(self.conditions) > len(topics) {
		return false
	}
	// Check each topic condition for existence (skip wild-cards)
	for i := 0; i < len(topics) && i < len(self.conditions); i++ {
		if len(self.conditions[i]) > 0 {
			if _, ok := self.conditions[i][topics[i]]; !ok {
				return false
			}
		}
	}
	return true
}
