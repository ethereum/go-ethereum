package engine_v2

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/log"
)

const (
	NUM_OF_FORENSICS_QC = 3
)

// Forensics instance. Placeholder for future properties to be added
type Forensics struct {
	HighestCommittedQCs []types.QuorumCert
	forensicsFeed       event.Feed
	scope               event.SubscriptionScope
}

// Initiate a forensics process
func NewForensics() *Forensics {
	return &Forensics{}
}

// SubscribeForensicsEvent registers a subscription of ForensicsEvent and
// starts sending event to the given channel.
func (f *Forensics) SubscribeForensicsEvent(ch chan<- types.ForensicsEvent) event.Subscription {
	return f.scope.Track(f.forensicsFeed.Subscribe(ch))
}

func (f *Forensics) ForensicsMonitoring(chain consensus.ChainReader, engine *XDPoS_v2, headerQcToBeCommitted []types.Header, incomingQC types.QuorumCert) error {
	f.ProcessForensics(chain, engine, incomingQC)
	return f.SetCommittedQCs(headerQcToBeCommitted, incomingQC)
}

// Set the forensics committed QCs list. The order is from grandparent to current header. i.e it shall follow the QC in its header as follow [hcqc1, hcqc2, hcqc3]
func (f *Forensics) SetCommittedQCs(headers []types.Header, incomingQC types.QuorumCert) error {
	// highestCommitQCs is an array, assign the parentBlockQc and its child as well as its grandchild QC into this array for forensics purposes.
	if len(headers) != NUM_OF_FORENSICS_QC-1 {
		log.Error("[SetCommittedQcs] Received input length not equal to 2", len(headers))
		return errors.New("received headers length not equal to 2 ")
	}

	var committedQCs []types.QuorumCert
	for i, h := range headers {
		var decodedExtraField types.ExtraFields_v2
		// Decode the qc1 and qc2
		err := utils.DecodeBytesExtraFields(h.Extra, &decodedExtraField)
		if err != nil {
			log.Error("[SetCommittedQCs] Fail to decode extra when committing QC to forensics", "err", err, "index", i)
			return err
		}
		if i != 0 {
			if decodedExtraField.QuorumCert.ProposedBlockInfo.Hash != headers[i-1].Hash() {
				log.Error("[SetCommittedQCs] Headers shall be on the same chain and in the right order", "parentHash", h.ParentHash.Hex(), "headers[i-1].Hash()", headers[i-1].Hash().Hex())
				return errors.New("headers shall be on the same chain and in the right order")
			} else if i == len(headers)-1 { // The last header shall be pointed by the incoming QC
				if incomingQC.ProposedBlockInfo.Hash != h.Hash() {
					log.Error("[SetCommittedQCs] incomingQc is not pointing at the last header received", "hash", h.Hash().Hex(), "incomingQC.ProposedBlockInfo.Hash", incomingQC.ProposedBlockInfo.Hash.Hex())
					return errors.New("incomingQc is not pointing at the last header received")
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
func (f *Forensics) ProcessForensics(chain consensus.ChainReader, engine *XDPoS_v2, incomingQC types.QuorumCert) error {
	return nil
	log.Debug("Received a QC in forensics", "QC", incomingQC)
	// Clone the values to a temporary variable
	highestCommittedQCs := f.HighestCommittedQCs
	if len(highestCommittedQCs) != NUM_OF_FORENSICS_QC {
		log.Error("[ProcessForensics] HighestCommittedQCs value not set", "incomingQcProposedBlockHash", incomingQC.ProposedBlockInfo.Hash, "incomingQcProposedBlockNumber", incomingQC.ProposedBlockInfo.Number.Uint64(), "incomingQcProposedBlockRound", incomingQC.ProposedBlockInfo.Round)
		return errors.New("HighestCommittedQCs value not set")
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
			log.Error("[ProcessForensics] Error while trying to find ancestor QC through round number", "err", err)
		}
		f.SendForensicProof(chain, engine, ancestorQC, lowerRoundQCs[NUM_OF_FORENSICS_QC-1])
	}

	return nil
}

// Last step of forensics which sends out detailed proof to report service.
func (f *Forensics) SendForensicProof(chain consensus.ChainReader, engine *XDPoS_v2, firstQc types.QuorumCert, secondQc types.QuorumCert) error {
	// Re-order the QC by its round number to make the function cleaner.
	lowerRoundQC := firstQc
	higherRoundQC := secondQc

	if secondQc.ProposedBlockInfo.Round < firstQc.ProposedBlockInfo.Round {
		lowerRoundQC = secondQc
		higherRoundQC = firstQc
	}

	// Find common ancestor block
	ancestorHash, ancestorToLowerRoundPath, ancestorToHigherRoundPath, err := f.FindAncestorBlockHash(chain, lowerRoundQC.ProposedBlockInfo, higherRoundQC.ProposedBlockInfo)
	if err != nil {
		log.Error("[SendForensicProof] Error while trying to find ancestor block hash", "err", err)
		return err
	}

	// Check if two QCs are across epoch, this is used as a indicator for the "prone to attack" scenario
	lowerRoundQcEpochSwitchInfo, err := engine.getEpochSwitchInfo(chain, nil, lowerRoundQC.ProposedBlockInfo.Hash)
	if err != nil {
		log.Error("[SendForensicProof] Errir while trying to find lowerRoundQcEpochSwitchInfo", "lowerRoundQC.ProposedBlockInfo.Hash", lowerRoundQC.ProposedBlockInfo.Hash, "err", err)
		return err
	}
	higherRoundQcEpochSwitchInfo, err := engine.getEpochSwitchInfo(chain, nil, higherRoundQC.ProposedBlockInfo.Hash)
	if err != nil {
		log.Error("[SendForensicProof] Errir while trying to find higherRoundQcEpochSwitchInfo", "higherRoundQC.ProposedBlockInfo.Hash", higherRoundQC.ProposedBlockInfo.Hash, "err", err)
		return err
	}
	accrossEpoches := false
	if lowerRoundQcEpochSwitchInfo.EpochSwitchBlockInfo.Hash != higherRoundQcEpochSwitchInfo.EpochSwitchBlockInfo.Hash {
		accrossEpoches = true
	}

	ancestorBlock := chain.GetHeaderByHash(ancestorHash)

	if ancestorBlock == nil {
		log.Error("[SendForensicProof] Unable to find the ancestor block by its hash", "Hash", ancestorHash)
		return errors.New("can't find ancestor block via hash")
	}

	content, err := json.Marshal(&types.ForensicsContent{
		DivergingBlockHash:   ancestorHash.Hex(),
		AcrossEpoch:          accrossEpoches,
		DivergingBlockNumber: ancestorBlock.Number.Uint64(),
		SmallerRoundInfo: &types.ForensicsInfo{
			HashPath:        ancestorToLowerRoundPath,
			QuorumCert:      lowerRoundQC,
			SignerAddresses: f.getQcSignerAddresses(lowerRoundQC),
		},
		LargerRoundInfo: &types.ForensicsInfo{
			HashPath:        ancestorToHigherRoundPath,
			QuorumCert:      higherRoundQC,
			SignerAddresses: f.getQcSignerAddresses(higherRoundQC),
		},
	})

	if err != nil {
		log.Error("[SendForensicProof] fail to json stringify forensics content", "err", err)
		return err
	}

	forensicsProof := &types.ForensicProof{
		Id:            generateForensicsId(ancestorHash.Hex(), &lowerRoundQC, &higherRoundQC),
		ForensicsType: "QC",
		Content:       string(content),
	}
	log.Info("Forensics proof report generated, sending to the stats server", "forensicsProof", forensicsProof)
	go f.forensicsFeed.Send(types.ForensicsEvent{ForensicsProof: forensicsProof})
	return nil
}

// Utils function to help find the n-th previous QC. It returns an array of QC in ascending order including the currentQc as the last item in the array
func (f *Forensics) findAncestorQCs(chain consensus.ChainReader, currentQc types.QuorumCert, distanceFromCurrrentQc int) ([]types.QuorumCert, error) {
	var quorumCerts []types.QuorumCert
	quorumCertificate := currentQc
	// Append the initial value
	quorumCerts = append(quorumCerts, quorumCertificate)
	// Append the parents
	for i := 0; i < distanceFromCurrrentQc; i++ {
		parentHash := quorumCertificate.ProposedBlockInfo.Hash
		parentHeader := chain.GetHeaderByHash(parentHash)
		if parentHeader == nil {
			log.Error("[findAncestorQCs] Forensics findAncestorQCs unable to find its parent block header", "ParentHash", parentHash.Hex())
			return nil, errors.New("unable to find parent block header in forensics")
		}
		var decodedExtraField types.ExtraFields_v2
		err := utils.DecodeBytesExtraFields(parentHeader.Extra, &decodedExtraField)
		if err != nil {
			log.Error("[findAncestorQCs] Error while trying to decode from parent block extra", "BlockNum", parentHeader.Number.Int64(), "ParentHash", parentHash.Hex())
		}
		quorumCertificate = *decodedExtraField.QuorumCert
		quorumCerts = append(quorumCerts, quorumCertificate)
	}
	// The quorumCerts is in the reverse order, we need to flip it
	var quorumCertsInAscendingOrder []types.QuorumCert
	for i := len(quorumCerts) - 1; i >= 0; i-- {
		quorumCertsInAscendingOrder = append(quorumCertsInAscendingOrder, quorumCerts[i])
	}
	return quorumCertsInAscendingOrder, nil
}

// Check whether two provided QC set are on the same chain
func (f *Forensics) checkQCsOnTheSameChain(chain consensus.ChainReader, highestCommittedQCs []types.QuorumCert, incomingQCandItsParents []types.QuorumCert) (bool, error) {
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
		var decodedExtraField types.ExtraFields_v2
		err := utils.DecodeBytesExtraFields(parentHeader.Extra, &decodedExtraField)
		if err != nil {
			log.Error("[checkQCsOnTheSameChain] Fail to decode extra when checking the two QCs set on the same chain", "err", err)
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
func (f *Forensics) findQCsInSameRound(quorumCerts1 []types.QuorumCert, quorumCerts2 []types.QuorumCert) (bool, types.QuorumCert, types.QuorumCert) {
	for _, quorumCert1 := range quorumCerts1 {
		for _, quorumCert2 := range quorumCerts2 {
			if quorumCert1.ProposedBlockInfo.Round == quorumCert2.ProposedBlockInfo.Round {
				return true, quorumCert1, quorumCert2
			}
		}
	}
	return false, types.QuorumCert{}, types.QuorumCert{}
}

// Find the signer list from QC signatures
func (f *Forensics) getQcSignerAddresses(quorumCert types.QuorumCert) []string {
	signerList := make([]string, 0, len(quorumCert.Signatures))

	// The QC signatures are signed by votes special struct VoteForSign
	quorumCertSignedHash := types.VoteSigHash(&types.VoteForSign{
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
func (f *Forensics) findAncestorQcThroughRound(chain consensus.ChainReader, highestCommittedQCs []types.QuorumCert, incomingQCandItsParents []types.QuorumCert) (types.QuorumCert, []types.QuorumCert, []types.QuorumCert, error) {
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
		var decodedExtraField types.ExtraFields_v2
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
	return ancestorQC, lowerRoundQCs, higherRoundQCs, errors.New("[findAncestorQcThroughRound] Could not find ancestor QC")
}

func (f *Forensics) FindAncestorBlockHash(chain consensus.ChainReader, firstBlockInfo *types.BlockInfo, secondBlockInfo *types.BlockInfo) (common.Hash, []string, []string, error) {
	// Re-arrange by block number
	lowerBlockNumHash := firstBlockInfo.Hash
	higherBlockNumberHash := secondBlockInfo.Hash

	var lowerBlockNumToAncestorHashPath []string
	var higherBlockToAncestorNumHashPath []string
	orderSwapped := false

	blockNumberDifference := big.NewInt(0).Sub(secondBlockInfo.Number, firstBlockInfo.Number).Int64()
	if blockNumberDifference < 0 {
		lowerBlockNumHash = secondBlockInfo.Hash
		higherBlockNumberHash = firstBlockInfo.Hash
		blockNumberDifference = -blockNumberDifference // and make it positive
		orderSwapped = true
	}
	lowerBlockNumToAncestorHashPath = append(lowerBlockNumToAncestorHashPath, lowerBlockNumHash.Hex())
	higherBlockToAncestorNumHashPath = append(higherBlockToAncestorNumHashPath, higherBlockNumberHash.Hex())

	// First, make their block number the same to start with
	for i := 0; i < int(blockNumberDifference); i++ {
		ph := chain.GetHeaderByHash(higherBlockNumberHash)
		if ph == nil {
			return common.Hash{}, lowerBlockNumToAncestorHashPath, higherBlockToAncestorNumHashPath, fmt.Errorf("unable to find parent block of hash %v", higherBlockNumberHash)
		}
		higherBlockNumberHash = ph.ParentHash
		higherBlockToAncestorNumHashPath = append(higherBlockToAncestorNumHashPath, ph.ParentHash.Hex())
	}

	// Now, they are on the same starting line, we try find the common ancestor
	for lowerBlockNumHash != higherBlockNumberHash {
		lowerBlockNumHash = chain.GetHeaderByHash(lowerBlockNumHash).ParentHash
		higherBlockNumberHash = chain.GetHeaderByHash(higherBlockNumberHash).ParentHash
		// Append the path
		lowerBlockNumToAncestorHashPath = append(lowerBlockNumToAncestorHashPath, lowerBlockNumHash.Hex())
		higherBlockToAncestorNumHashPath = append(higherBlockToAncestorNumHashPath, higherBlockNumberHash.Hex())
	}

	// Reverse the list order as it's from ancestor to X block path.
	ancestorToLowerBlockNumHashPath := reverse(lowerBlockNumToAncestorHashPath)
	ancestorToHigherBlockNumHashPath := reverse(higherBlockToAncestorNumHashPath)
	// Swap back the order. We must return in the order that matches what we acceptted in the parameter of firstBlock & secondBlock
	if orderSwapped {
		return lowerBlockNumHash, ancestorToHigherBlockNumHashPath, ancestorToLowerBlockNumHashPath, nil
	}
	return lowerBlockNumHash, ancestorToLowerBlockNumHashPath, ancestorToHigherBlockNumHashPath, nil
}

func generateForensicsId(divergingHash string, qc1 *types.QuorumCert, qc2 *types.QuorumCert) string {
	keysList := []string{divergingHash, qc1.ProposedBlockInfo.Hash.Hex(), qc2.ProposedBlockInfo.Hash.Hex()}
	return strings.Join(keysList[:], ":")
}

func reverse(ss []string) []string {
	last := len(ss) - 1
	for i := 0; i < len(ss)/2; i++ {
		ss[i], ss[last-i] = ss[last-i], ss[i]
	}
	return ss
}

func generateVoteEquivocationId(signer common.Address, round1, round2 types.Round) string {
	return fmt.Sprintf("%x:%d:%d", signer, round1, round2)
}

/*
Entry point for processing vote equivocation.
Triggered once handle vote is successfully.
Forensics runs in a seperate go routine as its no system critical
Link to the flow diagram: https://hashlabs.atlassian.net/wiki/spaces/HASHLABS/pages/99516417/Vote+Equivocation+detection+specification
*/
func (f *Forensics) ProcessVoteEquivocation(chain consensus.ChainReader, engine *XDPoS_v2, incomingVote *types.Vote) error {
	return nil
	log.Debug("Received a vote in forensics", "vote", incomingVote)
	// Clone the values to a temporary variable
	highestCommittedQCs := f.HighestCommittedQCs
	if len(highestCommittedQCs) != NUM_OF_FORENSICS_QC {
		log.Error("[ProcessVoteEquivocation] HighestCommittedQCs value not set", "incomingVoteProposedBlockHash", incomingVote.ProposedBlockInfo.Hash, "incomingVoteProposedBlockNumber", incomingVote.ProposedBlockInfo.Number.Uint64(), "incomingVoteProposedBlockRound", incomingVote.ProposedBlockInfo.Round)
		return errors.New("HighestCommittedQCs value not set")
	}
	if incomingVote.ProposedBlockInfo.Round < highestCommittedQCs[NUM_OF_FORENSICS_QC-1].ProposedBlockInfo.Round {
		log.Debug("Received a too old vote in forensics", "vote", incomingVote)
		return nil
	}
	// is vote extending committed block
	isOnTheChain, err := f.isExtendingFromAncestor(chain, incomingVote.ProposedBlockInfo, highestCommittedQCs[0].ProposedBlockInfo)
	if err != nil {
		return err
	}
	if isOnTheChain {
		// Passed the checking, nothing suspecious.
		log.Debug("[ProcessVoteEquivocation] Passed forensics checking, nothing suspecious need to be reported", "incomingVoteProposedBlockHash", incomingVote.ProposedBlockInfo.Hash, "incomingVoteProposedBlockNumber", incomingVote.ProposedBlockInfo.Number.Uint64(), "incomingVoteProposedBlockRound", incomingVote.ProposedBlockInfo.Round)
		return nil
	}
	// Trigger the safety Alarm if failed
	isVoteBlamed, parentQC, err := f.isVoteBlamed(chain, highestCommittedQCs, incomingVote)
	if err != nil {
		log.Error("[ProcessVoteEquivocation] Error while trying to call isVoteBlamed", "error", err)
		return err
	}
	if isVoteBlamed {
		signer, err := GetVoteSignerAddresses(incomingVote)
		if err != nil {
			log.Error("[ProcessVoteEquivocation] GetVoteSignerAddresses", "error", err)
		}
		qc := highestCommittedQCs[NUM_OF_FORENSICS_QC-1]
		for _, signature := range qc.Signatures {
			voteFromQC := &types.Vote{ProposedBlockInfo: qc.ProposedBlockInfo, Signature: signature, GapNumber: qc.GapNumber}
			signerFromQC, err := GetVoteSignerAddresses(voteFromQC)
			if err != nil {
				log.Error("[ProcessVoteEquivocation] GetVoteSignerAddresses", "error", err)
				return err
			}
			if signerFromQC == signer {
				f.SendVoteEquivocationProof(incomingVote, voteFromQC, signer)
				break
			}
		}
		// if no same-signer vote, nothing to report
	} else {
		// use the parent QC to do forensics
		f.ProcessForensics(chain, engine, *parentQC)
	}

	return nil
}

func (f *Forensics) isExtendingFromAncestor(blockChainReader consensus.ChainReader, currentBlock *types.BlockInfo, ancestorBlock *types.BlockInfo) (bool, error) {
	blockNumDiff := int(big.NewInt(0).Sub(currentBlock.Number, ancestorBlock.Number).Int64())

	nextBlockHash := currentBlock.Hash
	for i := 0; i < blockNumDiff; i++ {
		parentBlock := blockChainReader.GetHeaderByHash(nextBlockHash)
		if parentBlock == nil {
			return false, fmt.Errorf("could not find its parent block when checking whether currentBlock %v with hash %v is extending from the ancestorBlock %v", currentBlock.Number, currentBlock.Hash, ancestorBlock.Number)
		} else {
			nextBlockHash = parentBlock.ParentHash
		}
		log.Debug("[isExtendingFromAncestor] Found parent block", "CurrentBlockHash", currentBlock.Hash, "ParentHash", nextBlockHash)
	}

	if nextBlockHash == ancestorBlock.Hash {
		return true, nil
	}
	return false, nil
}

func (f *Forensics) isVoteBlamed(chain consensus.ChainReader, highestCommittedQCs []types.QuorumCert, incomingVote *types.Vote) (bool, *types.QuorumCert, error) {
	proposedBlock := chain.GetHeaderByHash(incomingVote.ProposedBlockInfo.Hash)
	var decodedExtraField types.ExtraFields_v2
	err := utils.DecodeBytesExtraFields(proposedBlock.Extra, &decodedExtraField)
	if err != nil {
		log.Error("[findAncestorVoteThroughRound] Error while trying to decode extra field", "ProposedBlockInfo.Hash", incomingVote.ProposedBlockInfo.Hash)
		return false, nil, err
	}
	// Found the parent QC, if its round < hcqc3's round, return true
	if decodedExtraField.QuorumCert.ProposedBlockInfo.Round < highestCommittedQCs[NUM_OF_FORENSICS_QC-1].ProposedBlockInfo.Round {
		return true, decodedExtraField.QuorumCert, nil
	}
	return false, decodedExtraField.QuorumCert, nil
}

func (f *Forensics) DetectEquivocationInVotePool(vote *types.Vote, votePool *utils.Pool) {
	return
	poolKey := vote.PoolKey()
	votePoolKeys := votePool.PoolObjKeysList()
	signer, err := GetVoteSignerAddresses(vote)
	if err != nil {
		log.Error("[detectEquivocationInVotePool]", "err", err)
	}

	for _, k := range votePoolKeys {
		if k == poolKey {
			continue
		}
		keyedRound, err := strconv.ParseInt(strings.Split(k, ":")[0], 10, 64)
		if err != nil {
			log.Error("[detectEquivocationInVotePool] Error while trying to get keyedRound inside pool", "Error", err)
			continue
		}
		if types.Round(keyedRound) == vote.ProposedBlockInfo.Round {
			votes := votePool.GetObjsByKey(k)
			for _, v := range votes {
				voteTransfered, ok := v.(*types.Vote)
				if !ok {
					log.Warn("[detectEquivocationInVotePool] obj type is not vote, potential a bug in votePool")
					continue
				}
				signer2, err := GetVoteSignerAddresses(voteTransfered)
				if err != nil {
					log.Warn("[detectEquivocationInVotePool]", "err", err)
					continue
				}
				if signer == signer2 {
					f.SendVoteEquivocationProof(vote, voteTransfered, signer)
				}
			}
		}
	}
}

func (f *Forensics) SendVoteEquivocationProof(vote1, vote2 *types.Vote, signer common.Address) error {
	smallerRoundVote := vote1
	largerRoundVote := vote2
	if vote1.ProposedBlockInfo.Round > vote2.ProposedBlockInfo.Round {
		smallerRoundVote = vote2
		largerRoundVote = vote1
	}
	content, err := json.Marshal(&types.VoteEquivocationContent{
		SmallerRoundVote: smallerRoundVote,
		LargerRoundVote:  largerRoundVote,
		Signer:           signer,
	})
	if err != nil {
		log.Error("[SendVoteEquivocationProof] fail to json stringify forensics content", "err", err)
		return err
	}
	forensicsProof := &types.ForensicProof{
		Id:            generateVoteEquivocationId(signer, smallerRoundVote.ProposedBlockInfo.Round, largerRoundVote.ProposedBlockInfo.Round),
		ForensicsType: "Vote",
		Content:       string(content),
	}
	log.Info("Forensics proof report generated, sending to the stats server", "forensicsProof", forensicsProof)
	go f.forensicsFeed.Send(types.ForensicsEvent{ForensicsProof: forensicsProof})
	return nil
}

func GetVoteSignerAddresses(vote *types.Vote) (common.Address, error) {
	// The QC signatures are signed by votes special struct VoteForSign
	signHash := types.VoteSigHash(&types.VoteForSign{
		ProposedBlockInfo: vote.ProposedBlockInfo,
		GapNumber:         vote.GapNumber,
	})
	var signerAddress common.Address
	pubkey, err := crypto.Ecrecover(signHash.Bytes(), vote.Signature)
	if err != nil {
		return signerAddress, fmt.Errorf("fail to Ecrecover signer from the vote: %v", vote)
	}
	copy(signerAddress[:], crypto.Keccak256(pubkey[1:])[12:])
	return signerAddress, nil
}
