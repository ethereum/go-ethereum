package eccpow

import (
	"encoding/binary"
	"hash"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/cryptoecc/ETH-ECC/consensus"
	"github.com/cryptoecc/ETH-ECC/core/types"
	"github.com/cryptoecc/ETH-ECC/crypto"
	"github.com/cryptoecc/ETH-ECC/log"
	"github.com/cryptoecc/ETH-ECC/metrics"
	"github.com/cryptoecc/ETH-ECC/rpc"
	"golang.org/x/crypto/sha3"
)

type ECC struct {
	config Config

	// Mining related fields
	rand     *rand.Rand    // Properly seeded random source for nonces
	threads  int           // Number of threads to mine on if mining
	update   chan struct{} // Notification channel to update mining parameters
	hashrate metrics.Meter // Meter tracking the average hashrate
	remote   *remoteSealer

	// Remote sealer related fields
	workCh       chan *sealTask   // Notification channel to push new work and relative result channel to remote sealer
	fetchWorkCh  chan *sealWork   // Channel used for remote sealer to fetch mining work
	submitWorkCh chan *mineResult // Channel used for remote sealer to submit their mining result
	fetchRateCh  chan chan uint64 // Channel used to gather submitted hash rate for local or remote sealer.
	submitRateCh chan *hashrate   // Channel used for remote sealer to submit their mining hashrate

	shared    *ECC          // Shared PoW verifier to avoid cache regeneration
	fakeFail  uint64        // Block number which fails PoW check even in fake mode
	fakeDelay time.Duration // Time delay to sleep for before returning from verify

	lock      sync.Mutex // Ensures thread safety for the in-memory caches and mining fields
	closeOnce sync.Once  // Ensures exit channel will not be closed twice.
}

type Mode uint

const (
	epochLength      = 30000 // Blocks per epoch
	ModeNormal  Mode = iota
	ModeShared
	ModeTest
	ModeFake
	ModeFullFake
)

// Config are the configuration parameters of the ethash.
type Config struct {
	PowMode Mode
	// When set, notifications sent by the remote sealer will
	// be block header JSON objects instead of work package arrays.
	NotifyFull bool
	Log        log.Logger `toml:"-"`
}

// hasher is a repetitive hasher allowing the same hash data structures to be
// reused between hash runs instead of requiring new ones to be created.
//var hasher func(dest []byte, data []byte)

var (
	two256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

	// sharedECC is a full instance that can be shared between multiple users.
	sharedECC *ECC

	// algorithmRevision is the data structure version used for file naming.
	algorithmRevision = 2
)

func init() {
	sharedConfig := Config{
		PowMode: ModeNormal,
	}
	sharedECC = New(sharedConfig, nil, false)
}

type verifyParameters struct {
	n          uint64
	m          uint64
	wc         uint64
	wr         uint64
	seed       uint64
	outputWord []uint64
}

//const cross_err = 0.01

//type (
//	intMatrix   [][]int
//	floatMatrix [][]float64
//)

//RunOptimizedConcurrencyLDPC use goroutine for mining block
func RunOptimizedConcurrencyLDPC(header *types.Header, hash []byte) (bool, []int, []int, uint64, []byte) {
	//Need to set difficulty before running LDPC
	// Number of goroutines : 500, Number of attempts : 50000 Not bad

	var LDPCNonce uint64
	var hashVector []int
	var outputWord []int
	var digest []byte
	var flag bool

	//var wg sync.WaitGroup
	//var outerLoopSignal = make(chan struct{})
	//var innerLoopSignal = make(chan struct{})
	//var goRoutineSignal = make(chan struct{})

	parameters, _ := setParameters(header)
	H := generateH(parameters)
	colInRow, rowInCol := generateQ(parameters, H)

	for i := 0; i < 64; i++ {
		var goRoutineHashVector []int
		var goRoutineOutputWord []int
		goRoutineNonce := generateRandomNonce()
		seed := make([]byte, 40)
		copy(seed, hash)
		binary.LittleEndian.PutUint64(seed[32:], goRoutineNonce)
		seed = crypto.Keccak512(seed)
		//fmt.Printf("nonce: %v\n", seed)

		goRoutineHashVector = generateHv(parameters, seed)
		goRoutineHashVector, goRoutineOutputWord, _ = OptimizedDecoding(parameters, goRoutineHashVector, H, rowInCol, colInRow)

		flag, _ = MakeDecision(header, colInRow, goRoutineOutputWord)

		if flag {
			hashVector = goRoutineHashVector
			outputWord = goRoutineOutputWord
			LDPCNonce = goRoutineNonce
			digest = seed
			break
		}
	}
	return flag, hashVector, outputWord, LDPCNonce, digest
}

