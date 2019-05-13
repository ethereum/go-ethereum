// Copyright 2019 The go-ethereum Authors
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

package pss

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/swarm/log"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
)

type KeyStore struct {
	w *whisper.Whisper // key and encryption backend

	mx                       sync.RWMutex
	pubKeyPool               map[string]map[Topic]*pssPeer // mapping of hex public keys to peer address by topic.
	symKeyPool               map[string]map[Topic]*pssPeer // mapping of symkeyids to peer address by topic.
	symKeyDecryptCache       []*string                     // fast lookup of symkeys recently used for decryption; last used is on top of stack
	symKeyDecryptCacheCursor int                           // modular cursor pointing to last used, wraps on symKeyDecryptCache array
}

func loadKeyStore() *KeyStore {
	return &KeyStore{
		w: whisper.New(&whisper.DefaultConfig),

		pubKeyPool:         make(map[string]map[Topic]*pssPeer),
		symKeyPool:         make(map[string]map[Topic]*pssPeer),
		symKeyDecryptCache: make([]*string, defaultSymKeyCacheCapacity),
	}
}

func (ks *KeyStore) isSymKeyStored(key string) bool {
	ks.mx.RLock()
	defer ks.mx.RUnlock()
	var ok bool
	_, ok = ks.symKeyPool[key]
	return ok
}

func (ks *KeyStore) isPubKeyStored(key string) bool {
	ks.mx.RLock()
	defer ks.mx.RUnlock()
	var ok bool
	_, ok = ks.pubKeyPool[key]
	return ok
}

func (ks *KeyStore) getPeerSym(symkeyid string, topic Topic) (*pssPeer, bool) {
	ks.mx.RLock()
	defer ks.mx.RUnlock()
	psp, ok := ks.symKeyPool[symkeyid][topic]
	return psp, ok
}

func (ks *KeyStore) getPeerPub(pubkeyid string, topic Topic) (*pssPeer, bool) {
	ks.mx.RLock()
	defer ks.mx.RUnlock()
	psp, ok := ks.pubKeyPool[pubkeyid][topic]
	return psp, ok
}

// Links a peer ECDSA public key to a topic.
// This is required for asymmetric message exchange on the given topic.
// The value in `address` will be used as a routing hint for the public key / topic association.
func (ks *KeyStore) SetPeerPublicKey(pubkey *ecdsa.PublicKey, topic Topic, address PssAddress) error {
	if err := validateAddress(address); err != nil {
		return err
	}
	pubkeybytes := crypto.FromECDSAPub(pubkey)
	if len(pubkeybytes) == 0 {
		return fmt.Errorf("invalid public key: %v", pubkey)
	}
	pubkeyid := common.ToHex(pubkeybytes)
	psp := &pssPeer{
		address: address,
	}
	ks.mx.Lock()
	if _, ok := ks.pubKeyPool[pubkeyid]; !ok {
		ks.pubKeyPool[pubkeyid] = make(map[Topic]*pssPeer)
	}
	ks.pubKeyPool[pubkeyid][topic] = psp
	ks.mx.Unlock()
	log.Trace("added pubkey", "pubkeyid", pubkeyid, "topic", topic, "address", address)
	return nil
}

// adds a symmetric key to the pss key pool, and optionally adds the key to the
// collection of keys used to attempt symmetric decryption of incoming messages
func (ks *KeyStore) addSymmetricKeyToPool(keyid string, topic Topic, address PssAddress, addtocache bool, protected bool) {
	psp := &pssPeer{
		address:   address,
		protected: protected,
	}
	ks.mx.Lock()
	if _, ok := ks.symKeyPool[keyid]; !ok {
		ks.symKeyPool[keyid] = make(map[Topic]*pssPeer)
	}
	ks.symKeyPool[keyid][topic] = psp
	ks.mx.Unlock()
	if addtocache {
		ks.symKeyDecryptCacheCursor++
		ks.symKeyDecryptCache[ks.symKeyDecryptCacheCursor%cap(ks.symKeyDecryptCache)] = &keyid
	}
}

// Returns all recorded topic and address combination for a specific public key
func (ks *KeyStore) GetPublickeyPeers(keyid string) (topic []Topic, address []PssAddress, err error) {
	ks.mx.RLock()
	defer ks.mx.RUnlock()
	for t, peer := range ks.pubKeyPool[keyid] {
		topic = append(topic, t)
		address = append(address, peer.address)
	}
	return topic, address, nil
}

func (ks *KeyStore) getPeerAddress(keyid string, topic Topic) (PssAddress, error) {
	ks.mx.RLock()
	defer ks.mx.RUnlock()
	if peers, ok := ks.pubKeyPool[keyid]; ok {
		if t, ok := peers[topic]; ok {
			return t.address, nil
		}
	}
	return nil, fmt.Errorf("peer with pubkey %s, topic %x not found", keyid, topic)
}

