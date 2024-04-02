package eccpow

import (
	crand "crypto/rand"
	"math"
	"math/big"
	"math/rand"

	"github.com/cryptoecc/ETH-ECC/core/types"
)

//Parameters for matrix and seed
const (
	BigInfinity = 1000000.0
	Inf         = 64.0
	MaxNonce    = 1<<32 - 1

	// These parameters are only used for the decoding function.
	maxIter  = 20   // The maximum number of iteration in the decoder
	crossErr = 0.01 // A transisient error probability. This is also fixed as a small value
)

type Parameters struct {
	n    int
	m    int
	wc   int
	wr   int
	seed int
}

// setParameters sets n, wc, wr, m, seed return parameters and difficulty level
func setParameters(header *types.Header) (Parameters, int) {
	//level := SearchLevel(header.Difficulty)
	level := SearchLevel(header.Difficulty)

	parameters := Parameters{
		n:  Table[level].n,
		wc: Table[level].wc,
		wr: Table[level].wr,
	}
	parameters.m = int(parameters.n * parameters.wc / parameters.wr)
	parameters.seed = generateSeed(header.ParentHash)

	return parameters, level
}

// setParameters sets n, wc, wr, m, seed return parameters and difficulty level
func setParameters_Seoul(header *types.Header) (Parameters, int) {
	//level := SearchLevel(header.Difficulty)
	level := SearchLevel_Seoul(header.Difficulty)
	table := getTable(level)
	parameters := Parameters{
		n:  table.n,
		wc: table.wc,
		wr: table.wr,
	}
	parameters.m = int(parameters.n * parameters.wc / parameters.wr)
	parameters.seed = generateSeed(header.ParentHash)

	return parameters, level
}


//generateRandomNonce generate 64bit random nonce with similar way of ethereum block nonce
func generateRandomNonce() uint64 {
	seed, _ := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	source := rand.New(rand.NewSource(seed.Int64()))

	return uint64(source.Int63())
}

func funcF(x float64) float64 {
	if x >= BigInfinity {
		return 1.0 / BigInfinity
	} else if x <= (1.0 / BigInfinity) {
		return BigInfinity
	} else {
		return math.Log((math.Exp(x) + 1) / (math.Exp(x) - 1))
	}
}

func infinityTest(x float64) float64 {
	if x >= Inf {
		return Inf
	} else if x <= -Inf {
		return -Inf
	} else {
		return x
	}
}

//generateSeed generate seed using previous hash vector
func generateSeed(phv [32]byte) int {
	sum := 0
	for i := 0; i < len(phv); i++ {
		sum += int(phv[i])
	}
	return sum
}

//generateH generate H matrix using parameters
//generateH Cannot be sure rand is same with original implementation of C++
func generateH(parameters Parameters) [][]int {
	var H [][]int
	var hSeed int64
	var colOrder []int

	hSeed = int64(parameters.seed)
	k := parameters.m / parameters.wc

	H = make([][]int, parameters.m)
	for i := range H {
		H[i] = make([]int, parameters.n)
	}

	for i := 0; i < k; i++ {
		for j := i * parameters.wr; j < (i+1)*parameters.wr; j++ {
			H[i][j] = 1
		}
	}

	for i := 1; i < parameters.wc; i++ {
		colOrder = nil
		for j := 0; j < parameters.n; j++ {
			colOrder = append(colOrder, j)
		}

		src := rand.NewSource(hSeed)
		rnd := rand.New(src)
		rnd.Seed(hSeed)
		rnd.Shuffle(len(colOrder), func(i, j int) {
			colOrder[i],colOrder[j] = colOrder[j], colOrder[i]
		})
		hSeed--

		for j := 0; j < parameters.n; j++ {
			index := (colOrder[j]/parameters.wr + k*i)
			H[index][j] = 1
		}
	}

	return H
}

//generateQ generate colInRow and rowInCol matrix using H matrix
func generateQ(parameters Parameters, H [][]int) ([][]int, [][]int) {
	colInRow := make([][]int, parameters.wr)
	for i := 0; i < parameters.wr; i++ {
		colInRow[i] = make([]int, parameters.m)
	}

	rowInCol := make([][]int, parameters.wc)
	for i := 0; i < parameters.wc; i++ {
		rowInCol[i] = make([]int, parameters.n)
	}

	rowIndex := 0
	colIndex := 0

	for i := 0; i < parameters.m; i++ {
		for j := 0; j < parameters.n; j++ {
			if H[i][j] == 1 {
				colInRow[colIndex%parameters.wr][i] = j
				colIndex++

				rowInCol[rowIndex/parameters.n][j] = i
				rowIndex++
			}
		}
	}

	return colInRow, rowInCol
}

//generateHv generate hashvector
//It needs to compare with origin C++ implementation Especially when sha256 function is used
func generateHv(parameters Parameters, encryptedHeaderWithNonce []byte) []int {
	hashVector := make([]int, parameters.n)

	/*
		if parameters.n <= 256 {
			tmpHashVector = sha256.Sum256(headerWithNonce)
		} else {
			/*
				This section is for a case in which the size of a hash vector is larger than 256.
				This section will be implemented soon.
		}
			transform the constructed hexadecimal array into an binary array
			ex) FE01 => 11111110000 0001
	*/

	for i := 0; i < parameters.n/8; i++ {
		decimal := int(encryptedHeaderWithNonce[i])
		for j := 7; j >= 0; j-- {
			hashVector[j+8*(i)] = decimal % 2
			decimal /= 2
		}
	}

	//outputWord := hashVector[:parameters.n]
	return hashVector
}
