package engine_v2

import (
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/log"
)

type ForensicProof struct {
	QcWithSmallerRound          utils.QuorumCert
	QcWithLargerRound           utils.QuorumCert
	DivergingHash               common.Hash
	HashesTillSmallerRoundQc    []common.Hash
	HashesTillLargerRoundQc     []common.Hash
	AcrossEpochs                bool
	QcWithSmallerRoundAddresses []common.Address
	QcWithLargerRoundAddresses  []common.Address
}

type Forensics struct {
	ReceiverCh <-chan utils.QuorumCert
	Abort      chan<- struct{}
}

// Initiate a forensics process
func (x *XDPoS_v2) AttachForensics() {
	receiver := make(chan utils.QuorumCert)
	abort := make(chan struct{})

	go func() {
		for {
			// A real event arrived, process interesting content
			select {
			case quorumCert := <-receiver:
				x.ProcessForensics(quorumCert)
			case <-abort:
				return
			}
		}
	}()
	x.forensics = &Forensics{
		ReceiverCh: receiver,
		Abort:      abort,
	}
}

func (x *XDPoS_v2) SendForensicProof() {
}

func (x *XDPoS_v2) ProcessForensics(quorumCert utils.QuorumCert) {
	log.Info("Received a QC in forensics", "QC", quorumCert)
}
