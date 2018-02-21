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
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
)

const ALL_TOPICS = ""

type Filter struct {
	Src        *ecdsa.PublicKey  // Sender of the message
	KeyAsym    *ecdsa.PrivateKey // Private Key of recipient
	KeySym     []byte            // Key associated with the Topic
	Topics     [][]byte          // Topics to filter messages with
	PoW        float64           // Proof of work as described in the Whisper spec
	AllowP2P   bool              // Indicates whether this filter is interested in direct peer-to-peer messages
	SymKeyHash common.Hash       // The Keccak256Hash of the symmetric key, needed for optimization

	Messages map[common.Hash]*ReceivedMessage
	mutex    sync.RWMutex
}

type Filters struct {
	watchers       map[string]*Filter
	watchersTopics map[string]map[string]struct{}
	whisper        *Whisper
	mutex          sync.RWMutex
}

func NewFilters(w *Whisper) *Filters {
	fs := &Filters{
		watchers:       make(map[string]*Filter),
		watchersTopics: make(map[string]map[string]struct{}),
		whisper:        w,
	}
	fs.watchersTopics[ALL_TOPICS] = make(map[string]struct{})
	return fs
}

func (fs *Filters) Install(watcher *Filter) (string, error) {
	if watcher.Messages == nil {
		watcher.Messages = make(map[common.Hash]*ReceivedMessage)
	}

	id, err := GenerateRandomID()
	if err != nil {
		return "", err
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	if fs.watchers[id] != nil {
		return "", fmt.Errorf("failed to generate unique ID")
	}

	if watcher.expectsSymmetricEncryption() {
		watcher.SymKeyHash = crypto.Keccak256Hash(watcher.KeySym)
	}

	fs.watchers[id] = watcher
	fs.addFilterToTopicsMapping(watcher, id)
	return id, err
}

func (fs *Filters) addFilterToTopicsMapping(watcher *Filter, id string) {
	for i := range fs.prepareTopicsMapping(watcher) {
		topicMapping, ok := fs.watchersTopics[i]
		if !ok {
			fs.watchersTopics[i] = make(map[string]struct{})
			topicMapping = fs.watchersTopics[i]
		}
		topicMapping[id] = struct{}{}
	}
}

func (fs *Filters) removeTopicFromTopicMapping(id string) {
	for i := range fs.watchersTopics {
		delete(fs.watchersTopics[i], id)
	}
}

func (fs *Filters) prepareTopicsMapping(watcher *Filter) map[string]struct{} {
	topics := make(map[string]struct{}, len(watcher.Topics))

	if len(watcher.Topics) == 0 {
		topics[ALL_TOPICS] = struct{}{}
		return topics
	}

	for _, topic := range watcher.Topics {
		topics[common.ToHex(topic)] = struct{}{}
	}

	return topics
}

func (fs *Filters) matchedTopics(topic TopicType) map[string]struct{} {
	m := fs.watchersTopics[ALL_TOPICS]
	for i := range fs.watchersTopics[topic.String()] {
		m[i] = struct{}{}
	}
	return m
}

func (fs *Filters) Uninstall(id string) bool {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	if fs.watchers[id] != nil {
		delete(fs.watchers, id)
		fs.removeTopicFromTopicMapping(id)
		return true
	}
	return false
}

func (fs *Filters) Get(id string) *Filter {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.watchers[id]
}

func (fs *Filters) NotifyWatchers(env *Envelope, p2pMessage bool) {
	var msg *ReceivedMessage

	fs.mutex.RLock()
	defer fs.mutex.RUnlock()

	for watcherID := range fs.matchedTopics(env.Topic) {
		watcher, ok := fs.watchers[watcherID]
		if !ok {
			log.Trace(fmt.Sprintf("msg [%x], filter [%s]: filter not exists", env.Hash(), watcherID))
			continue
		}

		if p2pMessage && !watcher.AllowP2P {
			log.Trace(fmt.Sprintf("msg [%x], filter [%s]: p2p messages are not allowed", env.Hash(), watcherID))
			continue
		}

		var match bool
		if msg != nil {
			match = watcher.MatchMessage(msg)
		} else {
			match = watcher.MatchEnvelope(env)
			if match {
				msg = env.Open(watcher)
				if msg == nil {
					log.Trace("processing message: failed to open", "message", env.Hash().Hex(), "filter", watcherID)
				}
			} else {
				log.Trace("processing message: does not match", "message", env.Hash().Hex(), "filter", watcherID)
			}
		}

		if match && msg != nil {
			log.Trace("processing message: decrypted", "hash", env.Hash().Hex())
			if watcher.Src == nil || IsPubKeyEqual(msg.Src, watcher.Src) {
				watcher.Trigger(msg)
			}
		}
	}
}

func (f *Filter) processEnvelope(env *Envelope) *ReceivedMessage {
	if f.MatchEnvelope(env) {
		msg := env.Open(f)
		if msg != nil {
			return msg
		} else {
			log.Trace("processing envelope: failed to open", "hash", env.Hash().Hex())
		}
	} else {
		log.Trace("processing envelope: does not match", "hash", env.Hash().Hex())
	}
	return nil
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

	if f.expectsAsymmetricEncryption() && msg.isAsymmetricEncryption() {
		return IsPubKeyEqual(&f.KeyAsym.PublicKey, msg.Dst)
	} else if f.expectsSymmetricEncryption() && msg.isSymmetricEncryption() {
		return f.SymKeyHash == msg.SymKeyHash
	}
	return false
}

func (f *Filter) MatchEnvelope(envelope *Envelope) bool {
	if f.PoW > 0 && envelope.pow < f.PoW {
		return false
	}

	if f.expectsAsymmetricEncryption() && envelope.isAsymmetric() {
		return true
	} else if f.expectsSymmetricEncryption() && envelope.IsSymmetric() {
		return true
	}
	return false
}

func IsPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	if !ValidatePublicKey(a) {
		return false
	} else if !ValidatePublicKey(b) {
		return false
	}
	// the curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}
