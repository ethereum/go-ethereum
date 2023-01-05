package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/urfave/cli/v2"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	DataDirFlag = &flags.DirectoryFlag{
		Name:     "datadir",
		Usage:    "Data directory for the databases and keystore",
		Value:    flags.DirectoryString(node.DefaultDataDir()),
		Category: flags.EthCategory,
	}
	CacheFlag = &cli.IntFlag{
		Name:     "cache",
		Usage:    "Cache size in Mb",
		Value:    1000,
		Category: flags.EthCategory,
	}
	FromBlockFlag = &cli.Int64Flag{
		Name:     "from",
		Usage:    "Starting block number, default is 1",
		Value:    1,
		Category: flags.EthCategory,
	}
	ToBlockFlag = &cli.Int64Flag{
		Name:     "to",
		Usage:    "Ending block number, default is unlimited",
		Category: flags.EthCategory,
	}
	MinBalanceFlag = &cli.Float64Flag{
		Name:     "min",
		Usage:    "Minimum current balance, ETH. Default is 0",
		Category: flags.EthCategory,
	}
	MaxBalanceFlag = &cli.Float64Flag{
		Name:     "max",
		Usage:    "Maximum current balance, ETH. Default is unlimited",
		Category: flags.EthCategory,
	}
	EmptyFlag = &cli.BoolFlag{
		Name:     "empty",
		Usage:    "Include empty accounts",
		Category: flags.EthCategory,
	}
	MaxMemoryFlag = &cli.IntFlag{
		Name:     "maxmem",
		Usage:    "Maximum number of megabytes to allocate for in-memory indexing, default 2000",
		Category: flags.EthCategory,
	}
	DistFlag = &cli.BoolFlag{
		Name:   "distribution",
		Usage:  "Compute the balance distribution (internal)",
		Hidden: true,
	}
	TestFlag = &cli.BoolFlag{
		Name:   "test",
		Usage:  "Test",
		Hidden: true,
	}
	app = flags.NewApp("account parser")
)

func init() {
	app.Action = parse
	app.Flags = []cli.Flag{DataDirFlag, FromBlockFlag, ToBlockFlag, MinBalanceFlag, MaxBalanceFlag, MaxMemoryFlag,
		DistFlag, TestFlag, CacheFlag, EmptyFlag}
}

func mgweiToStr(n *big.Int) string {
	balanceStr := n.String()
	if len(balanceStr) >= 3 {
		return balanceStr[:len(balanceStr)-3] + "." + balanceStr[:len(balanceStr)-3]
	} else {
		return "0." + strings.Repeat("0", 3-len(balanceStr)) + balanceStr
	}
}

func openDatabase(dataDir string) (ethdb.Database, error) {
	chainDataDir := filepath.Join(filepath.Join(dataDir, "geth"), "chaindata")
	err := state.CreateAddressIndex(filepath.Join(filepath.Join(dataDir, "geth"), "addressindex"))
	if err != nil {
		return nil, err
	}
	return rawdb.NewLevelDBDatabaseWithFreezer(
		chainDataDir,
		2048,
		8192,
		filepath.Join(chainDataDir, "ancient"),
		"eth/db/chaindata",
		true)
}

func weiFromEth(x float64) *big.Int {
	r := new(big.Int)
	G := big.NewInt(1000000000)
	for i := 0; i < 3; i++ {
		r.Mul(r, G)
		n, f := math.Modf(x)
		r.Add(r, big.NewInt(int64(n)))
		x = f * 1000000000.0
	}
	return r
}

var StopAddress = common.Address{1, 3, 5, 7, 9, 8, 6, 4, 2, 0, 33, 77, 99, 255, 255, 255, 255, 254, 253}

type BalanceFetcher struct {
	maxWorkers int
	mu         sync.Mutex
	jobs       chan common.Address
	snap       snapshot.Snapshot
}

func NewBalanceFetcher(db ethdb.Database, dataDir string, cache int) (*BalanceFetcher, error) {
	headBlockHash := rawdb.ReadHeadBlockHash(db)
	headBlockNumber := rawdb.ReadHeaderNumber(db, headBlockHash)
	header := rawdb.ReadHeader(db, headBlockHash, *headBlockNumber)

	snapConfig := snapshot.Config{
		CacheSize:  cache,
		Recovery:   false,
		NoBuild:    true,
		AsyncBuild: false,
	}

	root := header.Root

	triecfg := &trie.Config{1618, filepath.Join(filepath.Join(dataDir, "geth"), "triecache"), false}
	stateCache := state.NewDatabaseWithConfig(db, triecfg)
	snaps, err := snapshot.New(snapConfig, db, stateCache.TrieDB(), root)

	if err != nil {
		return nil, err
	}

	bf := &BalanceFetcher{
		maxWorkers: runtime.GOMAXPROCS(0),
		jobs:       make(chan common.Address),
		snap:       snaps.Snapshot(root),
	}

	return bf, nil
}

