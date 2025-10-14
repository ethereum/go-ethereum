package sync_service

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/event"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/node"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rlp"
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

var (
	l1MessageTotalCounter = metrics.NewRegisteredCounter("rollup/l1/message", nil)
)

// SyncService collects all L1 messages and stores them in a local database.
type SyncService struct {
	ctx                  context.Context
	cancel               context.CancelFunc
	client               *BridgeClient
	db                   ethdb.Database
	msgCountFeed         event.Feed
	pollInterval         time.Duration
	fetchBlockRange      uint64
	latestProcessedBlock uint64
	scope                event.SubscriptionScope
	stateMu              sync.Mutex
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

	client, err := newBridgeClient(ctx, l1Client, genesisConfig.Scroll.L1Config.L1ChainId, nodeConfig.L1Confirmations, genesisConfig.Scroll.L1Config.L1MessageQueueAddress, !nodeConfig.L1DisableMessageQueueV2, genesisConfig.Scroll.L1Config.L1MessageQueueV2Address)
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

	// reset synced height so that previous V2 messages are re-fetched in case a node upgraded after V2 deployment.
	// otherwise there's no way for the node to know if it missed any messages of the V2 queue (as it was not querying it before)
	// but continued to query the V1 queue (which after V2 deployment does not contain any messages anymore).
	// this is a one-time operation and will not be repeated on subsequent restarts.
	if genesisConfig.Scroll.L1Config.L1MessageQueueV2DeploymentBlock > 0 &&
		genesisConfig.Scroll.L1Config.L1MessageQueueV2DeploymentBlock < latestProcessedBlock { // node synced after V2 deployment

		// this means the node has never synced V2 messages before -> we need to reset the synced height to re-fetch V2 messages.
		// Resetting the synced height will not cause any inconsistency as V1 messages are only available before V2 deployment block
		// and V2 messages are only available after V2 deployment block. -> rawdb.ReadHighestSyncedQueueIndex(s.db) and the next expected index
		// will still be consistent.
		initialV2L1Block := rawdb.ReadL1MessageV2FirstL1BlockNumber(db)
		if initialV2L1Block == nil {
			latestProcessedBlock = genesisConfig.Scroll.L1Config.L1MessageQueueV2DeploymentBlock
			log.Info("Resetting L1 message sync height to fetch previous V2 messages", "L1 block", latestProcessedBlock)
		}
	}

	ctx, cancel := context.WithCancel(ctx)

	pollInterval := nodeConfig.L1SyncInterval
	if pollInterval == 0 {
		pollInterval = DefaultPollInterval
	}

	fetchBlockRange := nodeConfig.L1FetchBlockRange
	if fetchBlockRange == 0 {
		fetchBlockRange = DefaultFetchBlockRange
	}

	service := SyncService{
		ctx:                  ctx,
		cancel:               cancel,
		client:               client,
		db:                   db,
		pollInterval:         pollInterval,
		fetchBlockRange:      fetchBlockRange,
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

// ResetStartSyncHeight resets the SyncService to a specific L1 block height
func (s *SyncService) ResetStartSyncHeight(height uint64) {
	if s == nil {
		return
	}

	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	s.latestProcessedBlock = height
	log.Info("Reset sync service", "height", height)
}

// SubscribeNewL1MsgsEvent registers a subscription of NewL1MsgsEvent and
// starts sending event to the given channel.
func (s *SyncService) SubscribeNewL1MsgsEvent(ch chan<- core.NewL1MsgsEvent) event.Subscription {
	return s.scope.Track(s.msgCountFeed.Subscribe(ch))
}

func (s *SyncService) fetchMessages() {
	s.stateMu.Lock()
	defer s.stateMu.Unlock()

	latestConfirmed, err := s.client.getLatestConfirmedBlockNumber(s.ctx)
	if err != nil {
		log.Warn("Failed to get latest confirmed block number", "err", err)
		return
	}

	log.Trace("Sync service fetchMessages", "latestProcessedBlock", s.latestProcessedBlock, "latestConfirmed", latestConfirmed)

	// keep track of next queue index we're expecting to see
	queueIndex := rawdb.ReadHighestSyncedQueueIndex(s.db)

	// read start index of very first L1MessageV2 from database
	l1MessageV2StartIndex := rawdb.ReadL1MessageV2StartIndex(s.db)

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
			l1MessageTotalCounter.Inc(int64(numMessagesPendingDbWrite))
			s.msgCountFeed.Send(core.NewL1MsgsEvent{Count: numMessagesPendingDbWrite})
			numMessagesPendingDbWrite = 0
		}

		s.latestProcessedBlock = lastBlock
	}

	// ticker for logging progress
	t := time.NewTicker(LogProgressInterval)
	numMsgsCollected := 0

	// query in batches
	for from := s.latestProcessedBlock + 1; from <= latestConfirmed; from += s.fetchBlockRange {
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

		to := from + s.fetchBlockRange - 1
		if to > latestConfirmed {
			to = latestConfirmed
		}

		queryL1MessagesV1 := l1MessageV2StartIndex == nil
		msgsV1, msgsV2, err := s.client.fetchMessagesInRange(s.ctx, from, to, queryL1MessagesV1)
		if err != nil {
			// flush pending writes to database
			if from > 0 {
				flush(from - 1)
			}
			log.Warn("Failed to fetch L1 messages in range", "fromBlock", from, "toBlock", to, "err", err)
			return
		}

		// write start index of very first L1MessageV2 to database. This is true only once.
		if len(msgsV2) > 0 && l1MessageV2StartIndex == nil {
			firstL1MessageV2 := msgsV2[0]
			log.Info("Received first L1Message from MessageQueueV2", "queueIndex", firstL1MessageV2.QueueIndex, "L1 blockNumber", to)
			l1MessageV2StartIndex = &firstL1MessageV2.QueueIndex
			rawdb.WriteL1MessageV2StartIndex(batchWriter, firstL1MessageV2.QueueIndex)
			rawdb.WriteL1MessageV2FirstL1BlockNumber(batchWriter, to)
		}

		msgs := append(msgsV1, msgsV2...)

		if len(msgs) > 0 {
			log.Debug("Received new L1 events", "fromBlock", from, "toBlock", to, "count", len(msgs))
		}

		for _, msg := range msgs {
			if msg.QueueIndex > 0 {
				queueIndex++
			}

			// check if received queue index matches expected queue index
			if msg.QueueIndex > queueIndex {
				log.Error("Unexpected queue index in SyncService", "expected", queueIndex, "got", msg.QueueIndex, "msg", msg)
				return // do not flush inconsistent data to disk
			}

			// compare with stored message in database, abort if not equal, ignore if already exists
			if msg.QueueIndex < queueIndex {
				log.Warn("Duplicate queue index in SyncService", "expected", queueIndex, "got", msg.QueueIndex)

				receivedMsgBytes, err := rlp.EncodeToBytes(msg)
				if err != nil {
					log.Error("Failed to encode message", "err", err)
					return
				}
				storedMsgBytes := rawdb.ReadL1MessageRLP(s.db, msg.QueueIndex)
				if !bytes.Equal(storedMsgBytes, receivedMsgBytes) {
					storedL1Message := rawdb.ReadL1Message(s.db, msg.QueueIndex)
					log.Error("Stored message at same queue index does not match received message", "queueIndex", msg.QueueIndex, "expected", storedL1Message, "got", msg)
					return
				}

				// already exists, ignore
				queueIndex--
				continue
			}

			// store message to database (collected in memory and flushed periodically)
			rawdb.WriteL1Message(batchWriter, msg)
			numMsgsCollected++
		}

		numBlocksPendingDbWrite += to - from + 1
		numMessagesPendingDbWrite += len(msgs)

		// flush new messages to database periodically
		if to == latestConfirmed || batchWriter.ValueSize() >= DbWriteThresholdBytes || numBlocksPendingDbWrite >= DbWriteThresholdBlocks {
			flush(to)
		}
	}
}
