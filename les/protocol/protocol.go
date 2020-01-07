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

// Package protocol defines all protocol related structures which will be
// used in both server side and client side.
package protocol

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

// Constants to match up protocol versions and messages
const (
	Lpv2 = 2
	Lpv3 = 3
)

// Supported versions of the les protocol (first is primary)
var (
	ClientProtocolVersions    = []uint{Lpv2, Lpv3}
	ServerProtocolVersions    = []uint{Lpv2, Lpv3}
	AdvertiseProtocolVersions = []uint{Lpv2} // clients are searching for the first advertised protocol in the list
)

// Number of implemented message corresponding to different protocol versions.
var ProtocolLengths = map[uint]uint64{Lpv2: 22, Lpv3: 24}

const (
	NetworkId          = 1                // Default ethereum mainnet network ID
	ProtocolMaxMsgSize = 10 * 1024 * 1024 // Maximum cap on the size of a protocol message
)

// les protocol message codes
const (
	// Protocol messages inherited from LPV1
	StatusMsg          = 0x00
	AnnounceMsg        = 0x01
	GetBlockHeadersMsg = 0x02
	BlockHeadersMsg    = 0x03
	GetBlockBodiesMsg  = 0x04
	BlockBodiesMsg     = 0x05
	GetReceiptsMsg     = 0x06
	ReceiptsMsg        = 0x07
	GetCodeMsg         = 0x0a
	CodeMsg            = 0x0b

	// Protocol messages introduced in LPV2
	GetProofsV2Msg         = 0x0f
	ProofsV2Msg            = 0x10
	GetHelperTrieProofsMsg = 0x11
	HelperTrieProofsMsg    = 0x12
	SendTxV2Msg            = 0x13
	GetTxStatusMsg         = 0x14
	TxStatusMsg            = 0x15

	// Protocol messages introduced in LPV3
	StopMsg   = 0x16
	ResumeMsg = 0x17
)

// The maxmium amount of data requested per retrieval request.
const (
	MaxHeaderFetch           = 192 // Amount of block headers to be fetched per retrieval request
	MaxBodyFetch             = 32  // Amount of block bodies to be fetched per retrieval request
	MaxReceiptFetch          = 128 // Amount of transaction receipts to allow fetching per request
	MaxCodeFetch             = 64  // Amount of contract codes to allow fetching per request
	MaxProofsFetch           = 64  // Amount of merkle proofs to be fetched per retrieval request
	MaxHelperTrieProofsFetch = 64  // Amount of helper tries to be fetched per retrieval request
	MaxTxSend                = 64  // Amount of transactions to be send per request
	MaxTxStatus              = 256 // Amount of transactions to queried per request
)

type RequestInfo struct {
	Name     string
	MaxCount uint64
}

var LesRequests = map[uint64]RequestInfo{
	GetBlockHeadersMsg:     {"GetBlockHeaders", MaxHeaderFetch},
	GetBlockBodiesMsg:      {"GetBlockBodies", MaxBodyFetch},
	GetReceiptsMsg:         {"GetReceipts", MaxReceiptFetch},
	GetCodeMsg:             {"GetCode", MaxCodeFetch},
	GetProofsV2Msg:         {"GetProofsV2", MaxProofsFetch},
	GetHelperTrieProofsMsg: {"GetHelperTrieProofs", MaxHelperTrieProofsFetch},
	SendTxV2Msg:            {"SendTxV2", MaxTxSend},
	GetTxStatusMsg:         {"GetTxStatus", MaxTxStatus},
}

type ErrCode int

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrProtocolVersionMismatch
	ErrNetworkIdMismatch
	ErrGenesisBlockMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSuspendedPeer
	ErrUselessPeer
	ErrRequestRejected
	ErrUnexpectedResponse
	ErrInvalidResponse
	ErrTooManyTimeouts
	ErrMissingKey
)

func (e ErrCode) String() string {
	return errorToString[int(e)]
}