// Attempt to decrypt, validate and unpack a symmetrically encrypted message.
// If successful, returns the unpacked whisper ReceivedMessage struct
// encapsulating the decrypted message, and the whisper backend id
// of the symmetric key used to decrypt the message.
// It fails if decryption of the message fails or if the message is corrupted.
func (ks *KeyStore) processSym(envelope *whisper.Envelope) (*whisper.ReceivedMessage, string, PssAddress, error) {
	metrics.GetOrRegisterCounter("pss.process.sym", nil).Inc(1)

	for i := ks.symKeyDecryptCacheCursor; i > ks.symKeyDecryptCacheCursor-cap(ks.symKeyDecryptCache) && i > 0; i-- {
		symkeyid := ks.symKeyDecryptCache[i%cap(ks.symKeyDecryptCache)]
		symkey, err := ks.w.GetSymKey(*symkeyid)
		if err != nil {
			continue
		}
		recvmsg, err := envelope.OpenSymmetric(symkey)
		if err != nil {
			continue
		}
		if !recvmsg.ValidateAndParse() {
			return nil, "", nil, errors.New("symmetrically encrypted message has invalid signature or is corrupt")
		}
		var from PssAddress
		ks.mx.RLock()
		if ks.symKeyPool[*symkeyid][Topic(envelope.Topic)] != nil {
			from = ks.symKeyPool[*symkeyid][Topic(envelope.Topic)].address
		}
		ks.mx.RUnlock()
		ks.symKeyDecryptCacheCursor++
		ks.symKeyDecryptCache[ks.symKeyDecryptCacheCursor%cap(ks.symKeyDecryptCache)] = symkeyid
		return recvmsg, *symkeyid, from, nil
	}
	return nil, "", nil, errors.New("could not decrypt message")
}

// Attempt to decrypt, validate and unpack an asymmetrically encrypted message.
// If successful, returns the unpacked whisper ReceivedMessage struct
// encapsulating the decrypted message, and the byte representation of
// the public key used to decrypt the message.
// It fails if decryption of message fails, or if the message is corrupted.
func (ks *Pss) processAsym(envelope *whisper.Envelope) (*whisper.ReceivedMessage, string, PssAddress, error) {
	metrics.GetOrRegisterCounter("pss.process.asym", nil).Inc(1)

	recvmsg, err := envelope.OpenAsymmetric(ks.privateKey)
	if err != nil {
		return nil, "", nil, fmt.Errorf("could not decrypt message: %s", err)
	}
	// check signature (if signed), strip padding
	if !recvmsg.ValidateAndParse() {
		return nil, "", nil, errors.New("invalid message")
	}
	pubkeyid := common.ToHex(crypto.FromECDSAPub(recvmsg.Src))
	var from PssAddress
	ks.mx.RLock()
	if ks.pubKeyPool[pubkeyid][Topic(envelope.Topic)] != nil {
		from = ks.pubKeyPool[pubkeyid][Topic(envelope.Topic)].address
	}
	ks.mx.RUnlock()
	return recvmsg, pubkeyid, from, nil
}

// Symkey garbage collection
// a key is removed if:
// - it is not marked as protected
// - it is not in the incoming decryption cache
func (ks *Pss) cleanKeys() (count int) {
	ks.mx.Lock()
	defer ks.mx.Unlock()
	for keyid, peertopics := range ks.symKeyPool {
		var expiredtopics []Topic
		for topic, psp := range peertopics {
			if psp.protected {
				continue
			}

			var match bool
			for i := ks.symKeyDecryptCacheCursor; i > ks.symKeyDecryptCacheCursor-cap(ks.symKeyDecryptCache) && i > 0; i-- {
				cacheid := ks.symKeyDecryptCache[i%cap(ks.symKeyDecryptCache)]
				if *cacheid == keyid {
					match = true
				}
			}
			if !match {
				expiredtopics = append(expiredtopics, topic)
			}
		}
		for _, topic := range expiredtopics {
			delete(ks.symKeyPool[keyid], topic)
			log.Trace("symkey cleanup deletion", "symkeyid", keyid, "topic", topic, "val", ks.symKeyPool[keyid])
			count++
		}
	}
	return count
}

// Automatically generate a new symkey for a topic and address hint
func (ks *KeyStore) GenerateSymmetricKey(topic Topic, address PssAddress, addToCache bool) (string, error) {
	keyid, err := ks.w.GenerateSymKey()
	if err == nil {
		ks.addSymmetricKeyToPool(keyid, topic, address, addToCache, false)
	}
	return keyid, err
}

// Returns a symmetric key byte sequence stored in the whisper backend by its unique id.
// Passes on the error value from the whisper backend.
func (ks *KeyStore) GetSymmetricKey(symkeyid string) ([]byte, error) {
	return ks.w.GetSymKey(symkeyid)
}

// Links a peer symmetric key (arbitrary byte sequence) to a topic.
//
// This is required for symmetrically encrypted message exchange on the given topic.
//
// The key is stored in the whisper backend.
//
// If addtocache is set to true, the key will be added to the cache of keys
// used to attempt symmetric decryption of incoming messages.
//
// Returns a string id that can be used to retrieve the key bytes
// from the whisper backend (see pss.GetSymmetricKey())
func (ks *KeyStore) SetSymmetricKey(key []byte, topic Topic, address PssAddress, addtocache bool) (string, error) {
	if err := validateAddress(address); err != nil {
		return "", err
	}
	return ks.setSymmetricKey(key, topic, address, addtocache, true)
}

func (ks *KeyStore) setSymmetricKey(key []byte, topic Topic, address PssAddress, addtocache bool, protected bool) (string, error) {
	keyid, err := ks.w.AddSymKeyDirect(key)
	if err == nil {
		ks.addSymmetricKeyToPool(keyid, topic, address, addtocache, protected)
	}
	return keyid, err
}
