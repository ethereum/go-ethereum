package beacon

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/portalnetwork/portalwire"
	"github.com/ethereum/go-ethereum/portalnetwork/storage"
	ssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	"github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
	"github.com/protolambda/ztyp/tree"
)

func SetupBeaconNetwork(addr string, bootNodes []*enode.Node) (*BeaconNetwork, error) {
	conf := portalwire.DefaultPortalProtocolConfig()
	if addr != "" {
		conf.ListenAddr = addr
	}
	if bootNodes != nil {
		conf.BootstrapNodes = bootNodes
	}

	addr1, err := net.ResolveUDPAddr("udp", conf.ListenAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.ListenUDP("udp", addr1)
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.GenerateKey()
	if err != nil {
		panic("couldn't generate key: " + err.Error())
	}

	discCfg := discover.Config{
		PrivateKey:  privKey,
		NetRestrict: conf.NetRestrict,
		Bootnodes:   conf.BootstrapNodes,
	}

	nodeDB, err := enode.OpenDB(conf.NodeDBPath)
	if err != nil {
		return nil, err
	}

	localNode := enode.NewLocalNode(nodeDB, privKey)
	localNode.SetFallbackIP(net.IP{127, 0, 0, 1})
	localNode.Set(portalwire.Tag)

	discV5, err := discover.ListenV5(conn, localNode, discCfg)
	if err != nil {
		return nil, err
	}

	contentQueue := make(chan *portalwire.ContentElement, 50)

	utpSocket := portalwire.NewPortalUtp(context.Background(), conf, discV5, conn)
	portalProtocol, err := portalwire.NewPortalProtocol(conf, portalwire.Beacon, privKey, conn, localNode, discV5, utpSocket, &storage.MockStorage{Db: make(map[string][]byte)}, contentQueue)
	if err != nil {
		return nil, err
	}

	return NewBeaconNetwork(portalProtocol), nil
}

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
	file, err := os.ReadFile(fmt.Sprintf("testdata/beacon/deneb/LightClientFinalityUpdate/ssz_random/case_%d/serialized.ssz_snappy", number))
	if err != nil {
		return ForkedLightClientFinalityUpdate{}, err
	}
	data, err := snappy.Decode(nil, file)
	if err != nil {
		return ForkedLightClientFinalityUpdate{}, err
	}
	update := &deneb.LightClientFinalityUpdate{}
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
	file, err := os.ReadFile(fmt.Sprintf("testdata/beacon/deneb/LightClientOptimisticUpdate/ssz_random/case_%d/serialized.ssz_snappy", number))
	if err != nil {
		return ForkedLightClientOptimisticUpdate{}, err
	}
	data, err := snappy.Decode(nil, file)
	if err != nil {
		return ForkedLightClientOptimisticUpdate{}, err
	}
	bootstrap := &ForkedLightClientOptimisticUpdate{}

	forkData := make([]byte, 0)
	forkData = append(forkData, Deneb[:]...)
	forkData = append(forkData, data...)
	err = bootstrap.Deserialize(configs.Mainnet, codec.NewDecodingReader(bytes.NewReader(forkData), uint64(len(forkData))))
	if err != nil {
		return ForkedLightClientOptimisticUpdate{}, err
	}
	return *bootstrap, nil
}

func GetHistorySummariesWithProof() (HistoricalSummariesWithProof, common.Root, error) {
	file, err := os.ReadFile("testdata/beacon/BeaconState/ssz_random/case_0/serialized.ssz_snappy")
	if err != nil {
		return HistoricalSummariesWithProof{}, common.Root{}, err
	}
	data, err := snappy.Decode(nil, file)
	if err != nil {
		return HistoricalSummariesWithProof{}, common.Root{}, err
	}

	beaconState := &deneb.BeaconState{}
	err = beaconState.Deserialize(configs.Mainnet, codec.NewDecodingReader(bytes.NewReader(data), uint64(len(data))))
	if err != nil {
		return HistoricalSummariesWithProof{}, common.Root{}, err
	}
	root := beaconState.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	proof, err := BuildHistoricalSummariesProof(*beaconState)
	if err != nil {
		return HistoricalSummariesWithProof{}, common.Root{}, err
	}
	summariesProof := [5]common.Bytes32{tree.Root(proof[0]), tree.Root(proof[1]), tree.Root(proof[2]), tree.Root(proof[3]), tree.Root(proof[4])}
	return HistoricalSummariesWithProof{
		EPOCH:               common.Epoch(uint64(beaconState.Slot) / 32),
		HistoricalSummaries: beaconState.HistoricalSummaries,
		Proof: HistoricalSummariesProof{
			Proof: summariesProof,
		},
	}, root, nil
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
	leaves[16] = beaconState.CurrentEpochParticipation.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[17] = beaconState.JustificationBits.HashTreeRoot(tree.GetHashFn())
	leaves[18] = beaconState.PreviousJustifiedCheckpoint.HashTreeRoot(tree.GetHashFn())
	leaves[19] = beaconState.CurrentJustifiedCheckpoint.HashTreeRoot(tree.GetHashFn())
	leaves[20] = beaconState.FinalizedCheckpoint.HashTreeRoot(tree.GetHashFn())
	leaves[21] = beaconState.InactivityScores.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[22] = beaconState.CurrentSyncCommittee.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[23] = beaconState.NextSyncCommittee.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[24] = beaconState.LatestExecutionPayloadHeader.HashTreeRoot(tree.GetHashFn())
	leaves[25] = beaconState.NextWithdrawalIndex.HashTreeRoot(tree.GetHashFn())
	leaves[26] = beaconState.NextWithdrawalValidatorIndex.HashTreeRoot(tree.GetHashFn())
	leaves[27] = beaconState.HistoricalSummaries.HashTreeRoot(configs.Mainnet, tree.GetHashFn())
	leaves[28] = tree.Root{}
	leaves[29] = tree.Root{}
	leaves[30] = tree.Root{}
	leaves[31] = tree.Root{}

	leavesBytes := make([][]byte, 0)
	for _, item := range leaves {
		dest := make([]byte, len(item))
		copy(dest, item[:])
		leavesBytes = append(leavesBytes, dest)
	}

	chunks, err := ssz.TreeFromChunks(leavesBytes)
	if err != nil {
		return nil, err
	}
	proof, err := chunks.Prove(59)
	if err != nil {
		return nil, err
	}
	return proof.Hashes, nil
}
