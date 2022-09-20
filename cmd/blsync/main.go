// Copyright 2022 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/api"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/common/mclock"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

func main() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	app := flags.NewApp("beacon light syncer tool")
	app.Flags = []cli.Flag{
		utils.BeaconApiFlag,
		utils.BeaconApiHeaderFlag,
		utils.BeaconThresholdFlag,
		utils.BeaconNoFilterFlag,
		utils.BeaconConfigFlag,
		utils.BeaconGenesisRootFlag,
		utils.BeaconGenesisTimeFlag,
		utils.BeaconCheckpointFlag,
		//TODO datadir for optional permanent database
		utils.MainnetFlag,
		utils.SepoliaFlag,
		utils.GoerliFlag,
		utils.BlsyncApiFlag,
		utils.BlsyncJWTSecretFlag,
	}
	app.Action = blsync

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var (
	stateProofFormat    merkle.ProofFormat // requested multiproof format
	execBlockIndex      int                // index of execution block root in proof.Values where proof.Format == stateProofFormat
	finalizedBlockIndex int                // index of finalized block root in proof.Values where proof.Format == stateProofFormat
)

func blsync(ctx *cli.Context) error {
	if !ctx.IsSet(utils.BeaconApiFlag.Name) {
		utils.Fatalf("Beacon node light client API URL not specified")
	}
	stateProofFormat = merkle.NewIndexMapFormat().AddLeaf(params.BsiExecHead, nil).AddLeaf(params.BsiFinalBlock, nil)
	var (
		stateIndexMap = merkle.ProofFormatIndexMap(stateProofFormat)
		chainConfig   = makeChainConfig(ctx)
		customHeader  = make(map[string]string)
	)
	execBlockIndex = stateIndexMap[params.BsiExecHead]
	finalizedBlockIndex = stateIndexMap[params.BsiFinalBlock]

	for _, s := range utils.SplitAndTrim(ctx.String(utils.BeaconApiHeaderFlag.Name)) {
		kv := strings.Split(s, ":")
		if len(kv) != 2 {
			utils.Fatalf("Invalid custom API header entry: %s", s)
		}
		customHeader[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
	}

	var (
		beaconApi       = api.NewBeaconLightApi(ctx.String(utils.BeaconApiFlag.Name), customHeader)
		db              = memorydb.New()
		threshold       = ctx.Int(utils.BeaconThresholdFlag.Name)
		committeeChain  = light.NewCommitteeChain(db, chainConfig.Forks, threshold, !ctx.Bool(utils.BeaconNoFilterFlag.Name), light.BLSVerifier{}, &mclock.System{}, func() int64 { return time.Now().UnixNano() })
		checkpointStore = light.NewCheckpointStore(db, committeeChain)
		headTracker     = light.NewHeadTracker(committeeChain)
		scheduler       = request.NewScheduler()
	)
	committeeChain.SetGenesisData(chainConfig.GenesisData)

	checkpointInit := sync.NewCheckpointInit(committeeChain, checkpointStore, chainConfig.Checkpoint)
	forwardSync := sync.NewForwardUpdateSyncer(committeeChain)
	headSync := sync.NewHeadSyncer(headTracker, committeeChain)
	scheduler.RegisterModule(checkpointInit)
	scheduler.RegisterModule(forwardSync)
	scheduler.RegisterModule(headSync)
	scheduler.AddTriggers(forwardSync, []*request.ModuleTrigger{&checkpointInit.InitTrigger, &forwardSync.NewUpdateTrigger, &headSync.SignedHeadTrigger})
	scheduler.AddTriggers(headSync, []*request.ModuleTrigger{&forwardSync.NewUpdateTrigger})

	syncer := &execSyncer{
		api:           beaconApi,
		client:        makeRPCClient(ctx),
		execRootCache: lru.NewCache[common.Hash, common.Hash](1000),
	}
	headTracker.Subscribe(threshold, syncer.newHead)
	scheduler.Start()
	scheduler.RegisterServer(api.NewSyncServer(beaconApi))
	<-ctx.Done()
	scheduler.Stop()
	return nil
}

func callNewPayloadV1(client *rpc.Client, block *ctypes.Block) (string, error) {
	var resp engine.PayloadStatusV1
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err := client.CallContext(ctx, &resp, "engine_newPayloadV1", *engine.BlockToExecutableData(block, nil).ExecutionPayload)
	cancel()
	return resp.Status, err
}

func callForkchoiceUpdatedV1(client *rpc.Client, headHash, finalizedHash common.Hash) (string, error) {
	var resp engine.ForkChoiceResponse
	update := engine.ForkchoiceStateV1{
		HeadBlockHash:      headHash,
		SafeBlockHash:      finalizedHash,
		FinalizedBlockHash: finalizedHash,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err := client.CallContext(ctx, &resp, "engine_forkchoiceUpdatedV1", update, nil)
	cancel()
	return resp.PayloadStatus.Status, err
}

type execSyncer struct {
	api           *api.BeaconLightApi
	sub           *api.StateProofSub
	client        *rpc.Client
	execRootCache *lru.Cache[common.Hash, common.Hash] // beacon block root -> execution block root
}

// newHead fetches state proofs to determine the execution block root and calls
// the engine API if specified
func (e *execSyncer) newHead(signedHead types.SignedHead) {
	head := signedHead.Header
	log.Info("Received new beacon head", "slot", head.Slot, "blockRoot", head.Hash())
	block, err := e.api.GetExecutionPayload(head)
	if err != nil {
		log.Error("Error fetching execution payload from beacon API", "error", err)
		return
	}
	blockRoot := block.Hash()
	var finalizedExecRoot common.Hash
	if e.sub == nil {
		if sub, err := e.api.SubscribeStateProof(stateProofFormat, 0, 1); err == nil {
			log.Info("Successfully created beacon state subscription")
			e.sub = sub
		} else {
			log.Error("Failed to create beacon state subscription", "error", err)
			return
		}
	}
	proof, err := e.sub.Get(head.StateRoot)
	if err == nil {
		var (
			execBlockRoot       = common.Hash(proof.Values[execBlockIndex])
			finalizedBeaconRoot = common.Hash(proof.Values[finalizedBlockIndex])
			beaconRoot          = head.Hash()
		)
		e.execRootCache.Add(beaconRoot, execBlockRoot)
		if blockRoot != execBlockRoot {
			log.Error("Execution payload block hash does not match value in beacon state", "expected", execBlockRoot, "got", block.Hash())
			return
		}
		if _, ok := e.execRootCache.Get(head.ParentRoot); !ok {
			e.fetchExecRoots(head.ParentRoot)
		}
		finalizedExecRoot, _ = e.execRootCache.Get(finalizedBeaconRoot)
	} else if err != api.ErrNotFound {
		log.Error("Error fetching state proof from beacon API", "error", err)
	}
	if e.client == nil { // dry run, no engine API specified
		log.Info("New execution block retrieved", "block number", block.NumberU64(), "block hash", blockRoot, "finalized block hash", finalizedExecRoot)
		return
	}
	if status, err := callNewPayloadV1(e.client, block); err == nil {
		log.Info("Successful NewPayload", "block number", block.NumberU64(), "block hash", blockRoot, "status", status)
	} else {
		log.Error("Failed NewPayload", "block number", block.NumberU64(), "block hash", blockRoot, "error", err)
	}
	if status, err := callForkchoiceUpdatedV1(e.client, blockRoot, finalizedExecRoot); err == nil {
		log.Info("Successful ForkchoiceUpdated", "head", blockRoot, "finalized", finalizedExecRoot, "status", status)
	} else {
		log.Error("Failed ForkchoiceUpdated", "head", blockRoot, "finalized", finalizedExecRoot, "error", err)
	}
}

func (e *execSyncer) fetchExecRoots(blockRoot common.Hash) {
	for maxFetch := 256; maxFetch > 0; maxFetch-- {
		header, err := e.api.GetHeader(blockRoot)
		if err != nil {
			break
		}
		proof, err := e.sub.Get(header.StateRoot)
		if err != nil {
			// exit silently because we expect running into an error when parent is unknown
			break
		}
		e.execRootCache.Add(header.Hash(), common.Hash(proof.Values[execBlockIndex]))
		if _, ok := e.execRootCache.Get(header.ParentRoot); ok {
			break
		}
		blockRoot = header.ParentRoot
	}
}