func RunOptimizedConcurrencyLDPC_Seoul(header *types.Header, hash []byte) (bool, []int, []int, uint64, []byte) {
	//Need to set difficulty before running LDPC
	// Number of goroutines : 500, Number of attempts : 50000 Not bad

	var LDPCNonce uint64
	var hashVector []int
	var outputWord []int
	var digest []byte
	var flag bool

	//var wg sync.WaitGroup
	//var outerLoopSignal = make(chan struct{})
	//var innerLoopSignal = make(chan struct{})
	//var goRoutineSignal = make(chan struct{})

	parameters, _ := setParameters_Seoul(header)
	H := generateH(parameters)
	colInRow, rowInCol := generateQ(parameters, H)

	for i := 0; i < 64; i++ {
		var goRoutineHashVector []int
		var goRoutineOutputWord []int
		goRoutineNonce := generateRandomNonce()
		seed := make([]byte, 40)
		copy(seed, hash)
		binary.LittleEndian.PutUint64(seed[32:], goRoutineNonce)
		seed = crypto.Keccak512(seed)
		//fmt.Printf("nonce: %v\n", seed)

		goRoutineHashVector = generateHv(parameters, seed)
		goRoutineHashVector, goRoutineOutputWord, _ = OptimizedDecodingSeoul(parameters, goRoutineHashVector, H, rowInCol, colInRow)

		flag, _ = MakeDecision_Seoul(header, colInRow, goRoutineOutputWord)

		if flag {
			hashVector = goRoutineHashVector
			outputWord = goRoutineOutputWord
			LDPCNonce = goRoutineNonce
			digest = seed
			break
		}
	}
	return flag, hashVector, outputWord, LDPCNonce, digest
}

//MakeDecision check outputWord is valid or not using colInRow
func MakeDecision(header *types.Header, colInRow [][]int, outputWord []int) (bool, int) {
	parameters, difficultyLevel := setParameters(header)
	for i := 0; i < parameters.m; i++ {
		sum := 0
		for j := 0; j < parameters.wr; j++ {
			//	fmt.Printf("i : %d, j : %d, m : %d, wr : %d \n", i, j, m, wr)
			sum = sum + outputWord[colInRow[j][i]]
		}
		if sum%2 == 1 {
			return false, -1
		}
	}

	var numOfOnes int
	for _, val := range outputWord {
		numOfOnes += val
	}

	if numOfOnes >= Table[difficultyLevel].decisionFrom &&
		numOfOnes <= Table[difficultyLevel].decisionTo &&
		numOfOnes%Table[difficultyLevel].decisionStep == 0 {
		//fmt.Printf("hamming weight: %v\n", numOfOnes)
		return true, numOfOnes
	}

	return false, numOfOnes
}

//MakeDecision check outputWord is valid or not using colInRow
func MakeDecision_Seoul(header *types.Header, colInRow [][]int, outputWord []int) (bool, int) {
	parameters, _ := setParameters_Seoul(header)
	for i := 0; i < parameters.m; i++ {
		sum := 0
		for j := 0; j < parameters.wr; j++ {
			//	fmt.Printf("i : %d, j : %d, m : %d, wr : %d \n", i, j, m, wr)
			sum = sum + outputWord[colInRow[j][i]]
		}
		if sum%2 == 1 {
			return false, -1
		}
	}

	var numOfOnes int
	for _, val := range outputWord {
		numOfOnes += val
	}

	if numOfOnes >= parameters.n/4  &&
		numOfOnes <= parameters.n/4 * 3 {
		//fmt.Printf("hamming weight: %v\n", numOfOnes)
		return true, numOfOnes
	}

	return false, numOfOnes
}

