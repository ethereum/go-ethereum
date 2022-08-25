package types

import "github.com/XinFinOrg/XDPoSChain/common"

type ForensicsInfo struct {
	HashPath        []string   `json:"hashPath"`
	QuorumCert      QuorumCert `json:"quorumCert"`
	SignerAddresses []string   `json:"signerAddresses"`
}

type ForensicsContent struct {
	DivergingBlockNumber uint64         `json:"divergingBlockNumber"`
	DivergingBlockHash   string         `json:"divergingBlockHash"`
	AcrossEpoch          bool           `json:"acrossEpoch"`
	SmallerRoundInfo     *ForensicsInfo `json:"smallerRoundInfo"`
	LargerRoundInfo      *ForensicsInfo `json:"largerRoundInfo"`
}

type VoteEquivocationContent struct {
	SmallerRoundVote *Vote          `json:"smallerRoundVote"`
	LargerRoundVote  *Vote          `json:"largerRoundVote"`
	Signer           common.Address `json:"signer"`
}

type ForensicProof struct {
	Id            string `json:"id"`
	ForensicsType string `json:"forensicsType"` // QC or VOTE
	Content       string `json:"content"`       // Json string of the forensics data
}

type ForensicsEvent struct {
	ForensicsProof *ForensicProof
}
