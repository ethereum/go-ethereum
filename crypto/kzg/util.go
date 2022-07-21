package kzg

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
	"github.com/protolambda/go-kzg/bls"
)

var (
	BLSModulus *big.Int
	Domain     [params.FieldElementsPerBlob]*big.Int
	DomainFr   [params.FieldElementsPerBlob]bls.Fr
)

func initDomain() {
	BLSModulus = new(big.Int)
	BLSModulus.SetString("0x73eda753299d7d483339d80809a1d80553bda402fffe5bfeffffffff00000001", 0)

	// ROOT_OF_UNITY = pow(PRIMITIVE_ROOT, (MODULUS - 1) // WIDTH, MODULUS)
	primitiveRoot := big.NewInt(7)
	width := big.NewInt(int64(params.FieldElementsPerBlob))
	exp := new(big.Int).Div(new(big.Int).Sub(BLSModulus, big.NewInt(1)), width)
	rootOfUnity := new(big.Int).Exp(primitiveRoot, exp, BLSModulus)
	for i := 0; i < params.FieldElementsPerBlob; i++ {
		Domain[i] = new(big.Int).Exp(rootOfUnity, big.NewInt(int64(i)), BLSModulus)
		_ = BigToFr(&DomainFr[i], Domain[i])
	}
}

func MatrixLinComb(vectors [][]bls.Fr, scalars []bls.Fr) []bls.Fr {
	r := make([]bls.Fr, len(vectors[0]))
	for i := 0; i < len(vectors); i++ {
		var tmp bls.Fr
		for j := 0; j < len(r); j++ {
			bls.MulModFr(&tmp, &vectors[i][j], &scalars[i])
			bls.AddModFr(&r[j], &r[j], &tmp)
		}
	}
	return r
}

// EvaluatePolyInEvaluationForm evaluates the polynomial using the barycentric formula:
// f(x) = (1 - x**WIDTH) / WIDTH  *  sum_(i=0)^WIDTH  (f(DOMAIN[i]) * DOMAIN[i]) / (x - DOMAIN[i])
func EvaluatePolyInEvaluationForm(yFr *bls.Fr, poly []bls.Fr, x *bls.Fr) {
	if len(poly) != params.FieldElementsPerBlob {
		panic("invalid polynomial length")
	}

	width := big.NewInt(int64(params.FieldElementsPerBlob))
	var inverseWidth big.Int
	blsModInv(&inverseWidth, width)

	var y bls.Fr
	for i := 0; i < params.FieldElementsPerBlob; i++ {
		var num bls.Fr
		bls.MulModFr(&num, &poly[i], &DomainFr[i])

		var denom bls.Fr
		bls.SubModFr(&denom, x, &DomainFr[i])

		var div bls.Fr
		bls.DivModFr(&div, &num, &denom)

		var tmp bls.Fr
		bls.AddModFr(&tmp, &y, &div)
		bls.CopyFr(&y, &tmp)
	}

	xB := new(big.Int)
	frToBig(xB, x)
	powB := new(big.Int).Exp(xB, width, BLSModulus)
	powB.Sub(powB, big.NewInt(1))

	// TODO: add ExpModFr to go-kzg
	var yB big.Int
	frToBig(&yB, &y)
	yB.Mul(&yB, new(big.Int).Mul(powB, &inverseWidth))
	yB.Mod(&yB, BLSModulus)
	bls.SetFr(yFr, yB.String())
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
