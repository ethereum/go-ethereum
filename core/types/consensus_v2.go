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
	signer            common.Address
	ProposedBlockInfo *BlockInfo
	Signature         Signature
	GapNumber         uint64
}

func (v *Vote) Hash() common.Hash {
	return rlpHash(v)
}

func (v *Vote) PoolKey() string {
	// return the voted block hash
	return fmt.Sprint(v.ProposedBlockInfo.Round, ":", v.GapNumber, ":", v.ProposedBlockInfo.Number, ":", v.ProposedBlockInfo.Hash.Hex())
}

func (v *Vote) GetSigner() common.Address {
	return v.signer
}

func (v *Vote) SetSigner(signer common.Address) {
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
	Masternodes                []common.Address
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
