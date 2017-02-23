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
	crand "crypto/rand"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
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
	watchers map[string]*Filter
	whisper  *Whisper
	mutex    sync.RWMutex
}

func NewFilters(w *Whisper) *Filters {
	return &Filters{
		watchers: make(map[string]*Filter),
		whisper:  w,
	}
}

func (fs *Filters) generateRandomID() (id string, err error) {
	buf := make([]byte, 20)
	for i := 0; i < 3; i++ {
		_, err = crand.Read(buf)
		if err != nil {
			continue
		}
		if !validateSymmetricKey(buf) {
			err = fmt.Errorf("error in generateRandomID: crypto/rand failed to generate random data")
			continue
		}
		id = common.Bytes2Hex(buf)
		if fs.watchers[id] != nil {
			err = fmt.Errorf("error in generateRandomID: generated same ID twice")
			continue
		}
		return id, err
	}

	return "", err
}

func (fs *Filters) Install(watcher *Filter) (string, error) {
	if watcher.Messages == nil {
		watcher.Messages = make(map[common.Hash]*ReceivedMessage)
	}

	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	id, err := fs.generateRandomID()
	if err == nil {
		fs.watchers[id] = watcher
	}
	return id, err
}

func (fs *Filters) Uninstall(id string) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()
	delete(fs.watchers, id)
}

func (fs *Filters) Get(id string) *Filter {
	fs.mutex.RLock()
	defer fs.mutex.RUnlock()
	return fs.watchers[id]
}

func (fs *Filters) NotifyWatchers(env *Envelope, p2pMessage bool) {
	fs.mutex.RLock()
	var msg *ReceivedMessage
	for j, watcher := range fs.watchers {
		if p2pMessage && !watcher.AcceptP2P {
			glog.V(logger.Detail).Infof("msg [%x], filter [%d]: p2p messages are not allowed \n", env.Hash(), j)
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
					glog.V(logger.Detail).Infof("msg [%x], filter [%d]: failed to open \n", env.Hash(), j)
				}
			} else {
				glog.V(logger.Detail).Infof("msg [%x], filter [%d]: does not match \n", env.Hash(), j)
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
	if f.Src != nil && !IsPubKeyEqual(msg.Src, f.Src) {
		return false
	}

	if f.expectsAsymmetricEncryption() && msg.isAsymmetricEncryption() {
		return IsPubKeyEqual(&f.KeyAsym.PublicKey, msg.Dst) && f.MatchTopic(msg.Topic)
	} else if f.expectsSymmetricEncryption() && msg.isSymmetricEncryption() {
		return f.SymKeyHash == msg.SymKeyHash && f.MatchTopic(msg.Topic)
	}
	return false
}

func (f *Filter) MatchEnvelope(envelope *Envelope) bool {
	if f.PoW > 0 && envelope.pow < f.PoW {
		return false
	}

	if f.expectsAsymmetricEncryption() && envelope.isAsymmetric() {
		return f.MatchTopic(envelope.Topic)
	} else if f.expectsSymmetricEncryption() && envelope.IsSymmetric() {
		return f.MatchTopic(envelope.Topic)
	}
	return false
}

func (f *Filter) MatchTopic(topic TopicType) bool {
	if len(f.Topics) == 0 {
		// any topic matches
		return true
	}

	for _, t := range f.Topics {
		if t == topic {
			return true
		}
	}
	return false
}

func IsPubKeyEqual(a, b *ecdsa.PublicKey) bool {
	if !ValidatePublicKey(a) {
		return false
	} else if !ValidatePublicKey(b) {
		return false
	}
	// the Curve is always the same, just compare the points
	return a.X.Cmp(b.X) == 0 && a.Y.Cmp(b.Y) == 0
}
