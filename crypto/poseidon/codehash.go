package poseidon

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
)

const defaultPoseidonChunk = 3
const nBytesToFieldElement = 31

func CodeHash(code []byte) (h common.Hash) {
	nBytes := int64(len(code))

	// step 1: pad code with 0x0 (STOP) so that len(code) % nBytesToFieldElement == 0
	// step 2: for every nBytesToFieldElement bytes, convert to Fr, so that we get a Fr array
	var length = (len(code) + nBytesToFieldElement - 1) / nBytesToFieldElement

	Frs := make([]*big.Int, length)
	ii := 0

	for ii < length-1 {
		Frs[ii] = big.NewInt(0)
		Frs[ii].SetBytes(code[ii*nBytesToFieldElement : (ii+1)*nBytesToFieldElement])
		ii++
	}

	if length > 0 {
		Frs[ii] = big.NewInt(0)
		bytes := make([]byte, nBytesToFieldElement)
		copy(bytes, code[ii*nBytesToFieldElement:])
		Frs[ii].SetBytes(bytes)
	}

	// step 3: apply the array onto a sponge process with the current poseidon scheme
	// (3 Frs permutation and 1 Fr for output, so the throughout is 2 Frs)
	// step 4: convert final root Fr to u256 (big-endian representation)
	hash, err := HashWithCap(Frs, defaultPoseidonChunk, nBytes)
	if err != nil {
		return common.Hash{}
	}
	return common.BigToHash(hash)
}
