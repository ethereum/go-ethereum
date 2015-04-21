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
