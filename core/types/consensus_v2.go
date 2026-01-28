// Copyright (c) 2018 XDPoSChain
// XDPoS 2.0 consensus types

package types

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// Round number type in XDPoS 2.0
type Round uint64
type Signature []byte

// Block Info struct in XDPoS 2.0, used for vote message, etc.
type BlockInfo struct {
	Hash   common.Hash `json:"hash"`
	Round  Round       `json:"round"`
	Number *big.Int    `json:"number"`
}

// Vote message in XDPoS 2.0
type VoteXDPoS struct {
	signer            common.Address //field not exported
	ProposedBlockInfo *BlockInfo     `json:"proposedBlockInfo"`
	Signature         Signature      `json:"signature"`
	GapNumber         uint64         `json:"gapNumber"`
}

func (v *VoteXDPoS) Hash() common.Hash {
	return rlpHash(v)
}

func (v *VoteXDPoS) PoolKey() string {
	// return the voted block hash
	return fmt.Sprint(v.ProposedBlockInfo.Round, ":", v.GapNumber, ":", v.ProposedBlockInfo.Number, ":", v.ProposedBlockInfo.Hash.Hex())
}

func (v *VoteXDPoS) GetSigner() common.Address {
	return v.signer
}

func (v *VoteXDPoS) SetSigner(signer common.Address) {
	v.signer = signer
}

// Timeout message in XDPoS 2.0
type Timeout struct {
	signer    common.Address
	Round     Round
	Signature Signature
	GapNumber uint64
}

func (t *Timeout) Hash() common.Hash {
	return rlpHash(t)
}

func (t *Timeout) PoolKey() string {
	// timeout pool key is round:gapNumber
	return fmt.Sprint(t.Round, ":", t.GapNumber)
}

func (t *Timeout) GetSigner() common.Address {
	return t.signer
}

func (t *Timeout) SetSigner(signer common.Address) {
	t.signer = signer
}

// BFT Sync Info message in XDPoS 2.0
type SyncInfo struct {
	HighestQuorumCert  *QuorumCert
	HighestTimeoutCert *TimeoutCert
}

func (s *SyncInfo) Hash() common.Hash {
	return rlpHash(s)
}

// Quorum Certificate struct in XDPoS 2.0
type QuorumCert struct {
	ProposedBlockInfo *BlockInfo  `json:"proposedBlockInfo"`
	Signatures        []Signature `json:"signatures"`
	GapNumber         uint64      `json:"gapNumber"`
}

// Timeout Certificate struct in XDPoS 2.0
type TimeoutCert struct {
	Round      Round
	Signatures []Signature
	GapNumber  uint64
}

// The parsed extra fields in block header in XDPoS 2.0 (excluding the version byte)
// The version byte (consensus version) is the first byte in header's extra and it's only valid with value >= 2
type ExtraFields_v2 struct {
	Round      Round
	QuorumCert *QuorumCert
}

// Encode XDPoS 2.0 extra fields into bytes
func (e *ExtraFields_v2) EncodeToBytes() ([]byte, error) {
	bytes, err := rlp.EncodeToBytes(e)
	if err != nil {
		return nil, err
	}
	versionByte := []byte{2}
	return append(versionByte, bytes...), nil
}

type EpochSwitchInfo struct {
	Penalties                  []common.Address
	Standbynodes               []common.Address
	Masternodes                []common.Address
	MasternodesLen             int
	EpochSwitchBlockInfo       *BlockInfo
	EpochSwitchParentBlockInfo *BlockInfo
}

type VoteForSign struct {
	ProposedBlockInfo *BlockInfo
	GapNumber         uint64
}

func VoteSigHash(m *VoteForSign) common.Hash {
	return rlpHash(m)
}

type TimeoutForSign struct {
	Round     Round
	GapNumber uint64
}

func TimeoutSigHash(m *TimeoutForSign) common.Hash {
	return rlpHash(m)
}
