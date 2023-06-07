package sync_service

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
)

const (
	// DefaultFetchBlockRange is the number of blocks that we collect in a single eth_getLogs query.
	DefaultFetchBlockRange = uint64(100)

	// DefaultPollInterval is the frequency at which we query for new L1 messages.
	DefaultPollInterval = time.Second * 10

	// LogProgressInterval is the frequency at which we log progress.
	LogProgressInterval = time.Second * 10

	// DbWriteThresholdBytes is the size of batched database writes in bytes.
	DbWriteThresholdBytes = 10 * 1024

	// DbWriteThresholdBlocks is the number of blocks scanned after which we write to the database
	// even if we have not collected DbWriteThresholdBytes bytes of data yet. This way, if there is
	// a long section of L1 blocks with no messages and we stop or crash, we will not need to re-scan
	// this secion.
	DbWriteThresholdBlocks = 1000
)

// SyncService collects all L1 messages and stores them in a local database.
type SyncService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	client               *BridgeClient
	db                   ethdb.Database
	msgCountFeed         event.Feed
	pollInterval         time.Duration
	latestProcessedBlock uint64
	scope                event.SubscriptionScope
}

func NewSyncService(ctx context.Context, genesisConfig *params.ChainConfig, nodeConfig *node.Config, db ethdb.Database, l1Client EthClient) (*SyncService, error) {
	// terminate if the caller does not provide an L1 client (e.g. in tests)
	if l1Client == nil || (reflect.ValueOf(l1Client).Kind() == reflect.Ptr && reflect.ValueOf(l1Client).IsNil()) {
		log.Warn("No L1 client provided, L1 sync service will not run")
		return nil, nil
	}

	if genesisConfig.Scroll.L1Config == nil {
		return nil, fmt.Errorf("missing L1 config in genesis")
	}

	client, err := newBridgeClient(ctx, l1Client, genesisConfig.Scroll.L1Config.L1ChainId, nodeConfig.L1Confirmations, genesisConfig.Scroll.L1Config.L1MessageQueueAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize bridge client: %w", err)
	}

	// assume deployment block has 0 messages
	latestProcessedBlock := nodeConfig.L1DeploymentBlock
	block := rawdb.ReadSyncedL1BlockNumber(db)
	if block != nil {
		// restart from latest synced block number
		latestProcessedBlock = *block
	}

	ctx, cancel := context.WithCancel(ctx)

	service := SyncService{
		ctx:                  ctx,
		cancel:               cancel,
		client:               client,
		db:                   db,
		pollInterval:         DefaultPollInterval,
		latestProcessedBlock: latestProcessedBlock,
	}

	return &service, nil
}

func (s *SyncService) Start() {
	if s == nil {
		return
	}

	// wait for initial sync before starting node
	log.Info("Starting L1 message sync service", "latestProcessedBlock", s.latestProcessedBlock)

	// block node startup during initial sync and print some helpful logs
	latestConfirmed, err := s.client.getLatestConfirmedBlockNumber(s.ctx)
	if err == nil && latestConfirmed > s.latestProcessedBlock+1000 {
		log.Warn("Running initial sync of L1 messages before starting l2geth, this might take a while...")
		s.fetchMessages()
		log.Info("L1 message initial sync completed", "latestProcessedBlock", s.latestProcessedBlock)
	}

	go func() {
		t := time.NewTicker(s.pollInterval)
		defer t.Stop()

		for {
			// don't wait for ticker during startup
			s.fetchMessages()

			select {
			case <-s.ctx.Done():
				return
			case <-t.C:
				continue
			}
		}
	}()
}

func (s *SyncService) Stop() {
	if s == nil {
		return
	}

	log.Info("Stopping sync service")

	// Unsubscribe all subscriptions registered
	s.scope.Close()

	if s.cancel != nil {
		s.cancel()
	}
}

// SubscribeNewL1MsgsEvent registers a subscription of NewL1MsgsEvent and
// starts sending event to the given channel.
func (s *SyncService) SubscribeNewL1MsgsEvent(ch chan<- core.NewL1MsgsEvent) event.Subscription {
	return s.scope.Track(s.msgCountFeed.Subscribe(ch))
}

func (s *SyncService) fetchMessages() {
	latestConfirmed, err := s.client.getLatestConfirmedBlockNumber(s.ctx)
	if err != nil {
		log.Warn("Failed to get latest confirmed block number", "err", err)
		return
	}

	log.Trace("Sync service fetchMessages", "latestProcessedBlock", s.latestProcessedBlock, "latestConfirmed", latestConfirmed)

	batchWriter := s.db.NewBatch()
	numBlocksPendingDbWrite := uint64(0)
	numMessagesPendingDbWrite := 0

	// helper function to flush database writes cached in memory
	flush := func(lastBlock uint64) {
		// update sync progress
		rawdb.WriteSyncedL1BlockNumber(batchWriter, lastBlock)

		// write batch in a single transaction
		err := batchWriter.Write()
		if err != nil {
			// crash on database error, no risk of inconsistency here
			log.Crit("Failed to write L1 messages to database", "err", err)
		}

		batchWriter.Reset()
		numBlocksPendingDbWrite = 0

		if numMessagesPendingDbWrite > 0 {
			s.msgCountFeed.Send(core.NewL1MsgsEvent{Count: numMessagesPendingDbWrite})
			numMessagesPendingDbWrite = 0
		}

		s.latestProcessedBlock = lastBlock
	}

	// ticker for logging progress
	t := time.NewTicker(LogProgressInterval)
	numMsgsCollected := 0

	// query in batches
	for from := s.latestProcessedBlock + 1; from <= latestConfirmed; from += DefaultFetchBlockRange {
		select {
		case <-s.ctx.Done():
			// flush pending writes to database
			if from > 0 {
				flush(from - 1)
			}
			return
		case <-t.C:
			progress := 100 * float64(s.latestProcessedBlock) / float64(latestConfirmed)
			log.Info("Syncing L1 messages", "processed", s.latestProcessedBlock, "confirmed", latestConfirmed, "collected", numMsgsCollected, "progress(%)", progress)
		default:
		}

		to := from + DefaultFetchBlockRange - 1
		if to > latestConfirmed {
			to = latestConfirmed
		}

		msgs, err := s.client.fetchMessagesInRange(s.ctx, from, to)
		if err != nil {
			// flush pending writes to database
			if from > 0 {
				flush(from - 1)
			}
			log.Warn("Failed to fetch L1 messages in range", "fromBlock", from, "toBlock", to, "err", err)
			return
		}

		if len(msgs) > 0 {
			log.Debug("Received new L1 events", "fromBlock", from, "toBlock", to, "count", len(msgs))
			rawdb.WriteL1Messages(batchWriter, msgs) // collect messages in memory
			numMsgsCollected += len(msgs)
		}

		numBlocksPendingDbWrite += to - from
		numMessagesPendingDbWrite += len(msgs)

		// flush new messages to database periodically
		if to == latestConfirmed || batchWriter.ValueSize() >= DbWriteThresholdBytes || numBlocksPendingDbWrite >= DbWriteThresholdBlocks {
			flush(to)
		}
	}
}
