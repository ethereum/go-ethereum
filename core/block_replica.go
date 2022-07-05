package core

import (
	"bytes"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

type BlockReplicationEvent struct {
	Hash string
	Data []byte
}

func (bc *BlockChain) createBlockReplica(block *types.Block, replicaConfig *ReplicaConfig, chainConfig *params.ChainConfig, stateSpecimen *types.StateSpecimen) error {
	//block replica
	exportBlockReplica, err := bc.createReplica(block, replicaConfig, chainConfig, stateSpecimen)
	if err != nil {
		return err
	}
	//encode to rlp
	blockReplicaRLP, err := rlp.EncodeToBytes(exportBlockReplica)
	if err != nil {
		return err
	}

	sHash := block.Hash().String()

	if atomic.LoadUint32(replicaConfig.HistoricalBlocksSynced) == 0 {
		//log.Info("BSP running in Live mode", "Unexported block ", block.NumberU64(), "hash", sHash)
		return nil
	} else if atomic.LoadUint32(replicaConfig.HistoricalBlocksSynced) == 1 {
		log.Info("Creating Block Specimen", "Exported block", block.NumberU64(), "hash", sHash)
		bc.blockReplicationFeed.Send(BlockReplicationEvent{
			sHash,
			blockReplicaRLP,
		})
		return nil
	} else {
		return fmt.Errorf("error in setting atomic config historical block sync: %v", replicaConfig.HistoricalBlocksSynced)
	}

}

func (bc *BlockChain) createReplica(block *types.Block, replicaConfig *ReplicaConfig, chainConfig *params.ChainConfig, stateSpecimen *types.StateSpecimen) (*types.ExportBlockReplica, error) {

	bHash := block.Hash()
	bNum := block.NumberU64()

	//totalDifficulty
	tdRLP := rawdb.ReadTdRLP(bc.db, bHash, bNum)
	td := new(big.Int)
	if err := rlp.Decode(bytes.NewReader(tdRLP), td); err != nil {
		log.Error("Invalid block total difficulty RLP ", "hash ", bHash, "err", err)
		return nil, err
	}

	//header
	headerRLP := rawdb.ReadHeaderRLP(bc.db, bHash, bNum)
	header := new(types.Header)
	if err := rlp.Decode(bytes.NewReader(headerRLP), header); err != nil {
		log.Error("Invalid block header RLP ", "hash ", bHash, "err ", err)
		return nil, err
	}

	//transactions
	txsExp := make([]*types.TransactionForExport, len(block.Transactions()))
	txsRlp := make([]*types.TransactionExportRLP, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		txsExp[i] = (*types.TransactionForExport)(tx)
		txsRlp[i] = txsExp[i].ExportTx()
	}

	//receipts
	receipts := rawdb.ReadRawReceipts(bc.db, bHash, bNum)
	receiptsExp := make([]*types.ReceiptForExport, len(receipts))
	receiptsRlp := make([]*types.ReceiptExportRLP, len(receipts))
	for i, receipt := range receipts {
		receiptsExp[i] = (*types.ReceiptForExport)(receipt)
		receiptsRlp[i] = receiptsExp[i].ExportReceipt()
	}

	//senders
	signer := types.MakeSigner(bc.chainConfig, block.Number())
	senders := make([]common.Address, 0, len(block.Transactions()))
	for _, tx := range block.Transactions() {
		sender, err := types.Sender(signer, tx)
		if err != nil {
			return nil, err
		} else {
			senders = append(senders, sender)
		}
	}

	//uncles
	uncles := block.Uncles()

	//block replica export
	if replicaConfig.EnableSpecimen && replicaConfig.EnableResult {
		exportBlockReplica := &types.ExportBlockReplica{
			Type:         "block-replica",
			NetworkId:    chainConfig.ChainID.Uint64(),
			Hash:         bHash,
			TotalDiff:    td,
			Header:       header,
			Transactions: txsRlp,
			Uncles:       uncles,
			Receipts:     receiptsRlp,
			Senders:      senders,
			State:        stateSpecimen,
		}
		log.Debug("Exporting full block-replica")
		return exportBlockReplica, nil
	} else if replicaConfig.EnableSpecimen && !replicaConfig.EnableResult {
		exportBlockReplica := &types.ExportBlockReplica{
			Type:         "block-specimen",
			NetworkId:    chainConfig.ChainID.Uint64(),
			Hash:         bHash,
			TotalDiff:    &big.Int{},
			Header:       header,
			Transactions: txsRlp,
			Uncles:       uncles,
			Receipts:     []*types.ReceiptExportRLP{},
			Senders:      []common.Address{},
			State:        stateSpecimen,
		}
		log.Debug("Exporting block-specimen only")
		return exportBlockReplica, nil
	} else if !replicaConfig.EnableSpecimen && replicaConfig.EnableResult {
		exportBlockReplica := &types.ExportBlockReplica{
			Type:         "block-result",
			NetworkId:    chainConfig.ChainID.Uint64(),
			Hash:         bHash,
			TotalDiff:    td,
			Header:       header,
			Transactions: txsRlp,
			Uncles:       uncles,
			Receipts:     receiptsRlp,
			Senders:      senders,
			State:        &types.StateSpecimen{},
		}
		log.Debug("Exporting block-result only")
		return exportBlockReplica, nil
	} else {
		return nil, fmt.Errorf("--replication.targets flag is invalid without --replica.specimen and/or --replica.result")
	}
}

// SubscribeChainReplicationEvent registers a subscription of ChainReplicationEvent.
func (bc *BlockChain) SubscribeBlockReplicationEvent(ch chan<- BlockReplicationEvent) event.Subscription {
	return bc.scope.Track(bc.blockReplicationFeed.Subscribe(ch))
}

func (bc *BlockChain) SetBlockReplicaExports(replicaConfig *ReplicaConfig) bool {
	if replicaConfig.EnableResult {
		bc.ReplicaConfig.EnableResult = true
	}
	if replicaConfig.EnableSpecimen {
		bc.ReplicaConfig.EnableSpecimen = true
	}
	return true
}
