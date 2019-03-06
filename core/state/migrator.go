package state

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
	"golang.org/x/sync/errgroup"
)

// A Migrator is an entity that copies over account state at a particular snapshot
// from a source database to a destination database. Each Migrator instance's
// lifetime only lasts for a single migration.
//
// A Migrator is meant to be invoked as follows:
//
//    rootHash := common.Hash{}
// 		srcDB := ethdb.NewLDBDatabase("existing", /* cache */ 1024, /* handles */ 1024)
// 		dstDB := ethdb.NewLDBDatabase("migrated", /* cache */ 1024, /* handles */ 1024)
// 		m := NewStateMigrator(dstDB, srcDB, rootHash, /* numWorkers */ 1, /* batchSize */ 1)
// 		m.Start()
// 		if err := m.Wait(); err != nil {
//				// handle error
// .	}
//
// A typical state snapshot contains such a large amount of data that it can be prohibitively
// expensive to perform if implemented naively. As a result, the Migrator has taken on the following
// architecture below, which will be described at a high-level. It is worth noting that this
// framework has been specifically designed for LevelDB-based databases. More information can
// be found by looking at each component's corresponding struct. The components are managed as a
// a set of go-routines (boxes) connected by channels (arrows) operated within an errgroup.Group.
//
//                                unprocessed
//      ------------------------------------------------------------------
//      |                                                                |
//      v                                                                |
// -----------  unsorted   ----------  sorted  ----------  results   ----------  unsaved  ----------
// | batcher |-----------> | sorter |--------->| getter |----------->| fanout |---------->| putter |
// -----------             ----------          ----------            ----------           ----------
//
// Components:
//   - batcher: schedules and aggregates a group of database keys to look up in the source database
//   - sorter: sorts the keys to make the accesses to LevelDB more efficient
//   - getter: performs lookups in source database
//   - fanout: sends retrieved key-value pairs to multiple consumers
//   - putter: stores key-value pairs in destination database
//
type Migrator struct {
	group *errgroup.Group

	resultFanout *resultFanout
	batcher      *batcher
	sorter       *sorter
	getter       *getter
	putter       *putter

	numWorkers int
}

// NewMigrator returns a new Migrator instance that migrates account information from the
// source database srcDB to the destination database dstDB for the corresponding account state at
// the specified rootHash. Additionally the number of workers for getting/putting data from the two
// databases numWorkers is specified (where numWorkers are individually used for getting and
// putting) and the maximum number of items to migrate at once batchSize.
func NewMigrator(dstDB ethdb.Database, srcDB trie.DatabaseReader, rootHash common.Hash, numWorkers, batchSize int) *Migrator {
	// Channel sizes of one allow one batch to be buffered so that the batcher, sorter, and getter
	// do not block one another.
	unsorted := make(chan []common.Hash, 1)
	sorted := make(chan []common.Hash, 1)
	// Channel sizes of numWorkers allow the getter, resultFanout, and putter to not block.
	results := make(chan []trie.SyncResult, numWorkers)
	unprocessed := make(chan []trie.SyncResult, numWorkers)
	unsaved := make(chan []trie.SyncResult, numWorkers)
	// Round up to ensure that chunkSize is at least 1 when numWorkers > batchSize.
	chunkSize := (batchSize + numWorkers - 1) / numWorkers
	// The errgroup is used to manage all of the processing components (including workers).
	group, ctx := errgroup.WithContext(context.Background())

	return &Migrator{
		group:        group,
		resultFanout: newResultFanout(ctx, results, unprocessed, unsaved),
		batcher:      newBatcher(ctx, rootHash, unsorted, unprocessed, batchSize),
		sorter:       newSorter(ctx, unsorted, sorted),
		getter:       newGetter(ctx, srcDB, sorted, results, chunkSize),
		putter:       newPutter(ctx, dstDB, unsaved),
		numWorkers:   numWorkers,
	}
}

// Wait blocks until the state data migration has finished or encountered an error.
func (m *Migrator) Wait() error {
	return m.group.Wait()
}

