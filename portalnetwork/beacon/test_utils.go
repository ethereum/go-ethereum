package beacon

import (
	"bytes"
	"fmt"
	"os"

	ssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
)

func GetLightClientBootstrap(number uint8) (ForkedLightClientBootstrap, error) {
	file, err := os.ReadFile(fmt.Sprintf("testdata/beacon/LightClientBootstrap/ssz_random/case_%d/serialized.ssz_snappy", number))
	if err != nil {
		return ForkedLightClientBootstrap{}, err
	}
	data, err := snappy.Decode(nil, file)
	if err != nil {
		return ForkedLightClientBootstrap{}, err
	}
	bootstrap := &ForkedLightClientBootstrap{}

	forkData := make([]byte, 0)
	forkData = append(forkData, Capella[:]...)
	forkData = append(forkData, data...)
	err = bootstrap.Deserialize(configs.Mainnet, codec.NewDecodingReader(bytes.NewReader(forkData), uint64(len(forkData))))
	if err != nil {
		return ForkedLightClientBootstrap{}, err
	}
	return *bootstrap, nil
}

func GetClientUpdate(number uint8) (ForkedLightClientUpdate, error) {
	file, err := os.ReadFile(fmt.Sprintf("testdata/beacon/LightClientUpdate/ssz_random/case_%d/serialized.ssz_snappy", number))
	if err != nil {
		return ForkedLightClientUpdate{}, err
	}
	data, err := snappy.Decode(nil, file)
	if err != nil {
		return ForkedLightClientUpdate{}, err
	}
	update := &ForkedLightClientUpdate{}

	forkData := make([]byte, 0)
	forkData = append(forkData, Capella[:]...)
	forkData = append(forkData, data...)
	err = update.Deserialize(configs.Mainnet, codec.NewDecodingReader(bytes.NewReader(forkData), uint64(len(forkData))))
	if err != nil {
		return ForkedLightClientUpdate{}, err
	}
	return *update, nil
}

func GetLightClientFinalityUpdate(number uint8) (ForkedLightClientFinalityUpdate, error) {
	file, err := os.ReadFile(fmt.Sprintf("testdata/beacon/LightClientFinalityUpdate/ssz_random/case_%d/serialized.ssz_snappy", number))
	if err != nil {
		return ForkedLightClientFinalityUpdate{}, err
	}
	data, err := snappy.Decode(nil, file)
	if err != nil {
		return ForkedLightClientFinalityUpdate{}, err
	}
	update := &capella.LightClientFinalityUpdate{}
	err = update.Deserialize(configs.Mainnet, codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return ForkedLightClientFinalityUpdate{}, err
	}
	bootstrap := &ForkedLightClientFinalityUpdate{
		ForkDigest:                Deneb,
		LightClientFinalityUpdate: update,
	}

	return *bootstrap, nil
}

func GetLightClientOptimisticUpdate(number uint8) (ForkedLightClientOptimisticUpdate, error) {
	file, err := os.ReadFile(fmt.Sprintf("testdata/beacon/LightClientOptimisticUpdate/ssz_random/case_%d/serialized.ssz_snappy", number))
	if err != nil {
		return ForkedLightClientOptimisticUpdate{}, err
	}
	data, err := snappy.Decode(nil, file)
	if err != nil {
		return ForkedLightClientOptimisticUpdate{}, err
	}
	bootstrap := &ForkedLightClientOptimisticUpdate{}

	forkData := make([]byte, 0)
	forkData = append(forkData, Capella[:]...)
	forkData = append(forkData, data...)
	err = bootstrap.Deserialize(configs.Mainnet, codec.NewDecodingReader(bytes.NewReader(forkData), uint64(len(forkData))))
	if err != nil {
		return ForkedLightClientOptimisticUpdate{}, err
	}
	return *bootstrap, nil
}