func (bf *BalanceFetcher) Start(callback func(common.Address, *big.Int)) {
	for i := 0; i < bf.maxWorkers; i++ {
		go func(bf *BalanceFetcher) {
			//stateDb, _ := state.New(root, stateCache, snaps)
			hasher := crypto.NewKeccakState()
			for {
				address := <-bf.jobs
				if address == StopAddress {
					break
				}
				acc, err := bf.snap.Account(crypto.HashData(hasher, address.Bytes()))
				if acc != nil && err == nil {
					balance := acc.Balance
					bf.mu.Lock()
					callback(address, balance)
					bf.mu.Unlock()
				}
			}
		}(bf)
	}

}

func (bf *BalanceFetcher) Address(a common.Address) {
	bf.jobs <- a
}

func (bf *BalanceFetcher) Finish() {
	for i := 0; i < bf.maxWorkers; i++ {
		bf.jobs <- StopAddress
	}
}

func iterateAccounts(bf *BalanceFetcher, includeEmpty bool, min float64, max float64, callback func(common.Address, *big.Int)) error {
	minBig := weiFromEth(min)
	var maxBig *big.Int = nil
	if !math.IsNaN(max) {
		maxBig = weiFromEth(max)
	}

	state.GlobalAddressIndex.SetReadOnly(true)

	bf.Start(func(address common.Address, balance *big.Int) {
		c := balance.Cmp(minBig)
		if (includeEmpty && c >= 0 || !includeEmpty && c > 0) && (maxBig == nil || balance.Cmp(maxBig) <= 0) {
			callback(address, balance)
		}
	})

	state.GlobalAddressIndex.IterateSeenAddresses(func(address common.Address) bool {
		bf.Address(address)
		//fmt.Printf("%s: %s\n", accIt.Hash(), mgweiToStr(q))
		return true
	})

	bf.Finish()

	return nil
}

func lastBlock(db ethdb.Database) (uint64, error) {
	head := rawdb.ReadHeadBlockHash(db)
	if head == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		return 0, fmt.Errorf("empty database")
	}
	number := rawdb.ReadHeaderNumber(db, head)
	if number == nil {
		return 0, fmt.Errorf("damaged database: no header block found")
	} else {
		return *number, nil
	}
}

func iterateAddressCandidatesFromTxData(data []byte, callback func(addr common.Address)) {
	state := 0
	for i := 0; i < len(data)-21; i++ {
		if data[i] == 0 {
			state++
			if state >= 12 {
				callback(common.BytesToAddress(data[i+1 : i+21]))
			}
		} else {
			state = 0
		}
	}
}

func iterateTransactions(db ethdb.Database, from uint64, to uint64, callback func(common.Address)) error {
	for n := from; n <= to; n++ {
		data := rawdb.ReadCanonicalBodyRLP(db, n)
		var body types.Body
		if err := rlp.DecodeBytes(data, &body); err != nil {
			return fmt.Errorf("failed to decode block body #%d: %s", n, err)
		}

		signer := types.MakeSigner(params.MainnetChainConfig, big.NewInt(int64(n)))

		for _, tx := range body.Transactions {
			if tx.To() == nil {
				continue // contract creation
			}

			sender, err := signer.Sender(tx)
			if err == nil {
				callback(sender)
			}

			if len(tx.Data()) == 0 { // the usual transaction (not a call)
				callback(*tx.To())
			} else {
				iterateAddressCandidatesFromTxData(tx.Data(), func(addr common.Address) {
					if state.GlobalAddressIndex.HasAddress(addr) {
						callback(addr)
					}
				})
			}
		}

	}

	return nil
}

func printAddress(a common.Address) {
	os.Stdout.WriteString(a.String() + "\n")
}

