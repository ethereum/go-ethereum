package main

import (
	"bytes"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/urfave/cli/v2"
	"math"
	"math/big"
	"os"
	"path/filepath"
	"strings"
)

var (
	DataDirFlag = &flags.DirectoryFlag{
		Name:     "datadir",
		Usage:    "Data directory for the databases and keystore",
		Value:    flags.DirectoryString(node.DefaultDataDir()),
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
	app = flags.NewApp("account parser")
)

func init() {
	app.Action = parse
	app.Flags = []cli.Flag{DataDirFlag, FromBlockFlag, ToBlockFlag, MinBalanceFlag, MaxBalanceFlag, MaxMemoryFlag, DistFlag}
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
	return rawdb.NewLevelDBDatabaseWithFreezer(
		chainDataDir,
		2048,
		8192,
		filepath.Join(chainDataDir, "ancient"),
		"eth/db/chaindata",
		false)
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

func iterateAccounts(db ethdb.Database, min float64, max float64, callback func(common.Address, *big.Int)) error {
	header := rawdb.ReadHeadHeader(db)
	if header == nil {
		return fmt.Errorf("no header found in the database")
	}

	snapConfig := snapshot.Config{
		CacheSize:  256,
		Recovery:   false,
		NoBuild:    true,
		AsyncBuild: false,
	}
	snaptree, err := snapshot.New(snapConfig, db, trie.NewDatabase(db), header.Root)
	if err != nil {
		return err
	}
	start := make([]byte, 20)
	accIt, err := snaptree.AccountIterator(header.Root, common.BytesToHash(start))
	if err != nil {
		return err
	}

	accIt, err = snaptree.AccountIterator(header.Root, common.BytesToHash(start))
	defer accIt.Release()

	minBig := weiFromEth(min)
	var maxBig *big.Int = nil
	if !math.IsNaN(max) {
		maxBig = weiFromEth(max)
	}

	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode := crypto.Keccak256(nil)

	for accIt.Next() {
		account, err := snapshot.FullAccount(accIt.Account())
		if err != nil {
			return err
		}
		if !bytes.Equal(account.CodeHash, emptyCode) {
			continue // skip contracts
		}
		if account.Balance.Cmp(minBig) >= 0 && (maxBig == nil || account.Balance.Cmp(maxBig) <= 0) {
			callback(common.BytesToAddress(accIt.Account()), account.Balance)
		}
		//fmt.Printf("%s: %s\n", accIt.Hash(), mgweiToStr(q))
	}

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

func iterateTransactions(db ethdb.Database, from uint64, to uint64, callback func(common.Address)) error {
	for n := from; n <= to; n++ {
		data := rawdb.ReadCanonicalBodyRLP(db, n)
		var body types.Body
		if err := rlp.DecodeBytes(data, &body); err != nil {
			return fmt.Errorf("failed to decode block body #%d: %s", n, "error", err)
		}
		for _, tx := range body.Transactions {
			if len(tx.Data()) == 0 { // the usual transaction (not a call)
				callback(*tx.To())
			}
		}

	}

	return nil
}

func printAddress(a common.Address) {
	println(a.String())
}

var (
	addressDistributionByBalance = [2]uint64{0, 100}
	addressDistributionStep      = 0.08
)

const (
	addressesByBlock = 13.2
)

func estimateAddressCount(max float64) uint64 {
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

	db, err := openDatabase(ctx.Path(DataDirFlag.Name))
	if err != nil {
		return err
	}
	defer db.Close()

	min := ctx.Float64(MinBalanceFlag.Name)
	max := math.NaN()
	balanceLimited := true
	if ctx.IsSet(MaxBalanceFlag.Name) {
		max = ctx.Float64(MaxBalanceFlag.Name)
	} else if min == 0 {
		balanceLimited = false
	}

	if ctx.IsSet(FromBlockFlag.Name) || ctx.IsSet(ToBlockFlag.Name) {
		addressesByBalanceEstimate := estimateAddressCount(max) - estimateAddressCount(min)
		from := ctx.Uint64(FromBlockFlag.Name)
		to, err := lastBlock(db)
		if err != nil {
			return err
		}
		if ctx.IsSet(ToBlockFlag.Name) {
			to = minUint64(to, ctx.Uint64(ToBlockFlag.Name))
		}

		if from < to {
			return fmt.Errorf("-%s whould be greater or equal to -%s", ToBlockFlag.Name, FromBlockFlag.Name)
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

		var addressesByBalance AddressSet
		var usedAddresses AddressSet
		if addressesByBalanceEstimate+addressesByRangeEstimate <= maxAddressesInMemory {
			addressesByBalance = NewTreeSet()
			usedAddresses = NewTreeSet()
		} else {
			if balanceLimited {
				addressesByBalance = NewBloomFilterSet(maxMemory/2, addressesByBalanceEstimate)
				usedAddresses = NewBloomFilterSet(maxMemory/2, addressesByRangeEstimate)
			} else {
				usedAddresses = NewBloomFilterSet(maxMemory, addressesByRangeEstimate)
			}
		}

		if balanceLimited {
			iterateAccounts(db, min, max, func(a common.Address, balance *big.Int) { addressesByBalance.Add(a) })
		}

		iterateTransactions(db, from, to, func(a common.Address) {
			if !balanceLimited || addressesByBalance.Contains(a) {
				if usedAddresses.Add(a) {
					printAddress(a)
				}
			}
		})
	} else {
		iterateAccounts(db, min, max, func(a common.Address, balance *big.Int) { printAddress(a) })
	}

	return nil
}

func dist(ctx *cli.Context) error {
	db, err := openDatabase(ctx.Path(DataDirFlag.Name))
	if err != nil {
		return err
	}
	defer db.Close()

	gwei := big.NewInt(1000000000)
	q := big.NewInt(0)
	const Nbins = 200
	arr := make([]uint64, Nbins)
	err = iterateAccounts(db, 0.0, math.NaN(), func(a common.Address, balance *big.Int) {
		q.Div(balance, gwei) // q = balance in gwei
		eth := float64(q.Uint64()) * 0.000000001
		nbin := int(eth / addressDistributionStep)
		if nbin >= Nbins {
			nbin = Nbins - 1
		}
		arr[nbin]++
	})

	if err != nil {
		return err
	}

	sum := uint64(0)
	print(sum, ", ")
	for _, a := range arr {
		sum += a
		print(sum, ", ")
	}

	println()

	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