func GetHistorySummariesWithProof() (HistoricalSummariesWithProof, error) {
	file, err := os.ReadFile("testdata/beacon/BeaconState/ssz_random/case_0/serialized.ssz_snappy")
	if err != nil {
		return HistoricalSummariesWithProof{}, err
	}
	data, err := snappy.Decode(nil, file)
	if err != nil {
		return HistoricalSummariesWithProof{}, err
	}

	beaconState := &deneb.BeaconState{}
	err = beaconState.Deserialize(configs.Mainnet, codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return HistoricalSummariesWithProof{}, err
	}
	proof, err := BuildHistoricalSummariesProof(*beaconState)
	if err != nil {
		return HistoricalSummariesWithProof{}, err
	}
	summariesProof := [5]common.Bytes32{tree.Root(proof[0]), tree.Root(proof[1]), tree.Root(proof[2]), tree.Root(proof[3]), tree.Root(proof[4])}
	return HistoricalSummariesWithProof{
		EPOCH:               common.Epoch(uint64(beaconState.Slot) / 32),
		HistoricalSummaries: beaconState.HistoricalSummaries,
		Proof: &HistoricalSummariesProof{
			Proof: summariesProof,
		},
	}, nil
}

func BuildHistoricalSummariesProof(beaconState deneb.BeaconState) ([][]byte, error) {
	leaves := make([][32]byte, 32)
	leaves[0] = beaconState.GenesisTime.HashTreeRoot(tree.GetHashFn())
	leaves[1] = beaconState.GenesisValidatorsRoot.HashTreeRoot(tree.GetHashFn())
	leaves[2] = beaconState.Slot.HashTreeRoot(tree.GetHashFn())
	leaves[3] = beaconState.Fork.HashTreeRoot(tree.GetHashFn())
	leaves[4] = beaconState.LatestBlockHeader.HashTreeRoot(tree.GetHashFn())
	leaves[5] = beaconState.BlockRoots.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[6] = beaconState.StateRoots.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[7] = beaconState.HistoricalRoots.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[8] = beaconState.Eth1Data.HashTreeRoot(tree.GetHashFn())
	leaves[9] = beaconState.Eth1DataVotes.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[10] = beaconState.Eth1DepositIndex.HashTreeRoot(tree.GetHashFn())
	leaves[11] = beaconState.Validators.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[12] = beaconState.Balances.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[13] = beaconState.RandaoMixes.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[14] = beaconState.Slashings.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[15] = beaconState.PreviousEpochParticipation.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[16] = beaconState.JustificationBits.HashTreeRoot(tree.GetHashFn())
	leaves[17] = beaconState.PreviousEpochParticipation.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[18] = beaconState.CurrentJustifiedCheckpoint.HashTreeRoot(tree.GetHashFn())
	leaves[19] = beaconState.FinalizedCheckpoint.HashTreeRoot(tree.GetHashFn())
	leaves[20] = beaconState.InactivityScores.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[21] = beaconState.CurrentSyncCommittee.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[22] = beaconState.NextSyncCommittee.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[23] = beaconState.LatestExecutionPayloadHeader.HashTreeRoot(tree.GetHashFn())
	leaves[24] = beaconState.NextWithdrawalIndex.HashTreeRoot(tree.GetHashFn())
	leaves[25] = beaconState.NextWithdrawalValidatorIndex.HashTreeRoot(tree.GetHashFn())
	leaves[26] = beaconState.HistoricalSummaries.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[27] = tree.Root{}
	leaves[28] = tree.Root{}
	leaves[29] = tree.Root{}
	leaves[30] = tree.Root{}
	leaves[31] = tree.Root{}

	leavesBytes := make([][]byte, 0)
	for _, item := range leaves {
		leavesBytes = append(leavesBytes, item[:])
	}

	tree, err := ssz.TreeFromChunks(leavesBytes)
	if err != nil {
		return nil, err
	}
	proof, err := tree.Prove(27)
	if err != nil {
		return nil, err
	}
	return proof.Hashes, nil
}
