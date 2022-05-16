package engine_v2

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/log"
)

const (
	NUM_OF_FORENSICS_QC = 3
)

type ForensicsInfo struct {
	HashPath        []string // HashesTillSmallerRoundQc or HashesTillLargerRoundQc
	QuorumCert      utils.QuorumCert
	SignerAddresses []string
}

type ForensicProof struct {
	SmallerRoundInfo *ForensicsInfo
	LargerRoundInfo  *ForensicsInfo
	DivergingHash    common.Hash
	AcrossEpochs     bool
}

// Forensics instance. Placeholder for future properties to be added
type Forensics struct {
	HighestCommittedQCs []utils.QuorumCert
}

// Initiate a forensics process
func NewForensics() *Forensics {
	return &Forensics{}
}

func (f *Forensics) ForensicsMonitoring(chain consensus.ChainReader, engine *XDPoS_v2, headerQcToBeCommitted []types.Header, incomingQC utils.QuorumCert) error {
	f.ProcessForensics(chain, engine, incomingQC)
	return f.SetCommittedQCs(headerQcToBeCommitted, incomingQC)
}

// Set the forensics committed QCs list. The order is from grandparent to current header. i.e it shall follow the QC in its header as follow [hcqc1, hcqc2, hcqc3]
func (f *Forensics) SetCommittedQCs(headers []types.Header, incomingQC utils.QuorumCert) error {
	// highestCommitQCs is an array, assign the parentBlockQc and its child as well as its grandchild QC into this array for forensics purposes.
	if len(headers) != NUM_OF_FORENSICS_QC-1 {
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

/*
	Entry point for processing forensics.
	Triggered once processQC is successfully.
	Forensics runs in a seperate go routine as its no system critical
	Link to the flow diagram: https://hashlabs.atlassian.net/wiki/spaces/HASHLABS/pages/97878029/Forensics+Diagram+flow
*/
func (f *Forensics) ProcessForensics(chain consensus.ChainReader, engine *XDPoS_v2, incomingQC utils.QuorumCert) error {
	log.Debug("Received a QC in forensics", "QC", incomingQC)
	// Clone the values to a temporary variable
	highestCommittedQCs := f.HighestCommittedQCs
	if len(highestCommittedQCs) != NUM_OF_FORENSICS_QC {
		log.Error("[ProcessForensics] HighestCommittedQCs value not set", "incomingQcProposedBlockHash", incomingQC.ProposedBlockInfo.Hash, "incomingQcProposedBlockNumber", incomingQC.ProposedBlockInfo.Number.Uint64(), "incomingQcProposedBlockRound", incomingQC.ProposedBlockInfo.Round)
		return fmt.Errorf("HighestCommittedQCs value not set")
	}
	// Find the QC1 and QC2. We only care 2 parents in front of the incomingQC. The returned value contains QC1, QC2 and QC3(the incomingQC)
	incomingQuorunCerts, err := f.findAncestorQCs(chain, incomingQC, 2)
	if err != nil {
		return err
	}
	isOnTheChain, err := f.checkQCsOnTheSameChain(chain, highestCommittedQCs, incomingQuorunCerts)
	if err != nil {
		return err
	}
	if isOnTheChain {
		// Passed the checking, nothing suspecious.
		log.Debug("[ProcessForensics] Passed forensics checking, nothing suspecious need to be reported", "incomingQcProposedBlockHash", incomingQC.ProposedBlockInfo.Hash, "incomingQcProposedBlockNumber", incomingQC.ProposedBlockInfo.Number.Uint64(), "incomingQcProposedBlockRound", incomingQC.ProposedBlockInfo.Round)
		return nil
	}
	// Trigger the safety Alarm if failed
	// First, find the QC in the two sets that have the same round
	foundSameRoundQC, sameRoundHCQC, sameRoundQC := f.findQCsInSameRound(highestCommittedQCs, incomingQuorunCerts)

	if foundSameRoundQC {
		f.SendForensicProof(chain, engine, sameRoundHCQC, sameRoundQC)
	} else {
		// Not found, need a more complex approach to find the two QC
		ancestorQC, lowerRoundQCs, _, err := f.findAncestorQcThroughRound(chain, highestCommittedQCs, incomingQuorunCerts)
		if err != nil {
			log.Error("[ProcessForensics] Error while trying to find ancestor QC through round number", "Error", err)
		}
		f.SendForensicProof(chain, engine, ancestorQC, lowerRoundQCs[NUM_OF_FORENSICS_QC-1])
	}

	return nil
}

// Last step of forensics which sends out detailed proof to report service.
func (f *Forensics) SendForensicProof(chain consensus.ChainReader, engine *XDPoS_v2, firstQc utils.QuorumCert, secondQc utils.QuorumCert) error {
	// Re-order the QC by its round number to make the function cleaner.
	lowerRoundQC := firstQc
	higherRoundQC := secondQc

	if (secondQc.ProposedBlockInfo.Round - firstQc.ProposedBlockInfo.Round) < 0 {
		lowerRoundQC = secondQc
		higherRoundQC = firstQc
	}

	// Find common ancestor block
	ancestorHash, ancestorToLowerRoundPath, ancestorToHigherRoundPath, err := f.FindAncestorBlockHash(chain, lowerRoundQC.ProposedBlockInfo, higherRoundQC.ProposedBlockInfo)
	if err != nil {
		log.Error("[SendForensicProof] Error while trying to find ancestor block hash", err)
		return err
	}

	// Check if two QCs are across epoch, this is used as a indicator for the "prone to attack" scenario
	lowerRoundQcEpochSwitchInfo, err := engine.getEpochSwitchInfo(chain, nil, lowerRoundQC.ProposedBlockInfo.Hash)
	higherRoundQcEpochSwitchInfo, err := engine.getEpochSwitchInfo(chain, nil, higherRoundQC.ProposedBlockInfo.Hash)
	accrossEpoches := false
	if lowerRoundQcEpochSwitchInfo.EpochSwitchBlockInfo.Hash != higherRoundQcEpochSwitchInfo.EpochSwitchBlockInfo.Hash {
		accrossEpoches = true
	}

	forensicsProof := &ForensicProof{
		DivergingHash: ancestorHash,
		AcrossEpochs:  accrossEpoches,
		SmallerRoundInfo: &ForensicsInfo{
			HashPath:        ancestorToLowerRoundPath,
			QuorumCert:      lowerRoundQC,
			SignerAddresses: f.getQcSignerAddresses(lowerRoundQC),
		},
		LargerRoundInfo: &ForensicsInfo{
			HashPath:        ancestorToHigherRoundPath,
			QuorumCert:      higherRoundQC,
			SignerAddresses: f.getQcSignerAddresses(higherRoundQC),
		},
	}
	// TODO: send to dedicated channel which will redirect to stats server
	log.Info("Forensics proof report generated, sending to the stats server", forensicsProof)
	return nil
}

// Utils function to help find the n-th previous QC. It returns an array of QC in ascending order including the currentQc as the last item in the array
func (f *Forensics) findAncestorQCs(chain consensus.ChainReader, currentQc utils.QuorumCert, distanceFromCurrrentQc int) ([]utils.QuorumCert, error) {
	var quorumCerts []utils.QuorumCert
	quorumCertificate := currentQc
	// Append the initial value
	quorumCerts = append(quorumCerts, quorumCertificate)
	// Append the parents
	for i := 0; i < distanceFromCurrrentQc; i++ {
		parentHash := quorumCertificate.ProposedBlockInfo.Hash
		parentHeader := chain.GetHeaderByHash(parentHash)
		if parentHeader == nil {
			log.Error("[findAncestorQCs] Forensics findAncestorQCs unable to find its parent block header", "BlockNum", parentHeader.Number.Int64(), "ParentHash", parentHash.Hex())
			return nil, fmt.Errorf("Unable to find parent block header in forensics")
		}
		var decodedExtraField utils.ExtraFields_v2
		err := utils.DecodeBytesExtraFields(parentHeader.Extra, &decodedExtraField)
		if err != nil {
			log.Error("[findAncestorQCs] Error while trying to decode from parent block extra", "BlockNum", parentHeader.Number.Int64(), "ParentHash", parentHash.Hex())
		}
		quorumCertificate = *decodedExtraField.QuorumCert
		quorumCerts = append(quorumCerts, quorumCertificate)
	}
	// The quorumCerts is in the reverse order, we need to flip it
	var quorumCertsInAscendingOrder []utils.QuorumCert
	for i := len(quorumCerts) - 1; i >= 0; i-- {
		quorumCertsInAscendingOrder = append(quorumCertsInAscendingOrder, quorumCerts[i])
	}
	return quorumCertsInAscendingOrder, nil
}

// Check whether two provided QC set are on the same chain
func (f *Forensics) checkQCsOnTheSameChain(chain consensus.ChainReader, highestCommittedQCs []utils.QuorumCert, incomingQCandItsParents []utils.QuorumCert) (bool, error) {
	// Re-order two sets of QCs by block Number
	lowerBlockNumQCs := highestCommittedQCs
	higherBlockNumQCs := incomingQCandItsParents
	if incomingQCandItsParents[0].ProposedBlockInfo.Number.Cmp(highestCommittedQCs[0].ProposedBlockInfo.Number) == -1 {
		lowerBlockNumQCs = incomingQCandItsParents
		higherBlockNumQCs = highestCommittedQCs
	}

	proposedBlockInfo := higherBlockNumQCs[0].ProposedBlockInfo
	for i := 0; i < int((big.NewInt(0).Sub(higherBlockNumQCs[0].ProposedBlockInfo.Number, lowerBlockNumQCs[0].ProposedBlockInfo.Number)).Int64()); i++ {
		parentHeader := chain.GetHeaderByHash(proposedBlockInfo.Hash)
		var decodedExtraField utils.ExtraFields_v2
		err := utils.DecodeBytesExtraFields(parentHeader.Extra, &decodedExtraField)
		if err != nil {
			log.Error("[checkQCsOnTheSameChain] Fail to decode extra when checking the two QCs set on the same chain", "Error", err)
			return false, err
		}
		proposedBlockInfo = decodedExtraField.QuorumCert.ProposedBlockInfo
	}
	// Check the final proposed blockInfo is the same as what we have from lowerBlockNumQCs[0]
	if reflect.DeepEqual(proposedBlockInfo, lowerBlockNumQCs[0].ProposedBlockInfo) {
		return true, nil
	}

	return false, nil
}

// Given the two QCs set, find if there are any QC that have the same round
func (f *Forensics) findQCsInSameRound(quorumCerts1 []utils.QuorumCert, quorumCerts2 []utils.QuorumCert) (bool, utils.QuorumCert, utils.QuorumCert) {
	for _, quorumCert1 := range quorumCerts1 {
		for _, quorumCert2 := range quorumCerts2 {
			if quorumCert1.ProposedBlockInfo.Round == quorumCert2.ProposedBlockInfo.Round {
				return true, quorumCert1, quorumCert2
			}
		}
	}
	return false, utils.QuorumCert{}, utils.QuorumCert{}
}

// Find the signer list from QC signatures
func (f *Forensics) getQcSignerAddresses(quorumCert utils.QuorumCert) []string {
	var signerList []string

	// The QC signatures are signed by votes special struct VoteForSign
	quorumCertSignedHash := utils.VoteSigHash(&utils.VoteForSign{
		ProposedBlockInfo: quorumCert.ProposedBlockInfo,
		GapNumber:         quorumCert.GapNumber,
	})
	for _, signature := range quorumCert.Signatures {
		var signerAddress common.Address
		pubkey, err := crypto.Ecrecover(quorumCertSignedHash.Bytes(), signature)
		if err != nil {
			log.Error("[getQcSignerAddresses] Fail to Ecrecover signer from the quorumCertSignedHash", "quorumCert.GapNumber", quorumCert.GapNumber, "quorumCert.ProposedBlockInfo", quorumCert.ProposedBlockInfo)
		}

		copy(signerAddress[:], crypto.Keccak256(pubkey[1:])[12:])
		signerList = append(signerList, signerAddress.Hex())
	}
	return signerList
}

// Check whether the given QCs are on the same chain as the stored committed QCs(f.HighestCommittedQCs) regardless their orders
func (f *Forensics) findAncestorQcThroughRound(chain consensus.ChainReader, highestCommittedQCs []utils.QuorumCert, incomingQCandItsParents []utils.QuorumCert) (utils.QuorumCert, []utils.QuorumCert, []utils.QuorumCert, error) {
	/*
		Re-order two sets of QCs by Round number
	*/
	lowerRoundQCs := highestCommittedQCs
	higherRoundQCs := incomingQCandItsParents
	if incomingQCandItsParents[0].ProposedBlockInfo.Round < highestCommittedQCs[0].ProposedBlockInfo.Round {
		lowerRoundQCs = incomingQCandItsParents
		higherRoundQCs = highestCommittedQCs
	}

	// Find the ancestorFromIncomingQC1 that matches round number < lowerRoundQCs3
	ancestorQC := higherRoundQCs[0]
	for ancestorQC.ProposedBlockInfo.Round >= lowerRoundQCs[NUM_OF_FORENSICS_QC-1].ProposedBlockInfo.Round {
		proposedBlock := chain.GetHeaderByHash(ancestorQC.ProposedBlockInfo.Hash)
		var decodedExtraField utils.ExtraFields_v2
		err := utils.DecodeBytesExtraFields(proposedBlock.Extra, &decodedExtraField)
		if err != nil {
			log.Error("[findAncestorQcThroughRound] Error while trying to decode extra field", "ProposedBlockInfo.Hash", ancestorQC.ProposedBlockInfo.Hash)
			return ancestorQC, lowerRoundQCs, higherRoundQCs, err
		}
		// Found the ancestor QC
		if decodedExtraField.QuorumCert.ProposedBlockInfo.Round < lowerRoundQCs[NUM_OF_FORENSICS_QC-1].ProposedBlockInfo.Round {
			return ancestorQC, lowerRoundQCs, higherRoundQCs, nil
		}
		ancestorQC = *decodedExtraField.QuorumCert
	}
	return ancestorQC, lowerRoundQCs, higherRoundQCs, fmt.Errorf("[findAncestorQcThroughRound] Could not find ancestor QC")
}

func (f *Forensics) FindAncestorBlockHash(chain consensus.ChainReader, firstBlockInfo *utils.BlockInfo, secondBlockInfo *utils.BlockInfo) (common.Hash, []string, []string, error) {
	// Re-arrange by block number
	lowerBlockNumHash := firstBlockInfo.Hash
	higherBlockNumberHash := secondBlockInfo.Hash

	var ancestorToLowerBlockNumHashPath []string
	var ancestorToHigherBlockNumHashPath []string
	orderSwapped := false

	blockNumberDifference := big.NewInt(0).Sub(secondBlockInfo.Number, firstBlockInfo.Number).Int64()
	if blockNumberDifference < 0 {
		lowerBlockNumHash = secondBlockInfo.Hash
		higherBlockNumberHash = firstBlockInfo.Hash
		blockNumberDifference = -blockNumberDifference // and make it positive
		orderSwapped = true
	}
	ancestorToLowerBlockNumHashPath = append(ancestorToLowerBlockNumHashPath, lowerBlockNumHash.Hex())
	ancestorToHigherBlockNumHashPath = append(ancestorToHigherBlockNumHashPath, higherBlockNumberHash.Hex())

	// First, make their block number the same to start with
	for i := 0; i < int(blockNumberDifference); i++ {
		ph := chain.GetHeaderByHash(higherBlockNumberHash)
		if ph == nil {
			return common.Hash{}, ancestorToLowerBlockNumHashPath, ancestorToHigherBlockNumHashPath, fmt.Errorf("Unable to find parent block of hash %v", higherBlockNumberHash)
		}
		higherBlockNumberHash = ph.ParentHash
		ancestorToHigherBlockNumHashPath = append(ancestorToHigherBlockNumHashPath, ph.ParentHash.Hex())
	}

	// Now, they are on the same starting line, we try find the common ancestor
	for lowerBlockNumHash.Hex() != higherBlockNumberHash.Hex() {
		lowerBlockNumHash = chain.GetHeaderByHash(lowerBlockNumHash).ParentHash
		higherBlockNumberHash = chain.GetHeaderByHash(higherBlockNumberHash).ParentHash
		// Append the path
		ancestorToLowerBlockNumHashPath = append(ancestorToLowerBlockNumHashPath, lowerBlockNumHash.Hex())
		ancestorToHigherBlockNumHashPath = append(ancestorToHigherBlockNumHashPath, higherBlockNumberHash.Hex())
	}
	// Swap back the order. We must return in the order that matches what we acceptted in the parameter of firstBlock & secondBlock
	if orderSwapped {
		return lowerBlockNumHash, ancestorToHigherBlockNumHashPath, ancestorToLowerBlockNumHashPath, nil
	}
	return lowerBlockNumHash, ancestorToLowerBlockNumHashPath, ancestorToHigherBlockNumHashPath, nil
}
