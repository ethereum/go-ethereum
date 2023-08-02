// from github.com/iden3/go-iden3-crypto/ff/poseidon

package poseidon

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/iden3/go-iden3-crypto/ff"
	"github.com/iden3/go-iden3-crypto/utils"

	"github.com/scroll-tech/go-ethereum/log"
)

const NROUNDSF = 8 //nolint:golint

var NROUNDSP = []int{56, 57, 56, 60, 60, 63, 64, 63, 60, 66, 60, 65, 70, 60, 64, 68} //nolint:golint

func zero() *ff.Element {
	return ff.NewElement()
}

// exp5 performs x^5 mod p
// https://eprint.iacr.org/2019/458.pdf page 8
func exp5(a *ff.Element) {
	a.Exp(*a, big.NewInt(5)) //nolint:gomnd
}

// exp5state perform exp5 for whole state
func exp5state(state []*ff.Element) {
	for i := 0; i < len(state); i++ {
		exp5(state[i])
	}
}

// ark computes Add-Round Key, from the paper https://eprint.iacr.org/2019/458.pdf
func ark(state []*ff.Element, c []*ff.Element, it int) {
	for i := 0; i < len(state); i++ {
		state[i].Add(state[i], c[it+i])
	}
}

// mix returns [[matrix]] * [vector]
func mix(state []*ff.Element, t int, m [][]*ff.Element) []*ff.Element {
	mul := zero()
	newState := make([]*ff.Element, t)
	for i := 0; i < t; i++ {
		newState[i] = zero()
	}
	for i := 0; i < len(state); i++ {
		newState[i].SetUint64(0)
		for j := 0; j < len(state); j++ {
			mul.Mul(m[j][i], state[j])
			newState[i].Add(newState[i], mul)
		}
	}
	return newState
}

func permute(state []*ff.Element, t int) []*ff.Element {

	nRoundsF := NROUNDSF
	nRoundsP := NROUNDSP[t-2]
	C := c.c[t-2]
	S := c.s[t-2]
	M := c.m[t-2]
	P := c.p[t-2]

	ark(state, C, 0)

	for i := 0; i < nRoundsF/2-1; i++ {
		exp5state(state)
		ark(state, C, (i+1)*t)
		state = mix(state, t, M)
	}
	exp5state(state)
	ark(state, C, (nRoundsF/2)*t)
	state = mix(state, t, P)

	for i := 0; i < nRoundsP; i++ {
		exp5(state[0])
		state[0].Add(state[0], C[(nRoundsF/2+1)*t+i])

		mul := zero()
		newState0 := zero()
		for j := 0; j < len(state); j++ {
			mul.Mul(S[(t*2-1)*i+j], state[j])
			newState0.Add(newState0, mul)
		}

		for k := 1; k < t; k++ {
			mul = zero()
			state[k] = state[k].Add(state[k], mul.Mul(state[0], S[(t*2-1)*i+t+k-1]))
		}
		state[0] = newState0
	}

	for i := 0; i < nRoundsF/2-1; i++ {
		exp5state(state)
		ark(state, C, (nRoundsF/2+1)*t+nRoundsP+i*t)
		state = mix(state, t, M)
	}
	exp5state(state)
	return mix(state, t, M)
}

// for short, use size of inpBI as cap
func Hash(inpBI []*big.Int, width int) (*big.Int, error) {
	return HashWithCap(inpBI, width, int64(len(inpBI)))
}

// Hash using possible sponge specs specified by width (rate from 1 to 15), the size of input is applied as capacity
// (notice we do not include width in the capacity )
func HashWithCap(inpBI []*big.Int, width int, nBytes int64) (*big.Int, error) {
	if width < 2 {
		return nil, fmt.Errorf("width must be ranged from 2 to 16")
	}
	if width-2 > len(NROUNDSP) {
		return nil, fmt.Errorf("invalid inputs width %d, max %d", width, len(NROUNDSP)+1) //nolint:gomnd,lll
	}

	// capflag = nBytes * 2^64
	pow64 := big.NewInt(1)
	pow64.Lsh(pow64, 64)
	capflag := ff.NewElement().SetBigInt(big.NewInt(nBytes))
	capflag.Mul(capflag, ff.NewElement().SetBigInt(pow64))

	// initialize the state
	state := make([]*ff.Element, width)
	state[0] = capflag
	for i := 1; i < width; i++ {
		state[i] = zero()
	}

	rate := width - 1
	i := 0
	// always perform one round of permutation even when input is empty
	for {
		// each round absorb at most `rate` elements from `inpBI`
		for j := 0; j < rate && i < len(inpBI); i, j = i+1, j+1 {
			state[j+1].Add(state[j+1], ff.NewElement().SetBigInt(inpBI[i]))
		}
		state = permute(state, width)
		if i == len(inpBI) {
			break
		}
	}

	// squeeze
	rE := state[0]
	r := big.NewInt(0)
	rE.ToBigIntRegular(r)
	return r, nil

}

// Hash computes the Poseidon hash for the given fixed-size inputs, with specified domain field
func HashFixedWithDomain(inpBI []*big.Int, domain *big.Int) (*big.Int, error) {
	t := len(inpBI) + 1
	if len(inpBI) == 0 || len(inpBI) > len(NROUNDSP) {
		return nil, fmt.Errorf("invalid inputs length %d, max %d", len(inpBI), len(NROUNDSP)) //nolint:gomnd,lll
	}
	if !utils.CheckBigIntArrayInField(inpBI[:]) {
		return nil, errors.New("inputs values not inside Finite Field")
	}
	inp := utils.BigIntArrayToElementArray(inpBI[:])

	state := make([]*ff.Element, t)
	state[0] = ff.NewElement().SetBigInt(domain)
	copy(state[1:], inp[:])

	state = permute(state, t)

	rE := state[0]
	r := big.NewInt(0)
	rE.ToBigIntRegular(r)
	return r, nil
}

// Deprecated HashFixed entry, with domain field is 0
func HashFixed(inpBI []*big.Int) (*big.Int, error) {
	log.Warn("called a deprecated method for poseidon fixed hash", "inputs", inpBI)
	return HashFixedWithDomain(inpBI, big.NewInt(0))
}