// Start begins the state data migration process.
func (m *Migrator) Start() {
	m.group.Go(m.batcher.run)
	m.group.Go(m.sorter.run)
	m.group.Go(m.resultFanout.run)
	m.group.Go(m.getter.runManager)
	for i := 0; i < m.numWorkers; i++ {
		m.group.Go(m.getter.runWorker)
		m.group.Go(m.putter.runWorker)
	}
}

// A resultFanout broadcasts []*trie.SyncResult batches to consumers.
//
//                            consumer[0]    ----------
//                      -------------------> | worker |
//                      |                    ----------
//                      |
//                 ---------- consumer[1]    ----------
//  producer ----->| fanout |--------------> | worker |
//                 ----------                ----------
//                      |
//                      |     consumer[2]    ----------
//                      -------------------> | worker |
//                                           ----------
type resultFanout struct {
	ctx       context.Context
	producer  <-chan []trie.SyncResult
	consumers []chan<- []trie.SyncResult
}

// newResultFanout returns a new resultFanout instance monitoring the context ctx, ingesting
// results from producer and then broadcasting them out to consumers.
//
// Each of the consumers will be closed when the resultFanout has completed, which occurs either
// by having its incoming producer channel closed or being signaled via context.
func newResultFanout(ctx context.Context, producer <-chan []trie.SyncResult, consumers ...chan<- []trie.SyncResult) *resultFanout {
	return &resultFanout{
		ctx,
		producer,
		consumers,
	}
}

func (f *resultFanout) run() error {
	defer f.closeConsumers()

	for results := range f.producer {
		for _, consumer := range f.consumers {
			select {
			case <-f.ctx.Done():
				return nil
			case consumer <- results:
			}
		}
	}
	return nil
}

func (f *resultFanout) closeConsumers() {
	for _, c := range f.consumers {
		close(c)
	}
}

// A batcher schedules and groups hashes together for other components to operate upon as a unit
// of work.
type batcher struct {
	ctx context.Context

	sched *trie.Sync

	maxBatchSize int

	reqs  chan<- []common.Hash
	resps <-chan []trie.SyncResult

	reqsInflight int
	queue        []common.Hash
	putter       *droppingPutter
}

// newBatcher returns a new batcher instance monitoring the context ctx, starting to schedule hashes
// to look up from the rootHash of the account state trie, sending batches up to size maxBatchSize
// through reqs and then creating new batches based on the results received through resps.
//
// resps will be closed when the batcher has completed, which occurs either by running out of hashes
// to retrieve or being signaled via context.
func newBatcher(ctx context.Context, rootHash common.Hash, reqs chan<- []common.Hash, resps <-chan []trie.SyncResult, maxBatchSize int) *batcher {
	return &batcher{
		ctx:          ctx,
		sched:        NewStateSync(rootHash, &emptyTrieReader{}),
		reqs:         reqs,
		resps:        resps,
		maxBatchSize: maxBatchSize,
		putter:       &droppingPutter{},
	}
}

// A droppingPutter is an implementation of ethdb.Putter that discards the values inserted into it.
// The key-value pairs are placed into the destination database by the putter component instead.
//
// The sync.Trie logic uses a ethdb.Putter to flush requests that have been completed. The migrator
// does this independently within the putter component for two reasons:
//   1. Performance: the putter component has been designed to take large, key-sorted batches whereas
//      the trie.sync logic inserts results one-at-time in the order they were requested.
//   2. Memory efficiency: the trie.Sync logic holds on to completed requests
//      that have not been stored until trie.Sync.Commit has been called. Unlike the original
//      peer-based sync use case, there can be millions of requests needing to be stored at any
//      given time which can lead to holding onto their corresponding allocated memory for too longs.
type droppingPutter struct {
}

func (*droppingPutter) Put(key []byte, value []byte) error {
	// Does nothing.
	return nil
}

// An emptyTrieReader is an implementation of a trie.DatabaseReader that does not contain any values
// to read.
//
// trie.Sync uses a trie.Database reader to check whether or not a value being retrieved is already
// present in the destination database (which is useful for syncing from peers that can be cancelled
// and restarted). However, the Migrator performs the copy within a single invocation so this check
// does not add value and in fact incurs a performance penalty as database reads can be expensive.
type emptyTrieReader struct {
}

func (*emptyTrieReader) Get(key []byte) (value []byte, err error) {
	return nil, nil
}

