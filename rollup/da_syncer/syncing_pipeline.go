package da_syncer

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/common/backoff"
	"github.com/scroll-tech/go-ethereum/core"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/ethdb"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/blob_client"
	"github.com/scroll-tech/go-ethereum/rollup/da_syncer/serrors"
	"github.com/scroll-tech/go-ethereum/rollup/l1"
)

// Config is the configuration parameters of data availability syncing.
type Config struct {
	BlobScanAPIEndpoint    string // BlobScan blob api endpoint
	BlockNativeAPIEndpoint string // BlockNative blob api endpoint
	BeaconNodeAPIEndpoint  string // Beacon node api endpoint
	AwsS3BlobAPIEndpoint   string // AWS S3 blob data api endpoint

	RecoveryMode   bool   // Recovery mode is used to override existing blocks with the blocks read from the pipeline and start from a specific L1 block and batch
	InitialL1Block uint64 // L1 block in which the InitialBatch was committed (or any earlier L1 block but requires more RPC requests)
	InitialBatch   uint64 // Batch number from which to start syncing and overriding blocks
	SignBlocks     bool   // Whether to sign the blocks after reading them from the pipeline (requires correct Clique signer key) and history of blocks with Clique signatures
	L2EndBlock     uint64 // L2 block number to sync until

	ProduceBlocks bool // Whether to produce blocks in DA recovery mode. The pipeline will be disabled when starting the node with this flag.
}

// SyncingPipeline is a derivation pipeline for syncing data from L1 and DA and transform it into
// L2 blocks and chain.
type SyncingPipeline struct {
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	expBackoff *backoff.Exponential

	config Config

	db         ethdb.Database
	blockchain *core.BlockChain
	blockQueue *BlockQueue
	daSyncer   *DASyncer
	daQueue    *DAQueue
}