// XXX change once legacy code is out
var errorToString = map[int]string{
	ErrMsgTooLarge:             "Message too long",
	ErrDecode:                  "Invalid message",
	ErrInvalidMsgCode:          "Invalid message code",
	ErrProtocolVersionMismatch: "Protocol version mismatch",
	ErrNetworkIdMismatch:       "NetworkId mismatch",
	ErrGenesisBlockMismatch:    "Genesis block mismatch",
	ErrNoStatusMsg:             "No status message",
	ErrExtraStatusMsg:          "Extra status message",
	ErrSuspendedPeer:           "Suspended peer",
	ErrRequestRejected:         "Request rejected",
	ErrUnexpectedResponse:      "Unexpected response",
	ErrInvalidResponse:         "Invalid response",
	ErrTooManyTimeouts:         "Too many request timeouts",
	ErrMissingKey:              "Key missing from list",
}

// HeadHeader is the a part of announcement sent by the LES server to the
// LES client when a new block is generated in the network.
//
// HeadHeader can also be used to represent the head info of peer(both server
// and client).
type HeadHeader struct {
	Hash   common.Hash // Hash of one particular block being announced
	Number uint64      // Number of one particular block being announced
	Td     *big.Int    // Total difficulty of one particular block being announced
}

// Announcement is a network packet sent by the LES server to the LES client
// when a new block is generated in the network or the server has protocol
// parameters that need to be updated.
type Announcement struct {
	HeadHeader              // The data of new arrival header
	ReorgDepth uint64       // The reorg depth of new arrival header
	Update     KeyValueList // Updated protocol parameters
}

// SanityCheck verifies that the values are reasonable, as a DoS protection
func (a *Announcement) SanityCheck() error {
	if tdlen := a.Td.BitLen(); tdlen > 100 {
		return fmt.Errorf("too large block TD: bitlen %d", tdlen)
	}
	return nil
}

// Sign adds a signature to the block announcement by the given privKey
func (a *Announcement) Sign(privKey *ecdsa.PrivateKey) {
	rlp, _ := rlp.EncodeToBytes(HeadHeader{a.Hash, a.Number, a.Td})
	sig, _ := crypto.Sign(crypto.Keccak256(rlp), privKey)
	a.Update = a.Update.Add("sign", sig)
}

// CheckSignature verifies if the block announcement has a valid signature
// by the given pubKey.
func (a *Announcement) CheckSignature(id enode.ID, update KeyValueMap) error {
	var sig []byte
	if err := update.Get("sign", &sig); err != nil {
		return err
	}
	rlp, _ := rlp.EncodeToBytes(HeadHeader{a.Hash, a.Number, a.Td})
	recPubkey, err := crypto.SigToPub(crypto.Keccak256(rlp), sig)
	if err != nil {
		return err
	}
	if id == enode.PubkeyToIDV4(recPubkey) {
		return nil
	}
	return errors.New("wrong signature")
}

// GetBlockHeadersRequest represents a block header query sent by les client.
type GetBlockHeadersRequest struct {
	Origin  HashOrNumber // Block from which to retrieve headers
	Amount  uint64       // Maximum number of headers to retrieve
	Skip    uint64       // Blocks to skip between consecutive headers
	Reverse bool         // Query direction (false = rising towards latest, true = falling towards genesis)
}

// HashOrNumber is a combined field for specifying an origin block.
type HashOrNumber struct {
	Hash   common.Hash // Block hash from which to retrieve headers (excludes Number)
	Number uint64      // Block hash from which to retrieve headers (excludes Hash)
}

// EncodeRLP is a specialized encoder for HashOrNumber to encode only one of the
// two contained union fields.
func (hn *HashOrNumber) EncodeRLP(w io.Writer) error {
	if hn.Hash == (common.Hash{}) {
		return rlp.Encode(w, hn.Number)
	}
	if hn.Number != 0 {
		return fmt.Errorf("both origin hash (%x) and number (%d) provided", hn.Hash, hn.Number)
	}
	return rlp.Encode(w, hn.Hash)
}

