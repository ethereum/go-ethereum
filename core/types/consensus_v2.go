package types

import (
	"fmt"
	"math/big"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/rlp"
)

// Round number type in XDPoS 2.0
type Round uint64
type Signature []byte

// Block Info struct in XDPoS 2.0, used for vote message, etc.
type BlockInfo struct {
	Hash   common.Hash
	Round  Round
	Number *big.Int
}

// Vote message in XDPoS 2.0
type Vote struct {
	ProposedBlockInfo *BlockInfo
	Signature         Signature
	GapNumber         uint64
}

// Timeout message in XDPoS 2.0
type Timeout struct {
	Round     Round
	Signature Signature
	GapNumber uint64
}

// BFT Sync Info message in XDPoS 2.0
type SyncInfo struct {
	HighestQuorumCert  *QuorumCert
	HighestTimeoutCert *TimeoutCert
}

// Quorum Certificate struct in XDPoS 2.0
type QuorumCert struct {
	ProposedBlockInfo *BlockInfo
	Signatures        []Signature
	GapNumber         uint64
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

type EpochSwitchInfo struct {
	Masternodes                []common.Address
	EpochSwitchBlockInfo       *BlockInfo
	EpochSwitchParentBlockInfo *BlockInfo
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

func (m *Vote) Hash() common.Hash {
	return rlpHash(m)
}

func (m *Timeout) Hash() common.Hash {
	return rlpHash(m)
}

func (m *SyncInfo) Hash() common.Hash {
	return rlpHash(m)
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

func (m *Vote) PoolKey() string {
	// return the voted block hash
	return fmt.Sprint(m.ProposedBlockInfo.Round, ":", m.GapNumber, ":", m.ProposedBlockInfo.Number, ":", m.ProposedBlockInfo.Hash.Hex())
}

func (m *Timeout) PoolKey() string {
	// timeout pool key is round:gapNumber
	return fmt.Sprint(m.Round, ":", m.GapNumber)
}
