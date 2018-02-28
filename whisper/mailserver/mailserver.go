// Copyright 2017 The go-ethereum Authors
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

package mailserver

import (
	"encoding/binary"
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv6"
	"github.com/syndtr/goleveldb/leveldb"
	"github.com/syndtr/goleveldb/leveldb/util"
)

type WMailServer struct {
	db  *leveldb.DB
	w   *whisper.Whisper
	pow float64
	key []byte
}

type DBKey struct {
	timestamp uint32
	hash      common.Hash
	raw       []byte
}

func NewDbKey(t uint32, h common.Hash) *DBKey {
	const sz = common.HashLength + 4
	var k DBKey
	k.timestamp = t
	k.hash = h
	k.raw = make([]byte, sz)
	binary.BigEndian.PutUint32(k.raw, k.timestamp)
	copy(k.raw[4:], k.hash[:])
	return &k
}

func (s *WMailServer) Init(shh *whisper.Whisper, path string, password string, pow float64) {
	var err error
	if len(path) == 0 {
		utils.Fatalf("DB file is not specified")
	}

	if len(password) == 0 {
		utils.Fatalf("Password is not specified for MailServer")
	}

	s.db, err = leveldb.OpenFile(path, nil)
	if err != nil {
		utils.Fatalf("Failed to open DB file: %s", err)
	}

	s.w = shh
	s.pow = pow

	MailServerKeyID, err := s.w.AddSymKeyFromPassword(password)
	if err != nil {
		utils.Fatalf("Failed to create symmetric key for MailServer: %s", err)
	}
	s.key, err = s.w.GetSymKey(MailServerKeyID)
	if err != nil {
		utils.Fatalf("Failed to save symmetric key for MailServer")
	}
}

func (s *WMailServer) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *WMailServer) Archive(env *whisper.Envelope) {
	key := NewDbKey(env.Expiry-env.TTL, env.Hash())
	rawEnvelope, err := rlp.EncodeToBytes(env)
	if err != nil {
		log.Error(fmt.Sprintf("rlp.EncodeToBytes failed: %s", err))
	} else {
		err = s.db.Put(key.raw, rawEnvelope, nil)
		if err != nil {
			log.Error(fmt.Sprintf("Writing to DB failed: %s", err))
		}
	}
}

func (s *WMailServer) DeliverMail(peer *whisper.Peer, request *whisper.Envelope) {
	if peer == nil {
		log.Error("Whisper peer is nil")
		return
	}

	ok, lower, upper, topic := s.validateRequest(peer.ID(), request)
	if ok {
		s.processRequest(peer, lower, upper, topic)
	}
}

func (s *WMailServer) processRequest(peer *whisper.Peer, lower, upper uint32, topic whisper.TopicType) []*whisper.Envelope {
	ret := make([]*whisper.Envelope, 0)
	var err error
	var zero common.Hash
	var empty whisper.TopicType
	kl := NewDbKey(lower, zero)
	ku := NewDbKey(upper, zero)
	i := s.db.NewIterator(&util.Range{Start: kl.raw, Limit: ku.raw}, nil)
	defer i.Release()

	for i.Next() {
		var envelope whisper.Envelope
		err = rlp.DecodeBytes(i.Value(), &envelope)
		if err != nil {
			log.Error(fmt.Sprintf("RLP decoding failed: %s", err))
		}

		if topic == empty || envelope.Topic == topic {
			if peer == nil {
				// used for test purposes
				ret = append(ret, &envelope)
			} else {
				err = s.w.SendP2PDirect(peer, &envelope)
				if err != nil {
					log.Error(fmt.Sprintf("Failed to send direct message to peer: %s", err))
					return nil
				}
			}
		}
	}

	err = i.Error()
	if err != nil {
		log.Error(fmt.Sprintf("Level DB iterator error: %s", err))
	}

	return ret
}

func (s *WMailServer) validateRequest(peerID []byte, request *whisper.Envelope) (bool, uint32, uint32, whisper.TopicType) {
	var topic whisper.TopicType
	if s.pow > 0.0 && request.PoW() < s.pow {
		return false, 0, 0, topic
	}

	f := whisper.Filter{KeySym: s.key}
	decrypted := request.Open(&f)
	if decrypted == nil {
		log.Warn(fmt.Sprintf("Failed to decrypt p2p request"))
		return false, 0, 0, topic
	}

	if len(decrypted.Payload) < 8 {
		log.Warn(fmt.Sprintf("Undersized p2p request"))
		return false, 0, 0, topic
	}

	src := crypto.FromECDSAPub(decrypted.Src)
	if len(src)-len(peerID) == 1 {
		src = src[1:]
	}

	// if you want to check the signature, you can do it here. e.g.:
	// if !bytes.Equal(peerID, src) {
	if src == nil {
		log.Warn(fmt.Sprintf("Wrong signature of p2p request"))
		return false, 0, 0, topic
	}

	lower := binary.BigEndian.Uint32(decrypted.Payload[:4])
	upper := binary.BigEndian.Uint32(decrypted.Payload[4:8])

	if len(decrypted.Payload) >= 8+whisper.TopicLength {
		topic = whisper.BytesToTopic(decrypted.Payload[8:])
	}

	return true, lower, upper, topic
}
