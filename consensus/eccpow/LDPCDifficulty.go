package eccpow

import (
	"math"
	"math/big"

	"github.com/cryptoecc/ETH-ECC/core/types"
)

/*
	https://ethereum.stackexchange.com/questions/5913/how-does-the-ethereum-homestead-difficulty-adjustment-algorithm-work?noredirect=1&lq=1
	https://github.com/ethereum/EIPs/issues/100

	Ethereum difficulty adjustment
	 algorithm:
	diff = (parent_diff +
	         (parent_diff / 2048 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // 9), -99))
	        ) + 2^(periodCount - 2)

	LDPC difficulty adjustment
	algorithm:
	diff = (parent_diff +
			(parent_diff / 256 * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // BlockGenerationTime), -99)))

	Why 8?
	This number is sensitivity of blockgeneration time
	If this number is high, difficulty is not changed much when block generation time is different from goal of block generation time
	But if this number is low, difficulty is changed much when block generatin time is different  from goal of block generation time

*/

// "github.com/cryptoecc/ETH-ECC/consensus/ethash/consensus.go"
// Some weird constants to avoid constant memory allocs for them.
var (
	MinimumDifficulty   = ProbToDifficulty(Table[0].miningProb)
	BlockGenerationTime = big.NewInt(36) // 36) // 10 ) // 36)
	Sensitivity         = big.NewInt(8)

	// BlockGenerationTime for Seoul
	BlockGenerationTimeSeoul = big.NewInt(10) // 36) // 10 ) // 36)
	SeoulDifficulty   = big.NewInt(1023)

	//initLevel int = 10
	minLevel  int = 10
	diff_interval = 100

	//count  int = -1
	//init_c int = 2
)

const (
	// frontierDurationLimit is for Frontier:
	// The decision boundary on the blocktime duration used to determine
	// whether difficulty should go up or down.
	frontierDurationLimit = 10
	// minimumDifficulty The minimum that the difficulty may ever be.
	minimumDifficulty = 131072
	// expDiffPeriod is the exponential difficulty period
	expDiffPeriodUint = 100000
	// difficultyBoundDivisorBitShift is the bound divisor of the difficulty (2048),
	// This constant is the right-shifts to use for the division.
	difficultyBoundDivisor = 11
)

// MakeLDPCDifficultyCalculator calculate difficulty using difficulty table
func MakeLDPCDifficultyCalculator() func(time uint64, parent *types.Header) *big.Int {
	return func(time uint64, parent *types.Header) *big.Int {
		bigTime := new(big.Int).SetUint64(time)
		bigParentTime := new(big.Int).SetUint64(parent.Time)

		// holds intermediate values to make the algo easier to read & audit
		x := new(big.Int)
		y := new(big.Int)

		// (2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // BlockGenerationTime
		x.Sub(bigTime, bigParentTime)
		//fmt.Printf("block_timestamp - parent_timestamp : %v\n", x)

		x.Div(x, BlockGenerationTime)
		//fmt.Printf("(block_timestamp - parent_timestamp) / BlockGenerationTime : %v\n", x)

		if parent.UncleHash == types.EmptyUncleHash {
			//fmt.Printf("No uncle\n")
			x.Sub(big1, x)
		} else {
			//fmt.Printf("Uncle block exists")
			x.Sub(big2, x)
		}
		//fmt.Printf("(2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) / BlockGenerationTime : %v\n", x)

		// max((2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9, -99)
		if x.Cmp(bigMinus99) < 0 {
			x.Set(bigMinus99)
		}
		//fmt.Printf("max(1 - (block_timestamp - parent_timestamp) / BlockGenerationTime, -99) : %v\n", x)

		// parent_diff + (parent_diff / Sensitivity * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // BlockGenerationTime), -99))
		y.Div(parent.Difficulty, Sensitivity)
		//fmt.Printf("parent.Difficulty / 8 : %v\n", y)

		x.Mul(y, x)
		//fmt.Printf("parent.Difficulty / 8 * max(1 - (block_timestamp - parent_timestamp) / BlockGenerationTime, -99) : %v\n", x)

		x.Add(parent.Difficulty, x)
		//fmt.Printf("parent.Difficulty - parent.Difficulty / 8 * max(1 - (block_timestamp - parent_timestamp) / BlockGenerationTime, -99) : %v\n", x)

		// minimum difficulty can ever be (before exponential factor)
		if x.Cmp(MinimumDifficulty) < 0 {
			x.Set(MinimumDifficulty)
		}

		//fmt.Printf("x : %v, Minimum difficulty : %v\n", x, MinimumDifficulty)
		return x
	}
}

