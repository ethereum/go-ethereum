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

func (s Signature) DeepCopy() interface{} {
	cpy := make([]byte, len(s))
	copy(cpy, s)
	return s
}

// Block Info struct in XDPoS 2.0, used for vote message, etc.
type BlockInfo struct {
	Hash   common.Hash `json:"hash"`
	Round  Round       `json:"round"`
	Number *big.Int    `json:"number"`
}

// Vote message in XDPoS 2.0
type Vote struct {
	signer            common.Address //field not exported
	ProposedBlockInfo *BlockInfo     `json:"proposedBlockInfo"`
	Signature         Signature      `json:"signature"`
	GapNumber         uint64         `json:"gapNumber"`
}

func (v *Vote) DeepCopy() interface{} {
	proposedBlockInfoCopy := &BlockInfo{
		Hash:   v.ProposedBlockInfo.Hash,
		Round:  v.ProposedBlockInfo.Round,
		Number: new(big.Int).Set(v.ProposedBlockInfo.Number),
	}
	return &Vote{
		signer:            v.signer,
		ProposedBlockInfo: proposedBlockInfoCopy,
		Signature:         v.Signature.DeepCopy().(Signature),
		GapNumber:         v.GapNumber,
	}
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

func (t *Timeout) DeepCopy() interface{} {
	return &Timeout{
		signer:    t.signer,
		Round:     t.Round,
		Signature: t.Signature.DeepCopy().(Signature),
		GapNumber: t.GapNumber,
	}
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

func (s *SyncInfo) DeepCopy() interface{} {
	var highestQCCopy *QuorumCert
	if s.HighestQuorumCert != nil {
		sigsCopy := make([]Signature, len(s.HighestQuorumCert.Signatures))
		for i, sig := range s.HighestQuorumCert.Signatures {
			sigsCopy[i] = sig.DeepCopy().(Signature)
		}
		highestQCCopy = &QuorumCert{
			ProposedBlockInfo: &BlockInfo{
				Hash:   s.HighestQuorumCert.ProposedBlockInfo.Hash,
				Round:  s.HighestQuorumCert.ProposedBlockInfo.Round,
				Number: new(big.Int).Set(s.HighestQuorumCert.ProposedBlockInfo.Number),
			},
			Signatures: sigsCopy,
			GapNumber:  s.HighestQuorumCert.GapNumber,
		}
	}

	var highestTimeoutCopy *TimeoutCert
	if s.HighestTimeoutCert != nil {
		sigsCopy := make([]Signature, len(s.HighestTimeoutCert.Signatures))
		for i, sig := range s.HighestTimeoutCert.Signatures {
			sigsCopy[i] = sig.DeepCopy().(Signature)
		}
		highestTimeoutCopy = &TimeoutCert{
			Round:      s.HighestTimeoutCert.Round,
			Signatures: sigsCopy,
			GapNumber:  s.HighestTimeoutCert.GapNumber,
		}
	}
	return &SyncInfo{
		HighestQuorumCert:  highestQCCopy,
		HighestTimeoutCert: highestTimeoutCopy,
	}
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