var (
	addressDistributionZeros     uint64 = 104215475
	addressDistributionByBalance        = [200]uint64{0, 23390755, 32222266, 38373290, 42263243, 44817777, 46364352, 47701766,
		48990831, 50124440, 51266973, 52086074, 52829785, 53594345, 54313563, 54960480, 55502291, 56032794, 56514634,
		57057655, 57494212, 57923794, 58430800, 58897206, 59551253, 60014160, 60417561, 61428901, 62101478, 62446016,
		62739363, 63060839, 63353713, 63643046, 63974606, 64364952, 64614581, 64884586, 65159824, 65401100, 65633920,
		65875301, 66109442, 66359763, 66594693, 66829398, 67114831, 67375847, 67940094, 68102593, 68264981, 68402876,
		68539586, 68690021, 68819454, 68954814, 69084678, 69225974, 69352769, 69478807, 69606362, 69732841, 69881035,
		69999932, 70119183, 70224050, 70333106, 70448763, 70565561, 70673009, 70777618, 70875192, 70999535, 71097781,
		71195369, 71282797, 71423549, 71517958, 71612528, 71703333, 71794599, 71893630, 71987228, 72084802, 72186381,
		72295908, 72419395, 72551319, 72704080, 72915373, 73151710, 73456161, 74105440, 74945985, 75455647, 75632981,
		75692928, 75750026, 75806070, 75861697, 75921946, 75975312, 76028994, 76082085, 76137543, 76200264, 76270099,
		76338195, 76390358, 76455281, 76511318, 76566207, 76620018, 76672197, 76729972, 76781620, 76834793, 76890287,
		76947946, 77018904, 77070140, 77121185, 77169934, 77219473, 77271645, 77321433, 77372487, 77421284, 77471414,
		77515338, 77562673, 77626679, 77926189, 77977462, 78024544, 78071973, 78119174, 78167729, 78220038, 78269611,
		78327538, 78383125, 78431854, 78546127, 78590817, 78634876, 78677595, 78731770, 78770865, 78816172, 78857387,
		78895424, 78937972, 78977271, 79018853, 79057040, 79094775, 79136653, 79174971, 79212619, 79250199, 79287334,
		79328348, 79369595, 79406000, 79441866, 79484575, 79516120, 79547556, 79579789, 79612480, 79650992, 79699225,
		79759964, 79846736, 79879711, 79910344, 79938383, 79966592, 79994754, 80025115, 80057930, 80086796, 80116718,
		80143992, 80174280, 80203441, 80232933, 80262518, 80292748, 80354618, 80382563, 80408504, 80433262, 80458289,
		80485186, 80510985, 80536720, 80562216, 90323894}
	addressDistributionStep  = 0.00021111
	addressDistributionTotal = addressDistributionZeros + addressDistributionByBalance[199]
)

const (
	addressesByBlock = 13.2
)

func estimateAddressCount(max float64) uint64 {
	if math.IsNaN(max) {
		return addressDistributionByBalance[len(addressDistributionByBalance)-1]
	}
	nbin := int(max / addressDistributionStep)
	offset := max - addressDistributionStep*float64(nbin)
	if nbin >= len(addressDistributionByBalance)-1 {
		return addressDistributionByBalance[len(addressDistributionByBalance)-1]
	}
	// linear interpolation
	return uint64((1.0-offset)*float64(addressDistributionByBalance[nbin]) + offset*float64(addressDistributionByBalance[nbin+1]))
}

func minUint64(a uint64, b uint64) uint64 {
	if a > b {
		return b
	}
	return a
}