func MakeLDPCDifficultyCalculator_Seoul() func(time uint64, parent *types.Header) *big.Int {
	return func(time uint64, parent *types.Header) *big.Int {
		bigTime := new(big.Int).SetUint64(time)
		bigParentTime := new(big.Int).SetUint64(parent.Time)

		// holds intermediate values to make the algo easier to read & audit
		x := new(big.Int)
		y := new(big.Int)

		// (2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // BlockGenerationTime
		x.Sub(bigTime, bigParentTime)
		//fmt.Printf("block_timestamp - parent_timestamp : %v\n", x)

		x.Div(x, BlockGenerationTimeSeoul)
		//fmt.Printf("(block_timestamp - parent_timestamp) / BlockGenerationTime : %v\n", x)

		if parent.UncleHash == types.EmptyUncleHash {
			//fmt.Printf("No uncle\n")
			x.Sub(big1, x)
		} else {
			//fmt.Printf("Uncle block exists")
			x.Sub(big2, x)
		}
		//fmt.Printf("(2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) / BlockGenerationTime : %v\n", x)

		// max((2 if len(parent_uncles) else 1) - (block_timestamp - parent_timestamp) // 9, -99)
		if x.Cmp(bigMinus99) < 0 {
			x.Set(bigMinus99)
		}
		//fmt.Printf("max(1 - (block_timestamp - parent_timestamp) / BlockGenerationTime, -99) : %v\n", x)

		// parent_diff + (parent_diff / Sensitivity * max((2 if len(parent.uncles) else 1) - ((timestamp - parent.timestamp) // BlockGenerationTime), -99))
		y.Div(parent.Difficulty, Sensitivity)
		//fmt.Printf("parent.Difficulty / 8 : %v\n", y)

		x.Mul(y, x)
		//fmt.Printf("parent.Difficulty / 8 * max(1 - (block_timestamp - parent_timestamp) / BlockGenerationTime, -99) : %v\n", x)

		x.Add(parent.Difficulty, x)
		//fmt.Printf("parent.Difficulty - parent.Difficulty / 8 * max(1 - (block_timestamp - parent_timestamp) / BlockGenerationTime, -99) : %v\n", x)

		// minimum difficulty can ever be (before exponential factor)
		if x.Cmp(SeoulDifficulty) < 0 {
			x.Set(SeoulDifficulty)
		}

		//fmt.Printf("x : %v, Minimum difficulty : %v\n", x, MinimumDifficulty)
		return x
	}
}

// SearchLevel return next level by using currentDifficulty of header
// Type of Ethereum difficulty is *bit.Int so arg is *big.int
func SearchLevel(difficulty *big.Int) int {
	// foo := MakeLDPCDifficultyCalculator()
	// Next level := SearchNextLevel(foo(currentBlock's time stamp, parentBlock))

	var currentProb = DifficultyToProb(difficulty)
	var level int

	distance := 1.0
	for i := range Table {
		if math.Abs(currentProb-Table[i].miningProb) <= distance {
			level = Table[i].level
			distance = math.Abs(currentProb - Table[i].miningProb)
		} else {
			break
		}
	}

	return level
}

// SearchLevel return next level by using currentDifficulty of header
// Type of Ethereum difficulty is *bit.Int so arg is *big.int
func SearchLevel_Seoul(difficulty *big.Int) int {

	var level int

	difficultyf := new(big.Rat).SetInt(difficulty)
	level_prob := big.NewRat(29, 20)
	difficultyf.Quo(difficultyf, new(big.Rat).SetInt(SeoulDifficulty))

	for {
		difficultyf.Quo(difficultyf, level_prob)
		level++
		if difficultyf.Cmp(big.NewRat(1, 1)) < 0 {
			break
		}
	}

	return level
}
