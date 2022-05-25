package types

import "github.com/XinFinOrg/XDPoSChain/common"

type ForensicsInfo struct {
	HashPath        []string // HashesTillSmallerRoundQc or HashesTillLargerRoundQc
	QuorumCert      QuorumCert
	SignerAddresses []string
}

type ForensicProof struct {
	SmallerRoundInfo *ForensicsInfo
	LargerRoundInfo  *ForensicsInfo
	DivergingHash    common.Hash
	AcrossEpochs     bool
}

type ForensicsEvent struct {
	ForensicsProof *ForensicProof
}