func NewSyncingPipeline(ctx context.Context, blockchain *core.BlockChain, genesisConfig *params.ChainConfig, db ethdb.Database, ethClient l1.Client, l1DeploymentBlock uint64, config Config) (*SyncingPipeline, error) {
	l1Reader, err := l1.NewReader(ctx, l1.Config{
		ScrollChainAddress:    genesisConfig.Scroll.L1Config.ScrollChainAddress,
		L1MessageQueueAddress: genesisConfig.Scroll.L1Config.L1MessageQueueAddress,
	}, ethClient)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize l1.Reader, err = %w", err)
	}

	blobClientList := blob_client.NewBlobClients()
	if config.BeaconNodeAPIEndpoint != "" {
		beaconNodeClient, err := blob_client.NewBeaconNodeClient(config.BeaconNodeAPIEndpoint)
		if err != nil {
			log.Warn("failed to create BeaconNodeClient", "err", err)
		} else {
			blobClientList.AddBlobClient(beaconNodeClient)
		}
	}
	if config.BlobScanAPIEndpoint != "" {
		blobClientList.AddBlobClient(blob_client.NewBlobScanClient(config.BlobScanAPIEndpoint))
	}
	if config.BlockNativeAPIEndpoint != "" {
		blobClientList.AddBlobClient(blob_client.NewBlockNativeClient(config.BlockNativeAPIEndpoint))
	}
	if config.AwsS3BlobAPIEndpoint != "" {
		blobClientList.AddBlobClient(blob_client.NewAwsS3Client(config.AwsS3BlobAPIEndpoint))
	}
	if blobClientList.Size() == 0 {
		return nil, errors.New("DA syncing is enabled but no blob client is configured. Please provide at least one blob client via command line flag")
	}

	dataSourceFactory := NewDataSourceFactory(blockchain, genesisConfig, config, l1Reader, blobClientList, db)
	var lastProcessedBatchMeta *rawdb.DAProcessedBatchMeta
	if config.RecoveryMode {
		if config.InitialL1Block == 0 {
			return nil, errors.New("sync from DA: initial L1 block must be set in recovery mode")
		}
		if config.InitialBatch == 0 {
			return nil, errors.New("sync from DA: initial batch must be set in recovery mode")
		}

		l1MessageQueueHeightFinder, err := NewL1MessageQueueHeightFinder(ctx, config.InitialL1Block, l1Reader, blobClientList, db)
		if err != nil {
			return nil, fmt.Errorf("failed to create L1MessageQueueHeightFinder: %w", err)
		}

		l1MessageQueueHeightBeforeInitialBatch, err := l1MessageQueueHeightFinder.TotalL1MessagesPoppedBefore(config.InitialBatch)
		if err != nil {
			return nil, fmt.Errorf("failed to find L1 message queue height before initial batch: %w", err)
		}

		lastProcessedBatchMeta = &rawdb.DAProcessedBatchMeta{
			BatchIndex:            config.InitialBatch,
			L1BlockNumber:         config.InitialL1Block,
			TotalL1MessagesPopped: l1MessageQueueHeightBeforeInitialBatch,
		}

		log.Info("sync from DA: initializing pipeline in recovery mode", "initialL1Block", config.InitialL1Block, "initialBatch", config.InitialBatch, "L1BlockNumber", lastProcessedBatchMeta.L1BlockNumber, "TotalL1MessagesPopped", lastProcessedBatchMeta.TotalL1MessagesPopped)
	} else {
		lastProcessedBatchMeta = rawdb.ReadDAProcessedBatchMeta(db)
		if lastProcessedBatchMeta == nil {
			var l1BlockNumber uint64
			if l1DeploymentBlock > 0 {
				l1BlockNumber = l1DeploymentBlock - 1
			}
			lastProcessedBatchMeta = &rawdb.DAProcessedBatchMeta{
				BatchIndex:            0,
				L1BlockNumber:         l1BlockNumber,
				TotalL1MessagesPopped: 0,
			}
			rawdb.WriteDAProcessedBatchMeta(db, lastProcessedBatchMeta)
		}
		log.Info("sync from DA: initializing pipeline", "BatchIndex", lastProcessedBatchMeta.BatchIndex, "L1BlockNumber", lastProcessedBatchMeta.L1BlockNumber, "TotalL1MessagesPopped", lastProcessedBatchMeta.TotalL1MessagesPopped)
	}

	daQueue := NewDAQueue(lastProcessedBatchMeta.L1BlockNumber, dataSourceFactory)
	batchQueue := NewBatchQueue(daQueue, db, lastProcessedBatchMeta)
	blockQueue := NewBlockQueue(batchQueue)
	daSyncer := NewDASyncer(blockchain, config.L2EndBlock)

	ctx, cancel := context.WithCancel(ctx)
	return &SyncingPipeline{
		ctx:        ctx,
		cancel:     cancel,
		expBackoff: backoff.NewExponential(100*time.Millisecond, 10*time.Second, 100*time.Millisecond),
		wg:         sync.WaitGroup{},
		config:     config,
		db:         db,
		blockchain: blockchain,
		blockQueue: blockQueue,
		daSyncer:   daSyncer,
		daQueue:    daQueue,
	}, nil
}

func (s *SyncingPipeline) Step() error {
	block, err := s.blockQueue.NextBlock(s.ctx)
	if err != nil {
		return err
	}

	// in recovery mode, we override already existing blocks with whatever we read from the pipeline
	err = s.daSyncer.SyncOneBlock(block, s.config.RecoveryMode, s.config.SignBlocks)

	return err
}

func (s *SyncingPipeline) Start() {
	log.Info("sync from DA: starting pipeline")

	s.wg.Add(1)
	go func() {
		s.mainLoop()
		s.wg.Done()
	}()
}

