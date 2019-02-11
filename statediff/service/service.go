package service

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/statediff"
	b "github.com/ethereum/go-ethereum/statediff/builder"
	e "github.com/ethereum/go-ethereum/statediff/extractor"
	p "github.com/ethereum/go-ethereum/statediff/publisher"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

type BlockChain interface {
	SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription
	GetBlockByHash(hash common.Hash) *types.Block
}

type StateDiffService struct {
	Builder    *b.Builder
	Extractor  e.Extractor
	BlockChain BlockChain
}

func NewStateDiffService(db ethdb.Database, blockChain *core.BlockChain, config statediff.Config) (*StateDiffService, error) {
	builder := b.NewBuilder(db)
	publisher, err := p.NewPublisher(config)
	if err != nil {
		return nil, err
	}

	extractor := e.NewExtractor(builder, publisher)
	return &StateDiffService{
		BlockChain: blockChain,
		Extractor:  extractor,
	}, nil
}

func (StateDiffService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{}
}

func (StateDiffService) APIs() []rpc.API {
	return []rpc.API{}
}

func (sds *StateDiffService) Loop(chainEventCh chan core.ChainEvent) {
	chainEventSub := sds.BlockChain.SubscribeChainEvent(chainEventCh)
	defer chainEventSub.Unsubscribe()

	blocksCh := make(chan *types.Block, 10)
	errCh := chainEventSub.Err()
	quitCh := make(chan struct{})

	go func() {
	HandleChainEventChLoop:
		for {
			select {
			//Notify chain event channel of events
			case chainEvent := <-chainEventCh:
				log.Debug("Event received from chainEventCh", "event", chainEvent)
				blocksCh <- chainEvent.Block
			//if node stopped
			case err := <-errCh:
				log.Warn("Error from chain event subscription, breaking loop.", "error", err)
				break HandleChainEventChLoop
			}
		}
		close(quitCh)
	}()

	//loop through chain events until no more
HandleBlockChLoop:
	for {
		select {
		case block := <-blocksCh:
			currentBlock := block
			parentHash := currentBlock.ParentHash()
			parentBlock := sds.BlockChain.GetBlockByHash(parentHash)
			if parentBlock == nil {
				log.Error("Parent block is nil, skipping this block",
					"parent block hash", parentHash.String(),
					"current block number", currentBlock.Number())
				break HandleBlockChLoop
			}

			stateDiffLocation, err := sds.Extractor.ExtractStateDiff(*parentBlock, *currentBlock)
			if err != nil {
				log.Error("Error extracting statediff", "block number", currentBlock.Number(), "error", err)
			} else {
				log.Info("Statediff extracted", "block number", currentBlock.Number(), "location", stateDiffLocation)
			}
		case <-quitCh:
			log.Debug("Quitting the statediff block channel")
			return
		}
	}
}

func (sds *StateDiffService) Start(server *p2p.Server) error {
	log.Info("Starting statediff service")

	chainEventCh := make(chan core.ChainEvent, 10)
	go sds.Loop(chainEventCh)

	return nil
}

func (StateDiffService) Stop() error {
	log.Info("Stopping statediff service")
	return nil
}
