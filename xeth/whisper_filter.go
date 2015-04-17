// Contains the external API side message filter for watching, pooling and polling
// matched whisper messages.

package xeth

import "time"

// whisperFilter is the message cache matching a specific filter, accumulating
// inbound messages until the are requested by the client.
type whisperFilter struct {
	id      int              // Filter identifier
	cache   []WhisperMessage // Cache of messages not yet polled
	timeout time.Time        // Time when the last message batch was queries
}

// insert injects a new batch of messages into the filter cache.
func (w *whisperFilter) insert(msgs ...WhisperMessage) {
	w.cache = append(w.cache, msgs...)
}

// retrieve fetches all the cached messages from the filter.
func (w *whisperFilter) retrieve() (messages []WhisperMessage) {
	messages, w.cache = w.cache, nil
	w.timeout = time.Now()
	return
}
