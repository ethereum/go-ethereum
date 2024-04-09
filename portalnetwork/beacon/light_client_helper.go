package beacon

import (
	"github.com/protolambda/zrnt/eth2/beacon/altair"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/zrnt/eth2/util/merkle"
	"github.com/protolambda/ztyp/tree"
	"github.com/prysmaticlabs/go-bitfield"
)

func ComputeSigningRoot(root common.Root, domain common.BLSDomain) common.Root {
	data := common.SigningData{
		ObjectRoot: root,
		Domain:     domain,
	}
	return data.HashTreeRoot(tree.GetHashFn())
}

func CalcSyncPeriod(slot uint64) uint64 {
	epoch := slot / 32 // 32 slots per epoch
	return epoch / 256 // 256 epochs per sync committee
}

func IsFinalityProofValid(attestedHeader common.BeaconBlockHeader, finalityHeader common.BeaconBlockHeader, finalityBranch altair.FinalizedRootProofBranch) bool {
	leaf := finalityHeader.HashTreeRoot(tree.GetHashFn())
	root := attestedHeader.StateRoot
	return merkle.VerifyMerkleBranch(leaf, finalityBranch[:], 6, 41, root)
}

func IsNextCommitteeProofValid(spec *common.Spec, attestedHeader common.BeaconBlockHeader, nextCommittee common.SyncCommittee, nextCommitteeBranch altair.SyncCommitteeProofBranch) bool {
	leaf := nextCommittee.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	root := attestedHeader.StateRoot
	return merkle.VerifyMerkleBranch(leaf, nextCommitteeBranch[:], 5, 23, root)
}

func GetParticipatingKeys(committee common.SyncCommittee, syncBits altair.SyncCommitteeBits) []common.BLSPubkey {
	bits := bitfield.Bitlist(syncBits)
	res := make([]common.BLSPubkey, 0, bits.Count())
	for i := 0; i < int(bits.Len()); i++ {
		if bits.BitAt(uint64(i)) {
			res = append(res, committee.Pubkeys[i])
		}
	}
	return res
}