func (*emptyTrieReader) Has(key []byte) (bool, error) {
	return false, nil
}

func (b *batcher) run() error {
	defer close(b.reqs)

	// Send root hash.
	b.fillQueue()
	b.sendNextBatch()

	for resps := range b.resps {
		b.processResponses(resps)
		b.fillQueue()

		if b.noRequestsPending() {
			return nil
		}

		if b.shouldSendNextBatch() {
			if !b.sendNextBatch() {
				return nil
			}
		}
	}

	return nil
}

func (b *batcher) processResponses(resps []trie.SyncResult) {
	b.sched.Process(resps)
	b.sched.Commit(b.putter)
	b.reqsInflight -= len(resps)
}

func (b *batcher) sendNextBatch() bool {
	select {
	case <-b.ctx.Done():
		return false
	case b.reqs <- b.queue:
		b.reqsInflight += len(b.queue)
		b.queue = nil
		return true
	}
}

func (b *batcher) shouldSendNextBatch() bool {
	return b.isIdle() || b.isUnderCapacity()
}

func (b *batcher) isIdle() bool {
	return b.reqsInflight == 0
}

// The batcher system only allows 2 * b.maxBatchsize requests to be inflight at a given time
// in order to not overwhelm the system. At the same time, the system wants to batch up enough
// requests to amortize overheads and exploit sequential locality between request keys.
//
// Not that deadlock is prevented in the event len(b.queue) < b.maxBatchSize, as the system will
// eventually become idle as detected by batcher.isIdle, thus allowing the next batch to be sent
// when batcher.shouldSendNextBatch is invoked (from receiving the batch that results in idleness).
func (b *batcher) isUnderCapacity() bool {
	return len(b.queue) >= b.maxBatchSize && b.reqsInflight+len(b.queue) < 2*b.maxBatchSize
}

func (b *batcher) noRequestsPending() bool {
	return b.sched.Pending() == 0
}

func (b *batcher) fillQueue() {
	if len(b.queue) < b.maxBatchSize {
		b.queue = append(b.queue, b.sched.Missing(b.maxBatchSize-len(b.queue))...)
	}
}

// A sorter sorts lists of hashes.
type sorter struct {
	ctx context.Context

	unsorted <-chan []common.Hash
	sorted   chan<- []common.Hash
}

// newSorter returns a new sorter instance monitoring the context ctx, taking groups of hashes from
// unsorted and then outputting them to sorted once the sort is complete.
//
// sorted will be closed when the sort has completed, which occurs either by running out of hashes to
// sort or being signaled via context.
func newSorter(ctx context.Context, unsorted <-chan []common.Hash, sorted chan<- []common.Hash) *sorter {
	return &sorter{
		ctx:      ctx,
		unsorted: unsorted,
		sorted:   sorted,
	}
}

func (s *sorter) run() error {
	defer close(s.sorted)

	for data := range s.unsorted {
		sort.Sort(keys(data))
		select {
		case <-s.ctx.Done():
			return nil
		case s.sorted <- data:
		}
	}

	return nil
}

type keys []common.Hash

func (b keys) Len() int {
	return len(b)
}

func (b keys) Less(i, j int) bool {
	return bytes.Compare(b[i].Bytes(), b[j].Bytes()) < 0
}

func (b keys) Swap(i, j int) {
	b[j], b[i] = b[i], b[j]
}

// A getter retrieves values from an underlying database.
//
// The getter internally uses workers to concurrently retrieve values within a batch sent to it.
// Requests are sent to a manager which then breaks the batch into smaller chunks that are then
// distributed to workers within a pool to perform the actual retrievals.
//
//             ----------------------------------------------------
//             |  getter                                          |
//             |                               ----------         |
//             |                         ----> | worker |----     |
//             |                         |     ----------   |     |
//             |                         |                  |     |
//             |    -----------  chunks  |     ----------   |     |
//  hashes ----|--->| manager |--------------> | worker |---------|---> results
//             |    -----------          |     ----------   |     |
//             |                         |                  |     |
//             |                         |     ----------   |     |
//             |                         ----> | worker |----     |
//             |                               ----------         |
//             |                                                  |
//             ----------------------------------------------------
//
type getter struct {
	ctx context.Context

	db trie.DatabaseReader

	hashes  <-chan []common.Hash
	results chan<- []trie.SyncResult
	chunks  chan []common.Hash

	chunkSize int

	closeChannels sync.Once
}

