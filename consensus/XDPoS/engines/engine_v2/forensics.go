package engine_v2

import (
	"fmt"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/log"
)

const (
	NUM_OF_FORENSICS_PARENTS = 2
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

// Forensics instance. Placeholder for future properties to be added
type Forensics struct {
	HighestCommittedQCs []utils.QuorumCert
}

// Initiate a forensics process
func NewForensics() *Forensics {
	return &Forensics{}
}

/*
	Entry point for processing forensics.
	Triggered once processQC is successfully.
	Forensics runs in a seperate go routine as its no system critical
	Link to the flow diagram: https://hashlabs.atlassian.net/wiki/spaces/HASHLABS/pages/97878029/Forensics+Diagram+flow
*/
func (f *Forensics) ProcessForensics(chain consensus.ChainReader, incomingQC utils.QuorumCert) {
	log.Info("Received a QC in forensics", "QC", incomingQC)
}

// Set the forensics committed QCs list. The order is from grandparent to current header. i.e it shall follow the QC in its header as follow [hcqc1, hcqc2, hcqc3]
func (f *Forensics) SetCommittedQCs(headers []types.Header, incomingQC utils.QuorumCert) error {
	// highestCommitQCs is an array, assign the parentBlockQc and its child as well as its grandchild QC into this array for forensics purposes.
	if len(headers) != NUM_OF_FORENSICS_PARENTS {
		log.Error("[SetCommittedQcs] Received input length not equal to 2", len(headers))
		return fmt.Errorf("Received headers length not equal to 2 ")
	}

	var committedQCs []utils.QuorumCert
	for i, h := range headers {
		var decodedExtraField utils.ExtraFields_v2
		// Decode the qc1 and qc2
		err := utils.DecodeBytesExtraFields(h.Extra, &decodedExtraField)
		if err != nil {
			log.Error("[SetCommittedQCs] Fail to decode extra when committing QC to forensics", "Error", err, "Index", i)
			return err
		}
		if i != 0 {
			if decodedExtraField.QuorumCert.ProposedBlockInfo.Hash != headers[i-1].Hash() {
				log.Error("[SetCommittedQCs] Headers shall be on the same chain and in the right order", "ParentHash", h.ParentHash.Hex(), "headers[i-1].Hash()", headers[i-1].Hash().Hex())
				return fmt.Errorf("Headers shall be on the same chain and in the right order")
			} else if i == len(headers)-1 { // The last header shall be pointed by the incoming QC
				if incomingQC.ProposedBlockInfo.Hash != h.Hash() {
					log.Error("[SetCommittedQCs] incomingQc is not pointing at the last header received", "hash", h.Hash().Hex(), "incomingQC.ProposedBlockInfo.Hash", incomingQC.ProposedBlockInfo.Hash.Hex())
					return fmt.Errorf("incomingQc is not pointing at the last header received")
				}
			}
		}

		committedQCs = append(committedQCs, *decodedExtraField.QuorumCert)
	}
	f.HighestCommittedQCs = append(committedQCs, incomingQC)
	return nil
}

// Last step of forensics which sends out detailed proof to report service.
func (f *Forensics) SendForensicProof() {
}

// Find the blockInfo of the block -2 distance away from the QC. Note: We using block number which means not necessary on the same chain as QC received
func (f *Forensics) findParentsQc(chain consensus.ChainReader, currentQc utils.QuorumCert, distanceFromCurrrentQc int64) {
}

func (f *Forensics) findCommonSigners(currentQc utils.QuorumCert, higherQc utils.QuorumCert) {
}