// DecodeRLP is a specialized decoder for HashOrNumber to decode the contents
// into either a block hash or a block number.
func (hn *HashOrNumber) DecodeRLP(s *rlp.Stream) error {
	_, size, _ := s.Kind()
	origin, err := s.Raw()
	if err == nil {
		switch {
		case size == 32:
			err = rlp.DecodeBytes(origin, &hn.Hash)
		case size <= 8:
			err = rlp.DecodeBytes(origin, &hn.Number)
		default:
			err = fmt.Errorf("invalid input size %d for origin", size)
		}
	}
	return err
}

// TrieProofRequest represents a state/storage trie proof query
// sent by les client.
type TrieProofRequest struct {
	BlockHash common.Hash // The corresponding block hash of state
	Account   []byte      // The address of target account, nil if it's a global state trie proof request
	Key       []byte      // The key of target storage slot or account
	FromLevel uint        // The node level beyond which all trie nodes are contained in the proof
}

// CodeRequest represents a contract code query sent by les client.
type CodeRequest struct {
	BlockHash common.Hash // The corresponding block hash of state
	Account   []byte      // The address of target account
}

const (
	// HelperTrieCHT is the indicator of canonical hash trie, check
	// https://github.com/ethereum/devp2p/blob/master/caps/les.md#canonical-hash-trie
	// for more details.
	HelperTrieCHT = iota

	// HelperTrieBloomTrie is the indicator of bloom trie, check
	// https://github.com/ethereum/devp2p/blob/master/caps/les.md#bloombits-trie
	// for more details
	HelperTrieBloomTrie

	// The auxiliary data type of helperTrie request which is available for
	// all helperTrie request.
	AuxRoot = 1

	// The auxiliary data type of CHT request - corresponding block header
	// which is only avaiable for CHT request.
	AuxHeader = 2
)

// HelperTrieRequest represents a helper trie query sent by les client.
// HelperTrie includes: CHT and bloom trie. It's a shared structure between
// these two kinds of request.
//
// Except the helperTrie proof of requested entry will be returned, caller
// can specify more additional auxiliary data to be returned via `AuxType`.
type HelperTrieRequest struct {
	Type      uint   // Indicator of request type, 0 represents CHT, 1 represents Bloom trie
	TrieIndex uint64 // The index(section index) of requested trie
	Key       []byte // The list of entry keys, caller can request a batch of entries in a single request.
	FromLevel uint   // The node level beyond which all trie nodes are contained in the proof
	AuxType   uint   // The type of auxiliary data requested
}

// HelperTrieResponse represents the response of corresponding helperTrie
// request. A single response contains a batch of requested proofs and
// corresponding auxiliary data.
type HelperTrieResponse struct {
	Proofs  light.NodeList // The container for storing all requested proofs
	AuxData [][]byte       // The batch of requested auxiliary data
}

// ErrResp returns an protocol error with given error code and additional
// error message.
func ErrResp(code ErrCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

// LesTopic constructs the discovery v5 topic for LES protocol.
func LesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case Lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type (
	// RequestCost represents a cost policy of a specified request type.
	RequestCost struct {
		BaseCost, ReqCost uint64
	}
	// RequestCostTable assigns a cost estimate function to each request type
	// which is a linear function of the requested amount
	// (cost = BaseCost + ReqCost * amount)
	RequestCostTable map[uint64]*RequestCost
	// RequestCostList is a list representation of request costs which is used for
	// database storage and communication through the network
	RequestCostList     []RequestCostListItem
	RequestCostListItem struct {
		MsgCode, BaseCost, ReqCost uint64
	}
)

// GetMaxCost calculates the estimated cost for a given request type and amount
func (table RequestCostTable) GetMaxCost(code, amount uint64) uint64 {
	costs := table[code]
	return costs.BaseCost + amount*costs.ReqCost
}

// ToTable converts a cost list to a cost table
func (list RequestCostList) ToTable(protocolLength uint64) RequestCostTable {
	table := make(RequestCostTable)
	for _, e := range list {
		if e.MsgCode < protocolLength {
			table[e.MsgCode] = &RequestCost{
				BaseCost: e.BaseCost,
				ReqCost:  e.ReqCost,
			}
		}
	}
	return table
}
