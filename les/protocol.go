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

package les

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"io"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	lpc "github.com/ethereum/go-ethereum/les/lespay/client"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rlp"
)

// Constants to match up protocol versions and messages
const (
	lpv2 = 2
	lpv3 = 3
	lpv4 = 4
)

// Supported versions of the les protocol (first is primary)
var (
	ClientProtocolVersions    = []uint{lpv2, lpv3}
	ServerProtocolVersions    = []uint{lpv2, lpv3}
	AdvertiseProtocolVersions = []uint{lpv2} // clients are searching for the first advertised protocol in the list
)

// Number of implemented message corresponding to different protocol versions.
var ProtocolLengths = map[uint]uint64{lpv2: 22, lpv3: 24, lpv4: 24}

const (
	NetworkId          = 1
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

type requestInfo struct {
	name                          string
	maxCount                      uint64
	refBasketFirst, refBasketRest float64
}

// reqMapping maps an LES request to one or two lespay service vector entries.
// If rest != -1 and the request type is used with amounts larger than one then the
// first one of the multi-request is mapped to first while the rest is mapped to rest.
type reqMapping struct {
	first, rest int
}

var (
	// requests describes the available LES request types and their initializing amounts
	// in the lespay/client.ValueTracker reference basket. Initial values are estimates
	// based on the same values as the server's default cost estimates (reqAvgTimeCost).
	requests = map[uint64]requestInfo{
		GetBlockHeadersMsg:     {"GetBlockHeaders", MaxHeaderFetch, 10, 1000},
		GetBlockBodiesMsg:      {"GetBlockBodies", MaxBodyFetch, 1, 0},
		GetReceiptsMsg:         {"GetReceipts", MaxReceiptFetch, 1, 0},
		GetCodeMsg:             {"GetCode", MaxCodeFetch, 1, 0},
		GetProofsV2Msg:         {"GetProofsV2", MaxProofsFetch, 10, 0},
		GetHelperTrieProofsMsg: {"GetHelperTrieProofs", MaxHelperTrieProofsFetch, 10, 100},
		SendTxV2Msg:            {"SendTxV2", MaxTxSend, 1, 0},
		GetTxStatusMsg:         {"GetTxStatus", MaxTxStatus, 10, 0},
	}
	requestList    []lpc.RequestInfo
	requestMapping map[uint32]reqMapping
)

// init creates a request list and mapping between protocol message codes and lespay
// service vector indices.
func init() {
	requestMapping = make(map[uint32]reqMapping)
	for code, req := range requests {
		cost := reqAvgTimeCost[code]
		rm := reqMapping{len(requestList), -1}
		requestList = append(requestList, lpc.RequestInfo{
			Name:       req.name + ".first",
			InitAmount: req.refBasketFirst,
			InitValue:  float64(cost.baseCost + cost.reqCost),
		})
		if req.refBasketRest != 0 {
			rm.rest = len(requestList)
			requestList = append(requestList, lpc.RequestInfo{
				Name:       req.name + ".rest",
				InitAmount: req.refBasketRest,
				InitValue:  float64(cost.reqCost),
			})
		}
		requestMapping[uint32(code)] = rm
	}
}

type errCode int

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
	ErrForkIDRejected
)

func (e errCode) String() string {
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
	ErrForkIDRejected:          "ForkID rejected",
}

// announceData is the network packet for the block announcements.
type announceData struct {
	Hash       common.Hash // Hash of one particular block being announced
	Number     uint64      // Number of one particular block being announced
	Td         *big.Int    // Total difficulty of one particular block being announced
	ReorgDepth uint64
	Update     keyValueList
}

// sanityCheck verifies that the values are reasonable, as a DoS protection
func (a *announceData) sanityCheck() error {
	if tdlen := a.Td.BitLen(); tdlen > 100 {
		return fmt.Errorf("too large block TD: bitlen %d", tdlen)
	}
	return nil
}

// sign adds a signature to the block announcement by the given privKey
func (a *announceData) sign(privKey *ecdsa.PrivateKey) {
	rlp, _ := rlp.EncodeToBytes(blockInfo{a.Hash, a.Number, a.Td})
	sig, _ := crypto.Sign(crypto.Keccak256(rlp), privKey)
	a.Update = a.Update.add("sign", sig)
}

// checkSignature verifies if the block announcement has a valid signature by the given pubKey
func (a *announceData) checkSignature(id enode.ID, update keyValueMap) error {
	var sig []byte
	if err := update.get("sign", &sig); err != nil {
		return err
	}
	rlp, _ := rlp.EncodeToBytes(blockInfo{a.Hash, a.Number, a.Td})
	recPubkey, err := crypto.SigToPub(crypto.Keccak256(rlp), sig)
	if err != nil {
		return err
	}
	if id == enode.PubkeyToIDV4(recPubkey) {
		return nil
	}
	return errors.New("wrong signature")
}

type blockInfo struct {
	Hash   common.Hash // Hash of one particular block being announced
	Number uint64      // Number of one particular block being announced
	Td     *big.Int    // Total difficulty of one particular block being announced
}

// getBlockHeadersData represents a block header query.
type getBlockHeadersData struct {
	Origin  hashOrNumber // Block from which to retrieve headers
	Amount  uint64       // Maximum number of headers to retrieve
	Skip    uint64       // Blocks to skip between consecutive headers
	Reverse bool         // Query direction (false = rising towards latest, true = falling towards genesis)
}

// hashOrNumber is a combined field for specifying an origin block.
type hashOrNumber struct {
	Hash   common.Hash // Block hash from which to retrieve headers (excludes Number)
	Number uint64      // Block hash from which to retrieve headers (excludes Hash)
}

// EncodeRLP is a specialized encoder for hashOrNumber to encode only one of the
// two contained union fields.
func (hn *hashOrNumber) EncodeRLP(w io.Writer) error {
	if hn.Hash == (common.Hash{}) {
		return rlp.Encode(w, hn.Number)
	}
	if hn.Number != 0 {
		return fmt.Errorf("both origin hash (%x) and number (%d) provided", hn.Hash, hn.Number)
	}
	return rlp.Encode(w, hn.Hash)
}

// DecodeRLP is a specialized decoder for hashOrNumber to decode the contents
// into either a block hash or a block number.
func (hn *hashOrNumber) DecodeRLP(s *rlp.Stream) error {
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

// CodeData is the network response packet for a node data retrieval.
type CodeData []struct {
	Value []byte
}
