// Contains the external API side message filter for watching, pooling and polling
// matched whisper messages, also serializing data access to avoid duplications.

package xeth

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// whisperFilter is the message cache matching a specific filter, accumulating
// inbound messages until the are requested by the client.
type whisperFilter struct {
	id  int      // Filter identifier for old message retrieval
	ref *Whisper // Whisper reference for old message retrieval

	cache  []WhisperMessage         // Cache of messages not yet polled
	skip   map[common.Hash]struct{} // List of retrieved messages to avoid duplication
	update time.Time                // Time of the last message query

	lock sync.RWMutex // Lock protecting the filter internals
}

// newWhisperFilter creates a new serialized, poll based whisper topic filter.
func newWhisperFilter(id int, ref *Whisper) *whisperFilter {
	return &whisperFilter{
		id:  id,
		ref: ref,

		update: time.Now(),
		skip:   make(map[common.Hash]struct{}),
	}
}

// messages retrieves all the cached messages from the entire pool matching the
// filter, resetting the filter's change buffer.
func (w *whisperFilter) messages() []WhisperMessage {
	w.lock.Lock()
	defer w.lock.Unlock()

	w.cache = nil
	w.update = time.Now()

	w.skip = make(map[common.Hash]struct{})
	messages := w.ref.Messages(w.id)
	for _, message := range messages {
		w.skip[message.ref.Hash] = struct{}{}
	}
	return messages
}

// insert injects a new batch of messages into the filter cache.
func (w *whisperFilter) insert(messages ...WhisperMessage) {
	w.lock.Lock()
	defer w.lock.Unlock()

	for _, message := range messages {
		if _, ok := w.skip[message.ref.Hash]; !ok {
			w.cache = append(w.cache, messages...)
		}
	}
}

// retrieve fetches all the cached messages from the filter.
func (w *whisperFilter) retrieve() (messages []WhisperMessage) {
	w.lock.Lock()
	defer w.lock.Unlock()

	messages, w.cache = w.cache, nil
	w.update = time.Now()

	return
}

// activity returns the last time instance when client requests were executed on
// the filter.
func (w *whisperFilter) activity() time.Time {
	w.lock.RLock()
	defer w.lock.RUnlock()

	return w.update
}