//func isRegular(nSize, wCol, wRow int) bool {
//	res := float64(nSize*wCol) / float64(wRow)
//	m := math.Round(res)
//
//	if int(m)*wRow == nSize*wCol {
//		return true
//	}
//
//	return false
//}

//func SetDifficulty(nSize, wCol, wRow int) bool {
//	if isRegular(nSize, wCol, wRow) {
//		n = nSize
//		wc = wCol
//		wr = wRow
//		m = int(n * wc / wr)
//		return true
//	}
//	return false
//}

//func newIntMatrix(rows, cols int) intMatrix {
//	m := intMatrix(make([][]int, rows))
//	for i := range m {
//		m[i] = make([]int, cols)
//	}
//	return m
//}
//
//func newFloatMatrix(rows, cols int) floatMatrix {
//	m := floatMatrix(make([][]float64, rows))
//	for i := range m {
//		m[i] = make([]float64, cols)
//	}
//	return m
//}

// New creates a full sized ethash PoW scheme and starts a background thread for
// remote mining, also optionally notifying a batch of remote services of new work
// packages.

func New(config Config, notify []string, noverify bool) *ECC {
	if config.Log == nil {
		config.Log = log.Root()
	}
	ecc := &ECC{
		config:       config,
		update:       make(chan struct{}),
		hashrate:     metrics.NewMeterForced(),
		workCh:       make(chan *sealTask),
		fetchWorkCh:  make(chan *sealWork),
		submitWorkCh: make(chan *mineResult),
		fetchRateCh:  make(chan chan uint64),
		submitRateCh: make(chan *hashrate),
	}
	if config.PowMode == ModeShared {
		ecc.shared = sharedECC
	}
	ecc.remote = startRemoteSealer(ecc, notify, noverify)
	return ecc
}

func NewTester(notify []string, noverify bool) *ECC {
	ecc := &ECC{
		config:       Config{PowMode: ModeTest},
		update:       make(chan struct{}),
		hashrate:     metrics.NewMeterForced(),
		workCh:       make(chan *sealTask),
		fetchWorkCh:  make(chan *sealWork),
		submitWorkCh: make(chan *mineResult),
		fetchRateCh:  make(chan chan uint64),
		submitRateCh: make(chan *hashrate),
	}
	ecc.remote = startRemoteSealer(ecc, notify, noverify)
	return ecc
}

// NewFaker creates a ethash consensus engine with a fake PoW scheme that accepts
// all blocks' seal as valid, though they still have to conform to the Ethereum
// consensus rules.
func NewFaker() *ECC {
	return &ECC{
		config: Config{
			PowMode: ModeFake,
			Log:     log.Root(),
		},
	}
}

// NewFakeFailer creates a ethash consensus engine with a fake PoW scheme that
// accepts all blocks as valid apart from the single one specified, though they
// still have to conform to the Ethereum consensus rules.
func NewFakeFailer(fail uint64) *ECC {
	return &ECC{
		config: Config{
			PowMode: ModeFake,
			Log:     log.Root(),
		},
		fakeFail: fail,
	}
}

// NewFakeDelayer creates a ethash consensus engine with a fake PoW scheme that
// accepts all blocks as valid, but delays verifications by some time, though
// they still have to conform to the Ethereum consensus rules.
func NewFakeDelayer(delay time.Duration) *ECC {
	return &ECC{
		config: Config{
			PowMode: ModeFake,
			Log:     log.Root(),
		},
		fakeDelay: delay,
	}
}

// NewFullFaker creates an ethash consensus engine with a full fake scheme that
// accepts all blocks as valid, without checking any consensus rules whatsoever.
func NewFullFaker() *ECC {
	return &ECC{
		config: Config{
			PowMode: ModeFullFake,
			Log:     log.Root(),
		},
	}
}

