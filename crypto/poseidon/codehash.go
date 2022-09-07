package poseidon

import (
	"math/big"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/crypto"
)

const defaultPoseidonChunk = 3

func CodeHash(code []byte) (h common.Hash) {
	// special case for nil hash
	if len(code) == 0 {
		return crypto.Keccak256Hash(nil)
	}

	cap := int64(len(code))

	// step 1: pad code with 0x0 (STOP) so that len(code) % 16 == 0
	// step 2: for every 16 bytes, convert to Fr, so that we get a Fr array
	var length = (len(code) + 15) / 16

	Frs := make([]*big.Int, length)
	ii := 0

	for ii < length-1 {
		Frs[ii] = big.NewInt(0)
		Frs[ii].SetBytes(code[ii*16 : (ii+1)*16])
		ii++
	}

	Frs[ii] = big.NewInt(0)
	bytes := make([]byte, 16)
	copy(bytes, code[ii*16:])
	Frs[ii].SetBytes(bytes)

	// step 3: apply the array onto a sponge process with the current poseidon scheme
	// (3 Frs permutation and 1 Fr for output, so the throughout is 2 Frs)
	// step 4: convert final root Fr to u256 (big-endian representation)
	hash, _ := HashWithCap(Frs, defaultPoseidonChunk, cap)

	codeHash := common.Hash{}
	hash.FillBytes(codeHash[:])

	return codeHash
}
