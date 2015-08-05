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

// Contains the message filter for fine grained subscriptions.

package whisper

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/event/filter"
)

// Filter is used to subscribe to specific types of whisper messages.
type Filter struct {
	To     *ecdsa.PublicKey   // Recipient of the message
	From   *ecdsa.PublicKey   // Sender of the message
	Topics [][]Topic          // Topics to filter messages with
	Fn     func(msg *Message) // Handler in case of a match
}

// NewFilterTopics creates a 2D topic array used by whisper.Filter from binary
// data elements.
func NewFilterTopics(data ...[][]byte) [][]Topic {
	filter := make([][]Topic, len(data))
	for i, condition := range data {
		// Handle the special case of condition == [[]byte{}]
		if len(condition) == 1 && len(condition[0]) == 0 {
			filter[i] = []Topic{}
			continue
		}
		// Otherwise flatten normally
		filter[i] = NewTopics(condition...)
	}
	return filter
}

// NewFilterTopicsFlat creates a 2D topic array used by whisper.Filter from flat
// binary data elements.
func NewFilterTopicsFlat(data ...[]byte) [][]Topic {
	filter := make([][]Topic, len(data))
	for i, element := range data {
		// Only add non-wildcard topics
		filter[i] = make([]Topic, 0, 1)
		if len(element) > 0 {
			filter[i] = append(filter[i], NewTopic(element))
		}
	}
	return filter
}

// NewFilterTopicsFromStrings creates a 2D topic array used by whisper.Filter
// from textual data elements.
func NewFilterTopicsFromStrings(data ...[]string) [][]Topic {
	filter := make([][]Topic, len(data))
	for i, condition := range data {
		// Handle the special case of condition == [""]
		if len(condition) == 1 && condition[0] == "" {
			filter[i] = []Topic{}
			continue
		}
		// Otherwise flatten normally
		filter[i] = NewTopicsFromStrings(condition...)
	}
	return filter
}

// NewFilterTopicsFromStringsFlat creates a 2D topic array used by whisper.Filter from flat
// binary data elements.
func NewFilterTopicsFromStringsFlat(data ...string) [][]Topic {
	filter := make([][]Topic, len(data))
	for i, element := range data {
		// Only add non-wildcard topics
		filter[i] = make([]Topic, 0, 1)
		if element != "" {
			filter[i] = append(filter[i], NewTopicFromString(element))
		}
	}
	return filter
}

// filterer is the internal, fully initialized filter ready to match inbound
// messages to a variety of criteria.
type filterer struct {
	to      string                 // Recipient of the message
	from    string                 // Sender of the message
	matcher *topicMatcher          // Topics to filter messages with
	fn      func(data interface{}) // Handler in case of a match
}

// Compare checks if the specified filter matches the current one.
func (self filterer) Compare(f filter.Filter) bool {
	filter := f.(filterer)

	// Check the message sender and recipient
	if len(self.to) > 0 && self.to != filter.to {
		return false
	}
	if len(self.from) > 0 && self.from != filter.from {
		return false
	}
	// Check the topic filtering
	topics := make([]Topic, len(filter.matcher.conditions))
	for i, group := range filter.matcher.conditions {
		// Message should contain a single topic entry, extract
		for topics[i], _ = range group {
			break
		}
	}
	if !self.matcher.Matches(topics) {
		return false
	}
	return true
}

// Trigger is called when a filter successfully matches an inbound message.
func (self filterer) Trigger(data interface{}) {
	self.fn(data)
}
