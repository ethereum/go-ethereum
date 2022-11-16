package kzg

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
	"github.com/protolambda/go-kzg/bls"
)

var (
	BLSModulus *big.Int
	Domain     [params.FieldElementsPerBlob]*big.Int
	DomainFr   []bls.Fr
)

func initDomain() {
	BLSModulus = new(big.Int)
	BLSModulus.SetString(bls.ModulusStr, 10)

	// ROOT_OF_UNITY = pow(PRIMITIVE_ROOT, (MODULUS - 1) // WIDTH, MODULUS)
	primitiveRoot := big.NewInt(7)
	width := big.NewInt(int64(params.FieldElementsPerBlob))
	exp := new(big.Int).Div(new(big.Int).Sub(BLSModulus, big.NewInt(1)), width)
	rootOfUnity := new(big.Int).Exp(primitiveRoot, exp, BLSModulus)
	DomainFr = make([]bls.Fr, params.FieldElementsPerBlob)
	for i := 0; i < params.FieldElementsPerBlob; i++ {
		// We reverse the bits of the index as specified in https://github.com/ethereum/consensus-specs/pull/3011
		// This effectively permutes the order of the elements in Domain
		reversedIndex := reverseBits(uint64(i), params.FieldElementsPerBlob)
		Domain[i] = new(big.Int).Exp(rootOfUnity, big.NewInt(int64(reversedIndex)), BLSModulus)
		_ = BigToFr(&DomainFr[i], Domain[i])
	}
}

func frToBig(b *big.Int, val *bls.Fr) {
	//b.SetBytes((*kilicbls.Fr)(val).RedToBytes())
	// silly double conversion
	v := bls.FrTo32(val)
	for i := 0; i < 16; i++ {
		v[31-i], v[i] = v[i], v[31-i]
	}
	b.SetBytes(v[:])
}

func BigToFr(out *bls.Fr, in *big.Int) bool {
	var b [32]byte
	inb := in.Bytes()
	copy(b[32-len(inb):], inb)
	// again, we have to double convert as go-kzg only accepts little-endian
	for i := 0; i < 16; i++ {
		b[31-i], b[i] = b[i], b[31-i]
	}
	return bls.FrFrom32(out, b)
}

func blsModInv(out *big.Int, x *big.Int) {
	if len(x.Bits()) != 0 { // if non-zero
		out.ModInverse(x, BLSModulus)
	}
}

// faster than using big.Int ModDiv
func blsDiv(out *big.Int, a *big.Int, b *big.Int) {
	var bInv big.Int
	blsModInv(&bInv, b)
	out.Mod(new(big.Int).Mul(a, &bInv), BLSModulus)
}
