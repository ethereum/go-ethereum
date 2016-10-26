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

package whisperv5

import (
	"crypto/ecdsa"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type Filter struct {
	Src        *ecdsa.PublicKey  // Sender of the message
	KeyAsym    *ecdsa.PrivateKey // Private Key of recipient
	KeySym     []byte            // Key associated with the Topic
	Topics     []TopicType       // Topics to filter messages with
	PoW        float64           // Proof of work as described in the Whisper spec
	AcceptP2P  bool              // Indicates whether this filter is interested in direct peer-to-peer messages
	SymKeyHash common.Hash       // The Keccak256Hash of the symmetric key, needed for optimization

	Messages map[common.Hash]*ReceivedMessage
	mutex    sync.RWMutex
}

type Filters struct {
	id       int
	watchers map[int]*Filter
	whisper  *Whisper
	mutex    sync.RWMutex
}

func NewFilters(w *Whisper) *Filters {
	return &Filters{
		watchers: make(map[int]*Filter),
		whisper:  w,
	}
}

func (fs *Filters) Install(watcher *Filter) int {
	if watcher.Messages == nil {
		watcher.Messages = make(map[common.Hash]*ReceivedMessage)
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	fs.watchers[fs.id] = watcher
	ret := fs.id
	fs.id++
	return ret
}

func (fs *Filters) Uninstall(id int) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	delete(fs.watchers, id)
}

func (fs *Filters) Get(i int) *Filter {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.watchers[i]
}

func (fs *Filters) NotifyWatchers(env *Envelope, messageCode uint64) {
	fs.mutex.RLock()
	var msg *ReceivedMessage
	for _, watcher := range fs.watchers {
		if messageCode == p2pCode && !watcher.AcceptP2P {
			continue
		}

		match := false
		if msg != nil {
			match = watcher.MatchMessage(msg)
		} else {
			match = watcher.MatchEnvelope(env)
			if match {
				msg = env.Open(watcher)
			}
		}

		if match && msg != nil {
			watcher.Trigger(msg)
		}
	}
	fs.mutex.RUnlock() // we need to unlock before calling addDecryptedMessage

	if msg != nil {
		fs.whisper.addDecryptedMessage(msg)
	}
}

func (f *Filter) expectsAsymmetricEncryption() bool {
	return f.KeyAsym != nil
}

func (f *Filter) expectsSymmetricEncryption() bool {
	return f.KeySym != nil
}

func (f *Filter) Trigger(msg *ReceivedMessage) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if _, exist := f.Messages[msg.EnvelopeHash]; !exist {
		f.Messages[msg.EnvelopeHash] = msg
	}
}

func (f *Filter) Retrieve() (all []*ReceivedMessage) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	all = make([]*ReceivedMessage, 0, len(f.Messages))
	for _, msg := range f.Messages {
		all = append(all, msg)
	}
	f.Messages = make(map[common.Hash]*ReceivedMessage) // delete old messages
	return all
}

func (f *Filter) MatchMessage(msg *ReceivedMessage) bool {
	if f.PoW > 0 && msg.PoW < f.PoW {
		return false
	}
	if f.Src != nil && !isPubKeyEqual(msg.Src, f.Src) {
		return false
	}

	if f.expectsAsymmetricEncryption() && msg.isAsymmetricEncryption() {
		// if Dst match, ignore the topic
		return isPubKeyEqual(&f.KeyAsym.PublicKey, msg.Dst)
	} else if f.expectsSymmetricEncryption() && msg.isSymmetricEncryption() {
		// check if that both the key and the topic match
		if f.SymKeyHash == msg.SymKeyHash {
			for _, t := range f.Topics {
				if t == msg.Topic {
					return true
				}
			}
			return false
		}
	}
	return false
}

func (f *Filter) MatchEnvelope(envelope *Envelope) bool {
	if f.PoW > 0 && envelope.pow < f.PoW {
		return false
	}

	encryptionMethodMatch := false
	if f.expectsAsymmetricEncryption() && envelope.isAsymmetric() {
		encryptionMethodMatch = true
		if f.Topics == nil {
			// wildcard
			return true
		}
	} else if f.expectsSymmetricEncryption() && envelope.IsSymmetric() {
		encryptionMethodMatch = true
	}

	if encryptionMethodMatch {
		for _, t := range f.Topics {
			if t == envelope.Topic {
				return true
			}
		}
	}

	return false
}

func isPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	if !ValidatePublicKey(a) {
		return false
	} else if !ValidatePublicKey(b) {
		return false
	}
	// the Curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}