func (s *SyncingPipeline) mainLoop() {
	progressTicker := time.NewTicker(1 * time.Minute)
	defer progressTicker.Stop()

	stepCh := make(chan struct{}, 1)
	var delayedStepCh <-chan time.Time
	var resetCounter int
	var tempErrorCounter int

	// reqStep is a helper function to request a step to be executed.
	// If delay is true, it will request a delayed step with exponential backoff, otherwise it will request an immediate step.
	reqStep := func(delay bool) {
		if delay {
			if delayedStepCh == nil {
				delayDur := s.expBackoff.NextDuration()
				delayedStepCh = time.After(delayDur)
				log.Debug("requesting delayed step", "delay", delayDur, "attempt", s.expBackoff.Attempt())
			} else {
				log.Debug("ignoring step request because of ongoing delayed step", "attempt", s.expBackoff.Attempt())
			}
		} else {
			select {
			case stepCh <- struct{}{}:
			default:
			}
		}
	}

	// start pipeline
	reqStep(false)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		select {
		case <-s.ctx.Done():
			return
		case <-progressTicker.C:
			currentBlock := s.blockchain.CurrentBlock()
			dataSource := s.daQueue.DataSource()
			if dataSource != nil {
				log.Info("L1 sync progress", "L1 processed", dataSource.L1Height(), "L1 finalized", dataSource.L1Finalized(), "progress(%)", float64(dataSource.L1Height())/float64(dataSource.L1Finalized())*100, "L2 height", currentBlock.Number().Uint64(), "L2 hash", currentBlock.Hash().Hex())
			} else {
				log.Info("L1 sync progress", "blockchain height", currentBlock.Number().Uint64(), "block hash", currentBlock.Hash().Hex())
			}
		case <-delayedStepCh:
			delayedStepCh = nil
			reqStep(false)
		case <-stepCh:
			err := s.Step()
			if err == nil {
				// step succeeded, reset exponential backoff and continue
				reqStep(false)
				s.expBackoff.Reset()
				resetCounter = 0
				tempErrorCounter = 0
				continue
			}

			if errors.Is(err, serrors.EOFError) {
				// pipeline is empty, request a delayed step
				// TODO: eventually (with state manager) this should not trigger a delayed step because external events will trigger a new step anyway
				reqStep(true)
				log.Debug("syncing pipeline is empty, requesting delayed step")
				tempErrorCounter = 0
				continue
			} else if errors.Is(err, serrors.TemporaryError) {
				log.Warn("syncing pipeline step failed due to temporary error, retrying", "err", err)
				if tempErrorCounter > 100 {
					log.Warn("syncing pipeline step failed due to 100 consecutive temporary errors, stopping pipeline worker", "last err", err)
					return
				}

				// temporary error, request a delayed step
				reqStep(true)
				tempErrorCounter++
				continue
			} else if errors.Is(err, ErrBlockTooLow) {
				// block number returned by the block queue is too low,
				// we skip the blocks until we reach the correct block number again.
				reqStep(false)
				tempErrorCounter = 0
				continue
			} else if errors.Is(err, ErrBlockTooHigh) {
				// block number returned by the block queue is too high,
				// reset the pipeline and move backwards from the last L1 block we read
				s.reset(resetCounter)
				resetCounter++
				reqStep(false)
				tempErrorCounter = 0
				continue
			} else if errors.Is(err, context.Canceled) {
				log.Info("syncing pipeline stopped due to cancelled context", "err", err)
				return
			} else if errors.Is(err, serrors.Terminated) {
				log.Info("syncing pipeline stopped due to terminated state", "err", err)
				return
			}

			log.Warn("syncing pipeline step failed due to unrecoverable error, stopping pipeline worker", "err", err)
			return
		}
	}
}

func (s *SyncingPipeline) Stop() {
	log.Info("sync from DA: stopping pipeline...")
	s.cancel()
	s.wg.Wait()
	log.Info("sync from DA: stopping pipeline... done")
}

func (s *SyncingPipeline) reset(resetCounter int) {
	amount := 100 * uint64(resetCounter)

	lastProcessedBatchMeta := rawdb.ReadDAProcessedBatchMeta(s.db)
	if lastProcessedBatchMeta == nil {
		lastProcessedBatchMeta = &rawdb.DAProcessedBatchMeta{
			BatchIndex:            0,
			L1BlockNumber:         s.config.InitialL1Block - amount,
			TotalL1MessagesPopped: 0,
		}
	}

	if lastProcessedBatchMeta.L1BlockNumber > amount {
		lastProcessedBatchMeta.L1BlockNumber -= amount
		rawdb.WriteDAProcessedBatchMeta(s.db, lastProcessedBatchMeta)
	}

	log.Info("resetting syncing pipeline", "batch index", lastProcessedBatchMeta.BatchIndex, "L1BlockNumber", lastProcessedBatchMeta.L1BlockNumber, "TotalL1MessagesPopped", lastProcessedBatchMeta.TotalL1MessagesPopped)
	s.blockQueue.Reset(lastProcessedBatchMeta)
}