func parse(ctx *cli.Context) error {
	if ctx.IsSet(DistFlag.Name) {
		return dist(ctx)
	}

	if ctx.IsSet(TestFlag.Name) {
		return test(ctx)
	}

	dataDir := ctx.Path(DataDirFlag.Name)
	db, err := openDatabase(dataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	min := ctx.Float64(MinBalanceFlag.Name)
	max := math.NaN()
	balanceLimited := true
	if ctx.IsSet(MaxBalanceFlag.Name) {
		max = ctx.Float64(MaxBalanceFlag.Name)
	} else if min == 0 && ctx.IsSet(EmptyFlag.Name) {
		balanceLimited = false
	}
	includeEmpty := ctx.IsSet(EmptyFlag.Name)
	if min > 0 && includeEmpty {
		panic("--min is not compatible with -empty")
	}

	cache := CacheFlag.Value
	if ctx.IsSet(CacheFlag.Name) {
		cache = ctx.Int(CacheFlag.Name)
	}

	var bf *BalanceFetcher = nil
	if balanceLimited {
		bf, err = NewBalanceFetcher(db, dataDir, cache)
		if err != nil {
			return err
		}
	}

	if ctx.IsSet(FromBlockFlag.Name) || ctx.IsSet(ToBlockFlag.Name) {
		from := ctx.Uint64(FromBlockFlag.Name)
		to, err := lastBlock(db)
		if err != nil {
			return err
		}
		if ctx.IsSet(ToBlockFlag.Name) {
			to = minUint64(to, ctx.Uint64(ToBlockFlag.Name))
		}

		if from > to {
			return fmt.Errorf("-%s should be greater or equal to -%s", ToBlockFlag.Name, FromBlockFlag.Name)
		}
		addressesByRangeEstimate := uint64(addressesByBlock * float64(to-from+1))

		var maxMemoryMb uint64
		if ctx.IsSet(MaxMemoryFlag.Name) {
			maxMemoryMb = ctx.Uint64(MaxMemoryFlag.Name)
		} else {
			maxMemoryMb = 2000
		}
		maxMemory := maxMemoryMb * 1024 * 1024
		maxAddressesInMemory := maxMemory / 32

		if balanceLimited {
			// iterate over transactions and check the balance for every address
			var usedAddresses AddressSet // for unique address selection
			if addressesByRangeEstimate <= maxAddressesInMemory {
				usedAddresses = NewTreeSet()
			} else {
				println("Using a Bloom filter; expect false positives")
				usedAddresses = NewBloomFilterSet(maxMemory, addressesByRangeEstimate)
			}

			minBig := weiFromEth(min)
			var maxBig *big.Int = nil
			if !math.IsNaN(max) {
				maxBig = weiFromEth(max)
			}

			println("Iterating over transactions")
			bf.Start(func(a common.Address, balance *big.Int) {
				c := balance.Cmp(minBig)
				if (includeEmpty && c >= 0 || !includeEmpty && c > 0) && (maxBig == nil || balance.Cmp(maxBig) <= 0) {
					printAddress(a)
				}
			})
			iterateTransactions(db, from, to, func(a common.Address) {
				if usedAddresses.Add(a) {
					bf.Address(a)
				}
			})
			bf.Finish()
		} else {
			var usedAddresses AddressSet // for unique address selection
			if addressesByRangeEstimate <= maxAddressesInMemory {
				usedAddresses = NewTreeSet()
			} else {
				println("Using a Bloom filter; expect false positives")
				usedAddresses = NewBloomFilterSet(maxMemory, addressesByRangeEstimate)
			}
			println("Iterating over transactions")
			iterateTransactions(db, from, to, func(a common.Address) {
				if usedAddresses.Add(a) {
					printAddress(a)
				}
			})
		}

	} else {
		println("Iterating accounts")
		iterateAccounts(bf, includeEmpty, min, max, func(a common.Address, balance *big.Int) { printAddress(a) })
	}

	return nil
}

func dist(ctx *cli.Context) error {
	dataDir := ctx.Path(DataDirFlag.Name)
	db, err := openDatabase(dataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	cache := CacheFlag.Value
	if ctx.IsSet(CacheFlag.Name) {
		cache = ctx.Int(CacheFlag.Name)
	}

	gwei := big.NewInt(1000000000)
	q := big.NewInt(0)
	const Nbins = 199
	arr := make([]uint64, Nbins)
	zeros := 0
	speed := 0
	lastTime := time.Now().UnixNano()

	bf, err := NewBalanceFetcher(db, dataDir, cache)
	if err != nil {
		return err
	}
	err = iterateAccounts(bf, true, 0.0, math.NaN(), func(a common.Address, balance *big.Int) {
		//println(a.String())
		q.Div(balance, gwei) // q = balance in gwei
		eth := float64(q.Uint64()) * 0.000000001
		if eth == 0 {
			zeros++
		} else {
			nbin := int(eth / addressDistributionStep)
			if nbin >= Nbins {
				nbin = Nbins - 1
			}
			arr[nbin]++
		}
		speed++
		if speed%10 == 0 {
			time := time.Now().UnixNano()
			if time-lastTime >= 1000000000 {
				fmt.Printf("%f addresses/s    \r", float64(speed)/float64(time-lastTime)*1000000000)
				lastTime = time
				speed = 0
			}
		}
	})

	if err != nil {
		return err
	}

	fmt.Printf("Zeros: %d\n", zeros)
	fmt.Printf("Non-seros paced at step %f:\n", addressDistributionStep)
	sum := uint64(0)
	print(sum, ", ")
	for _, a := range arr {
		sum += a
		print(sum, ", ")
	}

	println()

	return nil
}

func test(ctx *cli.Context) error {
	dataDir := ctx.Path(DataDirFlag.Name)
	db, err := openDatabase(dataDir)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, _, _, _ := rawdb.ReadTransaction(db, common.HexToHash("0x03ba843efdcdce8d8e544ceff3e187da0a91fd35fb333f84b30a930b4f1d6a6e"))
	signer := types.MakeSigner(params.MainnetChainConfig, big.NewInt(16231664))

	sender, err := signer.Sender(tx)
	if err == nil {
		println(sender.String(), "->", tx.To().String())
	}

	iterateAddressCandidatesFromTxData(tx.Data(), func(addr common.Address) {
		println(addr.String())
	})

	return nil
}

func main() {

	log.Root().SetHandler(log.DiscardHandler())

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