// NewShared creates a full sized ethash PoW shared between all requesters running
// in the same process.
//func NewShared() *ECC {
//	return &ECC{shared: sharedECC}
//}

// Close closes the exit channel to notify all backend threads exiting.
func (ecc *ECC) Close() error {
	return ecc.StopRemoteSealer()
}

// StopRemoteSealer stops the remote sealer
func (ecc *ECC) StopRemoteSealer() error {
	ecc.closeOnce.Do(func() {
		// Short circuit if the exit channel is not allocated.
		if ecc.remote == nil {
			return
		}
		close(ecc.remote.requestExit)
		<-ecc.remote.exitCh
	})
	return nil
}

// Threads returns the number of mining threads currently enabled. This doesn't
// necessarily mean that mining is running!
func (ecc *ECC) Threads() int {
	ecc.lock.Lock()
	defer ecc.lock.Unlock()

	return ecc.threads
}

// SetThreads updates the number of mining threads currently enabled. Calling
// this method does not start mining, only sets the thread count. If zero is
// specified, the miner will use all cores of the machine. Setting a thread
// count below zero is allowed and will cause the miner to idle, without any
// work being done.
func (ecc *ECC) SetThreads(threads int) {
	ecc.lock.Lock()
	defer ecc.lock.Unlock()

	// If we're running a shared PoW, set the thread count on that instead
	if ecc.shared != nil {
		ecc.shared.SetThreads(threads)
		return
	}
	// Update the threads and ping any running seal to pull in any changes
	ecc.threads = threads
	select {
	case ecc.update <- struct{}{}:
	default:
	}
}

// Hashrate implements PoW, returning the measured rate of the search invocations
// per second over the last minute.
// Note the returned hashrate includes local hashrate, but also includes the total
// hashrate of all remote miner.
func (ecc *ECC) Hashrate() float64 {
	// Short circuit if we are run the ecc in normal/test mode.

	var res = make(chan uint64, 1)

	select {
	case ecc.remote.fetchRateCh <- res:
	case <-ecc.remote.exitCh:
		// Return local hashrate only if ecc is stopped.
		return ecc.hashrate.Rate1()
	}

	// Gather total submitted hash rate of remote sealers.
	return ecc.hashrate.Rate1() + float64(<-res)
}

// APIs implements consensus.Engine, returning the user facing RPC APIs.
func (ecc *ECC) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	// In order to ensure backward compatibility, we exposes ecc RPC APIs
	// to both eth and ecc namespaces.
	return []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   &API{ecc},
			Public:    true,
		},
		{
			Namespace: "ecc",
			Version:   "1.0",
			Service:   &API{ecc},
			Public:    true,
		},
	}
}

// hasher is a repetitive hasher allowing the same hash data structures to be
// reused between hash runs instead of requiring new ones to be created.
type hasher func(dest []byte, data []byte)

// makeHasher creates a repetitive hasher, allowing the same hash data structures to
// be reused between hash runs instead of requiring new ones to be created. The returned
// function is not thread safe!
func makeHasher(h hash.Hash) hasher {
	// sha3.state supports Read to get the sum, use it to avoid the overhead of Sum.
	// Read alters the state but we reset the hash before every operation.
	type readerHash interface {
		hash.Hash
		Read([]byte) (int, error)
	}
	rh, ok := h.(readerHash)
	if !ok {
		panic("can't find Read method on hash")
	}
	outputLen := rh.Size()
	return func(dest []byte, data []byte) {
		rh.Reset()
		rh.Write(data)
		rh.Read(dest[:outputLen])
	}
}

// seedHash is the seed to use for generating a verification cache and the mining
// dataset.
func seedHash(block uint64) []byte {
	seed := make([]byte, 32)
	if block < epochLength {
		return seed
	}
	keccak256 := makeHasher(sha3.NewLegacyKeccak256())
	for i := 0; i < int(block/epochLength); i++ {
		keccak256(seed, seed)
	}
	return seed
}

//// SeedHash is the seed to use for generating a verification cache and the mining
//// dataset.
func SeedHash(block uint64) []byte {
	return seedHash(block)
}