// newGetter returns a new getter instance monitoring the context ctx, taking groups of keys from
// hashes and the resulting key-value pairs found in db through results. The manager sends chunks
// of maximum size chunkSize to its workers.
//
// results will be closed when the getter has completed, which occurs either by running out of
// hashes to look up or being signaled via context.
func newGetter(ctx context.Context, db trie.DatabaseReader, hashes <-chan []common.Hash, results chan<- []trie.SyncResult, chunkSize int) *getter {
	return &getter{
		ctx:       ctx,
		db:        db,
		hashes:    hashes,
		results:   results,
		chunks:    make(chan []common.Hash),
		chunkSize: chunkSize,
	}
}

func (g *getter) runManager() error {
	defer g.closeOutboundChannels()

	for hashes := range g.hashes {
		for _, chunk := range g.splitIntoChunks(hashes) {
			select {
			case <-g.ctx.Done():
				return nil
			case g.chunks <- chunk:
			}
		}
	}

	return nil
}

// closeOutboundChannels closes the channels that output data from the getter (as shown in the
// block diagram above). It can be called by either the getter manager or workers as there are
// situations where either could halt execution of the getter (and Migrator as a whole). This is
// performed implicitly by closing channels. If a worker fails it will close the channel,
// otherwise the manager will close the channel. This method uses a sync.Once to
// ensure that multiple closes are not applied to the outbound channels.
func (g *getter) closeOutboundChannels() {
	g.closeChannels.Do(func() {
		close(g.chunks)
		g.chunks = nil
		close(g.results)
		g.chunks = nil
	})
}

func (g *getter) splitIntoChunks(hashes []common.Hash) [][]common.Hash {
	var chunks [][]common.Hash
	for len(hashes) > g.chunkSize {
		hashes, chunks = hashes[g.chunkSize:], append(chunks, hashes[:g.chunkSize])
	}
	chunks = append(chunks, hashes)
	return chunks
}

func (g *getter) runWorker() error {
	defer g.closeOutboundChannels()

	for hashes := range g.chunks {
		var results []trie.SyncResult
		for _, hash := range hashes {
			data, err := g.db.Get(hash.Bytes())
			if err != nil {
				return fmt.Errorf("error retrieving %s from database: %s", hex.EncodeToString(hash.Bytes()), err.Error())
			}

			result := trie.SyncResult{Hash: hash, Data: data}
			results = append(results, result)
		}

		select {
		case <-g.ctx.Done():
			return nil
		case g.results <- results:
		}
	}

	return nil
}

// A putter stores key-value pairs into an underlying database.
//
//              ------------------------
//              |  putter              |
//              |         ----------   |
//              |   ----> | worker |   |
//              |   |     ----------   |
//              |   |                  |
//              |   |     ----------   |
//  results ----|-------> | worker |   |
//              |   |     ----------   |
//              |   |                  |
//              |   |     ----------   |
//              |   ----> | worker |   |
//              |         ----------   |
//              |                      |
//              ------------------------
//
type putter struct {
	ctx context.Context

	db ethdb.Database

	results <-chan []trie.SyncResult
}

// newPutter returns a new putter instance monitoring the context ctx, taking groups of key-value
// pairs from results to store in db.
func newPutter(ctx context.Context, db ethdb.Database, results <-chan []trie.SyncResult) *putter {
	return &putter{
		ctx:     ctx,
		db:      db,
		results: results,
	}
}

func (p *putter) runWorker() error {
	for batch := range p.results {
		writeBatch := p.db.NewBatch()
		for _, r := range batch {
			if err := writeBatch.Put(r.Hash.Bytes(), r.Data); err != nil {
				return fmt.Errorf("error inserting pair (%s, %s) to batch: %s", hex.EncodeToString(r.Hash.Bytes()), hex.EncodeToString(r.Data), err.Error())
			}
		}
		if err := writeBatch.Write(); err != nil {
			return fmt.Errorf("error batch into database: %s", err.Error())
		}
	}
	return nil
}
