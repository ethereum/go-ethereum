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

func (sds *StateDiffService) Loop(events chan core.ChainEvent) {
	for elem := range events {
		currentBlock := elem.Block
		parentHash := currentBlock.ParentHash()
		parentBlock := sds.BlockChain.GetBlockByHash(parentHash)

		stateDiffLocation, err := sds.Extractor.ExtractStateDiff(*parentBlock, *currentBlock)
		if err != nil {
			log.Error("Error extracting statediff", "block number", currentBlock.Number(), "error", err)
		} else {
			log.Info("Statediff extracted", "block number", currentBlock.Number(), "location", stateDiffLocation)
		}
	}
}

var eventsChannel chan core.ChainEvent

func (sds *StateDiffService) Start(server *p2p.Server) error {
	log.Info("Starting statediff service")
	eventsChannel := make(chan core.ChainEvent, 10)
	sds.BlockChain.SubscribeChainEvent(eventsChannel)
	go sds.Loop(eventsChannel)
	return nil
}

func (StateDiffService) Stop() error {
	log.Info("Stopping statediff service")
	close(eventsChannel)

	return nil
}
