package eccpow

import (
	"fmt"
	"math"
	"math/big"
	"testing"
	"time"

	"github.com/cryptoecc/ETH-ECC/core/types"
)

func TestTablePrint(t *testing.T) {
	for i := range Table {
		fmt.Printf("level : %v, n : %v, wc : %v, wr : %v, decisionFrom : %v, decisionTo : %v, decisionStep : %v, miningProb : %v \n", Table[i].level, Table[i].n, Table[i].wc, Table[i].wr, Table[i].decisionFrom, Table[i].decisionTo, Table[i].decisionStep, Table[i].miningProb)
	}
}

func TestPrintReciprocal(t *testing.T) {
	for i := range Table {
		val := 1 / Table[i].miningProb
		bigInt := FloatToBigInt(val)
		fmt.Printf("Reciprocal of miningProb : %v \t big Int : %v\n", val, bigInt)
	}
}

func TestConversionFunc(t *testing.T) {
	for i := range Table {
		difficulty := ProbToDifficulty(Table[i].miningProb)
		miningProb := DifficultyToProb(difficulty)

		// Consider only integer part.
		fmt.Printf("Difficulty : %v \t MiningProb : %v\t, probability compare : %v \n", difficulty, miningProb, math.Abs(miningProb-Table[i].miningProb) < 1)
	}
}

func TestDifficultyChange(t *testing.T) {
	var hash []byte
	currentLevel := 0
	currentBlock := new(types.Header)
	// Parent block's timestamp is 0
	// compare elapse time(timestamp) and parent block's timestamp(0)
	currentBlock.Difficulty = big.NewInt(0)
	currentBlock.Time = 0
	currentBlock.UncleHash = types.EmptyUncleHash
	for i := 0; i < 5; i++ {
		fmt.Printf("Current Difficulty : %v\n", currentBlock.Difficulty)

		startTime := time.Now()

		RunOptimizedConcurrencyLDPC(currentBlock, hash)
		timeStamp := uint64(time.Since(startTime).Seconds())
		fmt.Printf("Block generation time : %v\n", timeStamp)

		difficultyCalculator := MakeLDPCDifficultyCalculator()
		nextDifficulty := difficultyCalculator(timeStamp, currentBlock)
		currentBlock.Difficulty = nextDifficulty
		nextLevel := SearchLevel(nextDifficulty)

		fmt.Printf("Current prob : %v, Next Level : %v,  Next difficulty : %v, Next difficulty from table : %v\n\n", Table[currentLevel].miningProb, Table[nextLevel].level, nextDifficulty, ProbToDifficulty(Table[nextLevel].miningProb))
		// currentBlock.ParentHash = outputWord conversion from []int to [32]byte
		currentLevel = nextLevel
		fmt.Printf("Current Level : %v\n", currentLevel)
	}
}
