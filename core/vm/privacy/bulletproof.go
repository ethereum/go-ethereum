package privacy

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/log"

	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/btcsuite/btcd/btcec"
)

type Bulletproof struct {
	proofData []byte
}

var EC CryptoParams
var VecLength = 512 // support maximum 8 spending value, each 64 bit (gwei is unit)
var curve elliptic.Curve = crypto.S256()

/*
Implementation of BulletProofs
*/
type ECPoint struct {
	X, Y *big.Int
}

func (p *ECPoint) toECPubKey() *ecdsa.PublicKey {
	return &ecdsa.PublicKey{curve, p.X, p.Y}
}

func toECPoint(key *ecdsa.PublicKey) *ECPoint {
	return &ECPoint{key.X, key.Y}
}

// Equal returns true if points p (self) and p2 (arg) are the same.
func (p ECPoint) Equal(p2 ECPoint) bool {
	if p.X.Cmp(p2.X) == 0 && p2.Y.Cmp(p2.Y) == 0 {
		return true
	}
	return false
}

// Mult multiplies point p by scalar s and returns the resulting point
func (p ECPoint) Mult(s *big.Int) ECPoint {
	modS := new(big.Int).Mod(s, EC.N)
	X, Y := EC.C.ScalarMult(p.X, p.Y, modS.Bytes())
	return ECPoint{X, Y}
}

// Add adds points p and p2 and returns the resulting point
func (p ECPoint) Add(p2 ECPoint) ECPoint {
	X, Y := EC.C.Add(p.X, p.Y, p2.X, p2.Y)
	return ECPoint{X, Y}
}

// Neg returns the additive inverse of point p
func (p ECPoint) Neg() ECPoint {
	negY := new(big.Int).Neg(p.Y)
	modValue := negY.Mod(negY, EC.C.Params().P) // mod P is fine here because we're describing a curve point
	return ECPoint{p.X, modValue}
}

type CryptoParams struct {
	C   elliptic.Curve      // curve
	KC  *btcec.KoblitzCurve // curve
	BPG []ECPoint           // slice of gen 1 for BP
	BPH []ECPoint           // slice of gen 2 for BP
	N   *big.Int            // scalar prime
	U   ECPoint             // a point that is a fixed group element with an unknown discrete-log relative to g,h
	V   int                 // Vector length
	G   ECPoint             // G value for commitments of a single value
	H   ECPoint             // H value for commitments of a single value
}

func (c CryptoParams) Zero() ECPoint {
	return ECPoint{big.NewInt(0), big.NewInt(0)}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

/*
Vector Pedersen Commitment
Given an array of values, we commit the array with different generators
for each element and for each randomness.
*/
func VectorPCommit(value []*big.Int) (ECPoint, []*big.Int) {
	R := make([]*big.Int, EC.V)

	commitment := EC.Zero()

	for i := 0; i < EC.V; i++ {
		r, err := rand.Int(rand.Reader, EC.N)
		check(err)

		R[i] = r

		modValue := new(big.Int).Mod(value[i], EC.N)

		// mG, rH
		lhsX, lhsY := EC.C.ScalarMult(EC.BPG[i].X, EC.BPG[i].Y, modValue.Bytes())
		rhsX, rhsY := EC.C.ScalarMult(EC.BPH[i].X, EC.BPH[i].Y, r.Bytes())

		commitment = commitment.Add(ECPoint{lhsX, lhsY}).Add(ECPoint{rhsX, rhsY})
	}

	return commitment, R
}

/*
Two Vector P Commit
Given an array of values, we commit the array with different generators
for each element and for each randomness.
*/
func TwoVectorPCommit(a []*big.Int, b []*big.Int) ECPoint {
	if len(a) != len(b) {
		log.Debug("TwoVectorPCommit: Arrays not of the same length", "len(a)", len(a), "len(b)", len(b))
	}

	commitment := EC.Zero()

	for i := 0; i < EC.V; i++ {
		commitment = commitment.Add(EC.BPG[i].Mult(a[i])).Add(EC.BPH[i].Mult(b[i]))
	}

	return commitment
}

/*
Vector Pedersen Commitment with Gens
Given an array of values, we commit the array with different generators
for each element and for each randomness.
We also pass in the Generators we want to use
*/
func TwoVectorPCommitWithGens(G, H []ECPoint, a, b []*big.Int) ECPoint {
	if len(G) != len(H) || len(G) != len(a) || len(a) != len(b) {
		log.Debug("TwoVectorPCommitWithGens: Arrays not of the same length", "len(G)", len(G), "len(H)", len(H), "len(a)", len(a), "len(b)", len(b))
	}

	commitment := EC.Zero()

	for i := 0; i < len(G); i++ {
		modA := new(big.Int).Mod(a[i], EC.N)
		modB := new(big.Int).Mod(b[i], EC.N)

		commitment = commitment.Add(G[i].Mult(modA)).Add(H[i].Mult(modB))
	}

	return commitment
}

// The length here always has to be a power of two
func InnerProduct(a []*big.Int, b []*big.Int) *big.Int {
	if len(a) != len(b) {
		log.Debug("InnerProduct: Arrays not of the same length", "len(a)", len(a), "len(b", len(b))
	}

	c := big.NewInt(0)

	for i := range a {
		tmp1 := new(big.Int).Mul(a[i], b[i])
		c = new(big.Int).Add(c, new(big.Int).Mod(tmp1, EC.N))
	}

	return new(big.Int).Mod(c, EC.N)
}

func VectorAdd(v []*big.Int, w []*big.Int) []*big.Int {
	if len(v) != len(w) {
		log.Debug("VectorAdd: Arrays not of the same length", "len(v)", len(v), "len(w)", len(w))
	}
	result := make([]*big.Int, len(v))

	for i := range v {
		result[i] = new(big.Int).Mod(new(big.Int).Add(v[i], w[i]), EC.N)
	}

	return result
}

func VectorHadamard(v, w []*big.Int) []*big.Int {
	if len(v) != len(w) {
		log.Debug("VectorHadamard: Arrays not of the same length", "len(v)", len(v), "len(w)", len(w))
	}

	result := make([]*big.Int, len(v))

	for i := range v {
		result[i] = new(big.Int).Mod(new(big.Int).Mul(v[i], w[i]), EC.N)
	}

	return result
}

func VectorAddScalar(v []*big.Int, s *big.Int) []*big.Int {
	result := make([]*big.Int, len(v))

	for i := range v {
		result[i] = new(big.Int).Mod(new(big.Int).Add(v[i], s), EC.N)
	}

	return result
}

func ScalarVectorMul(v []*big.Int, s *big.Int) []*big.Int {
	result := make([]*big.Int, len(v))

	for i := range v {
		result[i] = new(big.Int).Mod(new(big.Int).Mul(v[i], s), EC.N)
	}

	return result
}

func HashPointsToBytes(points []ECPoint) []byte {
	input := []byte{}

	for index := 0; index < len(points); index++ {
		pointInByte := append(PadTo32Bytes(points[index].X.Bytes()), PadTo32Bytes(points[index].Y.Bytes())...)
		input = append(input, pointInByte...)
	}

	return crypto.Keccak256(input)
}

/*
InnerProd Proof
This stores the argument values
*/
type InnerProdArg struct {
	L []ECPoint
	R []ECPoint
	A *big.Int
	B *big.Int

	Challenges []*big.Int
}

func GenerateNewParams(G, H []ECPoint, x *big.Int, L, R, P ECPoint) ([]ECPoint, []ECPoint, ECPoint) {
	nprime := len(G) / 2

	Gprime := make([]ECPoint, nprime)
	Hprime := make([]ECPoint, nprime)

	xinv := new(big.Int).ModInverse(x, EC.N)

	// Gprime = xinv * G[:nprime] + x*G[nprime:]
	// Hprime = x * H[:nprime] + xinv*H[nprime:]

	for i := range Gprime {
		//fmt.Printf("i: %d && i+nprime: %d\n", i, i+nprime)
		Gprime[i] = G[i].Mult(xinv).Add(G[i+nprime].Mult(x))
		Hprime[i] = H[i].Mult(x).Add(H[i+nprime].Mult(xinv))
	}

	x2 := new(big.Int).Mod(new(big.Int).Mul(x, x), EC.N)
	xinv2 := new(big.Int).ModInverse(x2, EC.N)

	Pprime := L.Mult(x2).Add(P).Add(R.Mult(xinv2)) // x^2 * L + P + xinv^2 * R

	return Gprime, Hprime, Pprime
}

/* Inner Product Argument
Proves that <a,b>=c
This is a building block for BulletProofs
*/
func InnerProductProveSub(proof InnerProdArg, G, H []ECPoint, a []*big.Int, b []*big.Int, u ECPoint, P ECPoint) InnerProdArg {
	if len(a) == 1 {
		// Prover sends a & b
		proof.A = a[0]
		proof.B = b[0]
		return proof
	}

	curIt := int(math.Log2(float64(len(a)))) - 1

	nprime := len(a) / 2
	cl := InnerProduct(a[:nprime], b[nprime:]) // either this line
	cr := InnerProduct(a[nprime:], b[:nprime]) // or this line
	L := TwoVectorPCommitWithGens(G[nprime:], H[:nprime], a[:nprime], b[nprime:]).Add(u.Mult(cl))
	R := TwoVectorPCommitWithGens(G[:nprime], H[nprime:], a[nprime:], b[:nprime]).Add(u.Mult(cr))

	proof.L[curIt] = L
	proof.R[curIt] = R

	// prover sends L & R and gets a challenge
	// LvalInBytes := append(PadTo32Bytes(L.X.Bytes()), PadTo32Bytes(L.Y.Bytes())...)
	// RvalInBytes := append(PadTo32Bytes(R.X.Bytes()), PadTo32Bytes(R.Y.Bytes())...)
	// input := append(LvalInBytes, RvalInBytes...)
	// s256 := crypto.Keccak256(input)
	s256 := HashPointsToBytes([]ECPoint{L, R})

	x := new(big.Int).SetBytes(s256[:])

	proof.Challenges[curIt] = x

	Gprime, Hprime, Pprime := GenerateNewParams(G, H, x, L, R, P)

	xinv := new(big.Int).ModInverse(x, EC.N)

	// or these two lines
	aprime := VectorAdd(
		ScalarVectorMul(a[:nprime], x),
		ScalarVectorMul(a[nprime:], xinv))
	bprime := VectorAdd(
		ScalarVectorMul(b[:nprime], xinv),
		ScalarVectorMul(b[nprime:], x))

	return InnerProductProveSub(proof, Gprime, Hprime, aprime, bprime, u, Pprime)
}

//rpresult.IPP = InnerProductProve(left, right, that, P, EC.U, EC.BPG, HPrime)
func InnerProductProve(a []*big.Int, b []*big.Int, c *big.Int, P, U ECPoint, G, H []ECPoint) InnerProdArg {
	loglen := int(math.Log2(float64(len(a))))

	challenges := make([]*big.Int, loglen+1)
	Lvals := make([]ECPoint, loglen)
	Rvals := make([]ECPoint, loglen)

	runningProof := InnerProdArg{
		Lvals,
		Rvals,
		big.NewInt(0),
		big.NewInt(0),
		challenges}

	// randomly generate an x value from public data
	// input := append(PadTo32Bytes(P.X.Bytes()), PadTo32Bytes(P.Y.Bytes())...)
	// x := crypto.Keccak256(input)
	x := HashPointsToBytes([]ECPoint{P})

	runningProof.Challenges[loglen] = new(big.Int).SetBytes(x[:])

	Pprime := P.Add(U.Mult(new(big.Int).Mul(new(big.Int).SetBytes(x[:]), c)))

	ux := U.Mult(new(big.Int).SetBytes(x[:]))
	//fmt.Printf("Prover Pprime value to run sub off of: %s\n", Pprime)

	return InnerProductProveSub(runningProof, G, H, a, b, ux, Pprime)
}

/* Inner Product Verify
Given a inner product proof, verifies the correctness of the proof
Since we're using the Fiat-Shamir transform, we need to verify all x hash computations,
all g' and h' computations
P : the Pedersen commitment we are verifying is a commitment to the innner product
ipp : the proof
*/
func InnerProductVerify(c *big.Int, P, U ECPoint, G, H []ECPoint, ipp InnerProdArg) bool {
	//fmt.Println("Verifying Inner Product Argument")
	//fmt.Printf("Commitment Value: %s \n", P)
	// s1 := sha256.Sum256([]byte(P.X.String() + P.Y.String()))

	// input := append(PadTo32Bytes(P.X.Bytes()), PadTo32Bytes(P.Y.Bytes())...)
	// s1 := crypto.Keccak256(input)
	s1 := HashPointsToBytes([]ECPoint{P})

	chal1 := new(big.Int).SetBytes(s1[:])
	ux := U.Mult(chal1)
	curIt := len(ipp.Challenges) - 1

	if ipp.Challenges[curIt].Cmp(chal1) != 0 {
		log.Info("Initial Challenge Failed")
		return false
	}

	curIt -= 1

	Gprime := G
	Hprime := H
	Pprime := P.Add(ux.Mult(c)) // line 6 from protocol 1
	//fmt.Printf("New Commitment value with u^cx: %s \n", Pprime)

	for curIt >= 0 {
		Lval := ipp.L[curIt]
		Rval := ipp.R[curIt]

		// prover sends L & R and gets a challenge
		// s256 := sha256.Sum256([]byte(
		// 	Lval.X.String() + Lval.Y.String() +
		// 		Rval.X.String() + Rval.Y.String()))
		// LvalInBytes := append(PadTo32Bytes(Lval.X.Bytes()), PadTo32Bytes(Lval.Y.Bytes())...)
		// RvalInBytes := append(PadTo32Bytes(Rval.X.Bytes()), PadTo32Bytes(Rval.Y.Bytes())...)
		// input := append(LvalInBytes, RvalInBytes...)
		// s256 := crypto.Keccak256(input)
		s256 := HashPointsToBytes([]ECPoint{Lval, Rval})

		chal2 := new(big.Int).SetBytes(s256[:])

		if ipp.Challenges[curIt].Cmp(chal2) != 0 {
			log.Info("Challenge verification failed", "index", strconv.Itoa(curIt))
			return false
		}

		Gprime, Hprime, Pprime = GenerateNewParams(Gprime, Hprime, chal2, Lval, Rval, Pprime)
		curIt -= 1
	}
	ccalc := new(big.Int).Mod(new(big.Int).Mul(ipp.A, ipp.B), EC.N)

	Pcalc1 := Gprime[0].Mult(ipp.A)
	Pcalc2 := Hprime[0].Mult(ipp.B)
	Pcalc3 := ux.Mult(ccalc)
	Pcalc := Pcalc1.Add(Pcalc2).Add(Pcalc3)

	if !Pprime.Equal(Pcalc) {
		return false
	}

	return true
}

/* Inner Product Verify Fast
Given a inner product proof, verifies the correctness of the proof. Does the same as above except
we replace n separate exponentiations with a single multi-exponentiation.
*/

func InnerProductVerifyFast(c *big.Int, P, U ECPoint, G, H []ECPoint, ipp InnerProdArg) bool {
	// input := append(PadTo32Bytes(P.X.Bytes()), PadTo32Bytes(P.Y.Bytes())...)
	// s1 := crypto.Keccak256(input)
	s1 := HashPointsToBytes([]ECPoint{P})

	chal1 := new(big.Int).SetBytes(s1[:])
	ux := U.Mult(chal1)
	curIt := len(ipp.Challenges) - 1

	// check all challenges
	if ipp.Challenges[curIt].Cmp(chal1) != 0 {
		log.Debug("Initial Challenge Failed")
		return false
	}

	for j := curIt - 1; j >= 0; j-- {
		Lval := ipp.L[j]
		Rval := ipp.R[j]

		// prover sends L & R and gets a challenge
		// LvalInBytes := append(PadTo32Bytes(Lval.X.Bytes()), PadTo32Bytes(Lval.Y.Bytes())...)
		// RvalInBytes := append(PadTo32Bytes(Rval.X.Bytes()), PadTo32Bytes(Rval.Y.Bytes())...)
		// input := append(LvalInBytes, RvalInBytes...)
		// s256 := crypto.Keccak256(input)
		s256 := HashPointsToBytes([]ECPoint{Lval, Rval})

		chal2 := new(big.Int).SetBytes(s256[:])

		if ipp.Challenges[j].Cmp(chal2) != 0 {
			log.Debug("Challenge verification failed", "index", strconv.Itoa(j))
			return false
		}
	}
	// begin computing

	curIt -= 1
	Pprime := P.Add(ux.Mult(c)) // line 6 from protocol 1

	tmp1 := EC.Zero()
	for j := curIt; j >= 0; j-- {
		x2 := new(big.Int).Exp(ipp.Challenges[j], big.NewInt(2), EC.N)
		x2i := new(big.Int).ModInverse(x2, EC.N)
		//fmt.Println(tmp1)
		tmp1 = ipp.L[j].Mult(x2).Add(ipp.R[j].Mult(x2i)).Add(tmp1)
		//fmt.Println(tmp1)
	}
	rhs := Pprime.Add(tmp1)
	//rhs=P+c*ux + L*xw+R*x2i 公式29

	sScalars := make([]*big.Int, EC.V)
	invsScalars := make([]*big.Int, EC.V)

	for i := 0; i < EC.V; i++ {
		si := big.NewInt(1)
		for j := curIt; j >= 0; j-- {
			// original challenge if the jth bit of i is 1, inverse challenge otherwise
			chal := ipp.Challenges[j]
			if big.NewInt(int64(i)).Bit(j) == 0 {
				chal = new(big.Int).ModInverse(chal, EC.N)
			}
			// fmt.Printf("Challenge raised to value: %d\n", chal)
			si = new(big.Int).Mod(new(big.Int).Mul(si, chal), EC.N)
		}
		//fmt.Printf("Si value: %d\n", si)
		sScalars[i] = si
		invsScalars[i] = new(big.Int).ModInverse(si, EC.N)
	}

	ccalc := new(big.Int).Mod(new(big.Int).Mul(ipp.A, ipp.B), EC.N)
	lhs := TwoVectorPCommitWithGens(G, H, ScalarVectorMul(sScalars, ipp.A), ScalarVectorMul(invsScalars, ipp.B)).Add(ux.Mult(ccalc))

	if !rhs.Equal(lhs) {
		log.Info("IPVerify - Final Commitment checking failed")
		return false
	}

	return true
}

// from here: https://play.golang.org/p/zciRZvD0Gr with a fix
func PadLeft(str, pad string, l int) string {
	strCopy := str
	for len(strCopy) < l {
		strCopy = pad + strCopy
	}

	return strCopy
}

func STRNot(str string) string {
	result := ""

	for _, i := range str {
		if i == '0' {
			result += "1"
		} else {
			result += "0"
		}
	}
	return result
}

func StrToBigIntArray(str string) []*big.Int {
	result := make([]*big.Int, len(str))

	for i := range str {
		t, success := new(big.Int).SetString(string(str[i]), 10)
		if success {
			result[i] = t
		}
	}

	return result
}

func reverse(l []*big.Int) []*big.Int {
	result := make([]*big.Int, len(l))

	for i := range l {
		result[i] = l[len(l)-i-1]
	}

	return result
}

func PowerVector(l int, base *big.Int) []*big.Int {
	result := make([]*big.Int, l)

	for i := 0; i < l; i++ {
		result[i] = new(big.Int).Exp(base, big.NewInt(int64(i)), EC.N)
	}

	return result
}

func RandVector(l int) []*big.Int {
	result := make([]*big.Int, l)

	for i := 0; i < l; i++ {
		x, err := rand.Int(rand.Reader, EC.N)
		check(err)
		result[i] = x
	}

	return result
}

func VectorSum(y []*big.Int) *big.Int {
	result := big.NewInt(0)

	for _, j := range y {
		result = new(big.Int).Mod(new(big.Int).Add(result, j), EC.N)
	}

	return result
}

type RangeProof struct {
	Comm ECPoint
	A    ECPoint
	S    ECPoint
	T1   ECPoint
	T2   ECPoint
	Tau  *big.Int
	Th   *big.Int
	Mu   *big.Int
	IPP  InnerProdArg

	// challenges
	Cy *big.Int
	Cz *big.Int
	Cx *big.Int
}

/*
Delta is a helper function that is used in the range proof
\delta(y, z) = (z-z^2)<1^n, y^n> - z^3<1^n, 2^n>
*/

func Delta(y []*big.Int, z *big.Int) *big.Int {
	result := big.NewInt(0)

	// (z-z^2)<1^n, y^n>
	z2 := new(big.Int).Mod(new(big.Int).Mul(z, z), EC.N)
	t1 := new(big.Int).Mod(new(big.Int).Sub(z, z2), EC.N)
	t2 := new(big.Int).Mod(new(big.Int).Mul(t1, VectorSum(y)), EC.N)

	// z^3<1^n, 2^n>
	z3 := new(big.Int).Mod(new(big.Int).Mul(z2, z), EC.N)
	po2sum := new(big.Int).Sub(new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(EC.V)), EC.N), big.NewInt(1))
	t3 := new(big.Int).Mod(new(big.Int).Mul(z3, po2sum), EC.N)

	result = new(big.Int).Mod(new(big.Int).Sub(t2, t3), EC.N)

	return result
}

// Calculates (aL - z*1^n) + sL*x
func CalculateL(aL, sL []*big.Int, z, x *big.Int) []*big.Int {
	result := make([]*big.Int, len(aL))

	tmp1 := VectorAddScalar(aL, new(big.Int).Neg(z))
	tmp2 := ScalarVectorMul(sL, x)

	result = VectorAdd(tmp1, tmp2)

	return result
}

func CalculateR(aR, sR, y, po2 []*big.Int, z, x *big.Int) []*big.Int {
	if len(aR) != len(sR) || len(aR) != len(y) || len(y) != len(po2) {
		log.Info("CalculateR: Arrays not of the same length")
	}

	result := make([]*big.Int, len(aR))

	z2 := new(big.Int).Exp(z, big.NewInt(2), EC.N)
	tmp11 := VectorAddScalar(aR, z)
	tmp12 := ScalarVectorMul(sR, x)
	tmp1 := VectorHadamard(y, VectorAdd(tmp11, tmp12))
	tmp2 := ScalarVectorMul(po2, z2)

	result = VectorAdd(tmp1, tmp2)

	return result
}

// Calculates (aL - z*1^n) + sL*x
func CalculateLMRP(aL, sL []*big.Int, z, x *big.Int) []*big.Int {
	result := make([]*big.Int, len(aL))

	tmp1 := VectorAddScalar(aL, new(big.Int).Neg(z))
	tmp2 := ScalarVectorMul(sL, x)

	result = VectorAdd(tmp1, tmp2)

	return result
}

func CalculateRMRP(aR, sR, y, zTimesTwo []*big.Int, z, x *big.Int) []*big.Int {
	if len(aR) != len(sR) || len(aR) != len(y) || len(y) != len(zTimesTwo) {
		log.Info("CalculateRMRP: Arrays not of the same length")
	}

	result := make([]*big.Int, len(aR))

	tmp11 := VectorAddScalar(aR, z)
	tmp12 := ScalarVectorMul(sR, x)
	tmp1 := VectorHadamard(y, VectorAdd(tmp11, tmp12))

	result = VectorAdd(tmp1, zTimesTwo)

	return result
}

/*
DeltaMRP is a helper function that is used in the multi range proof
\delta(y, z) = (z-z^2)<1^n, y^n> - \sum_j z^3+j<1^n, 2^n>
*/

func DeltaMRP(y []*big.Int, z *big.Int, m int) *big.Int {
	result := big.NewInt(0)

	// (z-z^2)<1^n, y^n>
	z2 := new(big.Int).Mod(new(big.Int).Mul(z, z), EC.N)
	t1 := new(big.Int).Mod(new(big.Int).Sub(z, z2), EC.N)
	t2 := new(big.Int).Mod(new(big.Int).Mul(t1, VectorSum(y)), EC.N)

	// \sum_j z^3+j<1^n, 2^n>
	// <1^n, 2^n> = 2^n - 1
	po2sum := new(big.Int).Sub(
		new(big.Int).Exp(
			big.NewInt(2), big.NewInt(int64(EC.V/m)), EC.N), big.NewInt(1))
	t3 := big.NewInt(0)

	for j := 0; j < m; j++ {
		zp := new(big.Int).Exp(z, big.NewInt(3+int64(j)), EC.N)
		tmp1 := new(big.Int).Mod(new(big.Int).Mul(zp, po2sum), EC.N)
		t3 = new(big.Int).Mod(new(big.Int).Add(t3, tmp1), EC.N)
	}

	result = new(big.Int).Mod(new(big.Int).Sub(t2, t3), EC.N)

	return result
}

type MultiRangeProof struct {
	Comms []ECPoint
	A     ECPoint
	S     ECPoint
	T1    ECPoint
	T2    ECPoint
	Tau   *big.Int
	Th    *big.Int
	Mu    *big.Int
	IPP   InnerProdArg

	// challenges
	Cy *big.Int
	Cz *big.Int
	Cx *big.Int
}

func serializePointArray(pa []ECPoint, serializeSize bool) []byte {
	ret := []byte{}

	if serializeSize {
		size := make([]byte, 4)
		binary.BigEndian.PutUint32(size, uint32(len(pa)))
		ret = append(ret, size[:]...)
	}

	for i := 0; i < len(pa); i++ {
		sp := SerializeCompressed(pa[i].toECPubKey())
		ret = append(ret, sp[:]...)
	}
	return ret
}

func deserializePointArray(input []byte, numPoint uint32) ([]ECPoint, error) {
	if len(input) <= 4 {
		return []ECPoint{}, errors.New("input data invalid")
	}
	numPointSize := uint32(0)
	if numPoint == 0 {
		numPointSize = 4
		numPoint = binary.BigEndian.Uint32(input[0:4])
	}
	if uint32(len(input)) < (numPointSize + 33*numPoint) {
		return []ECPoint{}, errors.New("input data too short")
	}
	ret := make([]ECPoint, numPoint)
	offset := numPointSize
	for i := 0; i < int(numPoint); i++ {
		compressed := DeserializeCompressed(curve, input[offset:offset+33])
		if compressed == nil {
			return ret, errors.New("invalid input data")
		}
		ret[i] = *toECPoint(compressed)
		offset += 33
	}
	return ret, nil
}

func (ipp *InnerProdArg) Serialize() []byte {
	proof := []byte{}

	spa := serializePointArray(ipp.L, false)
	proof = append(proof, spa[:]...)

	spa = serializePointArray(ipp.R, false)
	proof = append(proof, spa[:]...)

	if ipp.A.Cmp(big.NewInt(0)) < 0 {
		ipp.A.Mod(ipp.A, EC.N)
	}
	sp := PadTo32Bytes(ipp.A.Bytes())
	proof = append(proof, sp[:]...)

	sp = PadTo32Bytes(ipp.B.Bytes())
	proof = append(proof, sp[:]...)

	for i := 0; i < len(ipp.Challenges); i++ {
		sp = PadTo32Bytes(ipp.Challenges[i].Bytes())
		proof = append(proof, sp[:]...)
	}

	return proof
}

func (ipp *InnerProdArg) Deserialize(proof []byte, numChallenges int) error {
	if len(proof) <= 12 {
		return errors.New("proof data too short")
	}
	offset := 0
	L, err := deserializePointArray(proof[:], uint32(numChallenges)-1)
	if err != nil {
		return err
	}
	ipp.L = append(ipp.L, L[:]...)
	offset += len(L) * 33

	R, err := deserializePointArray(proof[offset:], uint32(numChallenges)-1)
	if err != nil {
		return err
	}

	ipp.R = append(ipp.R, R[:]...)
	offset += len(R) * 33

	if len(proof) <= offset+64+4 {
		return errors.New("proof data too short")
	}

	ipp.A = new(big.Int).SetBytes(proof[offset : offset+32])
	offset += 32
	ipp.B = new(big.Int).SetBytes(proof[offset : offset+32])
	offset += 32

	if len(proof) <= (offset + 32*numChallenges) {
		return errors.New("input data too short")
	}
	for i := 0; i < int(numChallenges); i++ {
		ipp.Challenges = append(ipp.Challenges, new(big.Int).SetBytes(proof[offset:offset+32]))
		offset += 32
	}
	return nil
}

func (mrp *MultiRangeProof) Serialize() []byte {
	proof := []byte{}

	serializedPA := serializePointArray(mrp.Comms, true)
	proof = append(proof, serializedPA[:]...)

	sp := SerializeCompressed(mrp.A.toECPubKey())
	proof = append(proof, sp[:]...)

	sp = SerializeCompressed(mrp.S.toECPubKey())
	proof = append(proof, sp[:]...)

	sp = SerializeCompressed(mrp.T1.toECPubKey())
	proof = append(proof, sp[:]...)

	sp = SerializeCompressed(mrp.T2.toECPubKey())
	proof = append(proof, sp[:]...)

	//Tau, Th, Mu
	sp = PadTo32Bytes(mrp.Tau.Bytes())
	proof = append(proof, sp[:]...)

	if mrp.Th.Cmp(big.NewInt(0)) < 0 {
		mrp.Th.Mod(mrp.Th, EC.N)
	}
	sp = PadTo32Bytes(mrp.Th.Bytes())
	proof = append(proof, sp[:]...)

	sp = PadTo32Bytes(mrp.Mu.Bytes())
	proof = append(proof, sp[:]...)

	//challenges
	sp = mrp.IPP.Serialize()
	proof = append(proof, sp[:]...)

	//challenges
	sp = PadTo32Bytes(mrp.Cy.Bytes())
	proof = append(proof, sp[:]...)

	sp = PadTo32Bytes(mrp.Cz.Bytes())
	proof = append(proof, sp[:]...)

	sp = PadTo32Bytes(mrp.Cx.Bytes())
	proof = append(proof, sp[:]...)

	return proof
}

func (mrp *MultiRangeProof) Deserialize(proof []byte) error {
	Cs, err := deserializePointArray(proof[:], 0)
	if err != nil {
		return err
	}
	mrp.Comms = append(mrp.Comms, Cs[:]...)

	offset := 4 + len(Cs)*33

	if len(proof) <= offset+4+4*33+6*32 {
		return errors.New("invalid input data")
	}
	compressed := DeserializeCompressed(curve, proof[offset:offset+33])
	if compressed == nil {
		return errors.New("failed to decode A")
	}
	offset += 33
	mrp.A = *toECPoint(compressed)

	compressed = DeserializeCompressed(curve, proof[offset:offset+33])
	if compressed == nil {
		return errors.New("failed to decode S")
	}
	offset += 33
	mrp.S = *toECPoint(compressed)

	compressed = DeserializeCompressed(curve, proof[offset:offset+33])
	if compressed == nil {
		return errors.New("failed to decode T2")
	}
	offset += 33
	mrp.T1 = *toECPoint(compressed)

	compressed = DeserializeCompressed(curve, proof[offset:offset+33])
	if compressed == nil {
		return errors.New("failed to decode T2")
	}
	offset += 33
	mrp.T2 = *toECPoint(compressed)

	mrp.Tau = new(big.Int).SetBytes(proof[offset : offset+32])
	offset += 32

	mrp.Th = new(big.Int).SetBytes(proof[offset : offset+32])
	offset += 32

	mrp.Mu = new(big.Int).SetBytes(proof[offset : offset+32])
	offset += 32

	numChallenges := int(math.Log2(float64(len(mrp.Comms)*bitsPerValue))) + 1
	mrp.IPP.Deserialize(proof[offset:], numChallenges)
	offset += len(mrp.IPP.L)*33 + len(mrp.IPP.R)*33 + len(mrp.IPP.Challenges)*32 + 2*32

	mrp.Cy = new(big.Int).SetBytes(proof[offset : offset+32])
	offset += 32

	mrp.Cz = new(big.Int).SetBytes(proof[offset : offset+32])
	offset += 32

	mrp.Cx = new(big.Int).SetBytes(proof[offset : offset+32])
	offset += 32

	return nil
}

func pedersenCommitment(gamma *big.Int, value *big.Int) ECPoint {
	return EC.G.Mult(value).Add(EC.H.Mult(gamma))
}

var MAX_64_BITS = new(big.Int).SetUint64(0xFFFFFFFFFFFFFFFF)

/*
MultiRangeProof Prove
Takes in a list of values and provides an aggregate
range proof for all the values.
changes:
 all values are concatenated
 r(x) is computed differently
 tau_x calculation is different
 delta calculation is different
{(g, h \in G, \textbf{V} \in G^m ; \textbf{v, \gamma} \in Z_p^m) :
	V_j = h^{\gamma_j}g^{v_j} \wedge v_j \in [0, 2^n - 1] \forall j \in [1, m]}
*/
var bitsPerValue = 64

func MRPProve(values []*big.Int) (MultiRangeProof, error) {
	var acceptedInputNumber bool

	MRPResult := MultiRangeProof{}

	m := len(values)

	if m == 1 || m == 2 || m == 4 || m == 8 {
		acceptedInputNumber = true
	}

	if !acceptedInputNumber {
		return MultiRangeProof{}, errors.New("Value number is not supported - just 1, 2, 4, 8")
	}

	EC = genECPrimeGroupKey(m * bitsPerValue)

	// we concatenate the binary representation of the values

	PowerOfTwos := PowerVector(bitsPerValue, big.NewInt(2))

	Comms := make([]ECPoint, m)
	gammas := make([]*big.Int, m)
	aLConcat := make([]*big.Int, EC.V)
	aRConcat := make([]*big.Int, EC.V)

	for j := range values {
		v := values[j]
		if v.Cmp(big.NewInt(0)) == -1 {
			return MultiRangeProof{}, errors.New("Value is below range! Not proving")
		}

		if v.Cmp(MAX_64_BITS) == 1 {
			return MultiRangeProof{}, errors.New("Value is above range! Not proving")
		}

		if v.Cmp(new(big.Int).Exp(big.NewInt(2), big.NewInt(int64(bitsPerValue)), EC.N)) == 1 {
			return MultiRangeProof{}, errors.New("Value is above range! Not proving")
		}

		gamma, err := rand.Int(rand.Reader, EC.N)

		check(err)
		Comms[j] = pedersenCommitment(gamma, v)
		gammas[j] = gamma

		// break up v into its bitwise representation
		aL := reverse(StrToBigIntArray(PadLeft(fmt.Sprintf("%b", v), "0", bitsPerValue)))
		aR := VectorAddScalar(aL, big.NewInt(-1))

		for i := range aR {
			aLConcat[bitsPerValue*j+i] = aL[i]
			aRConcat[bitsPerValue*j+i] = aR[i]
		}
	}

	//compare aL, aR
	MRPResult.Comms = Comms

	alpha, err := rand.Int(rand.Reader, EC.N)
	check(err)

	A := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, aLConcat, aRConcat).Add(EC.H.Mult(alpha))
	MRPResult.A = A

	// fmt.Println("Ec.V %+v", EC.V)
	sL := RandVector(EC.V)
	sR := RandVector(EC.V)

	rho, err := rand.Int(rand.Reader, EC.N)
	check(err)

	S := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, sL, sR).Add(EC.H.Mult(rho))
	MRPResult.S = S

	// input := append(PadTo32Bytes(A.X.Bytes()), PadTo32Bytes(A.Y.Bytes())...)
	// chal1s256 := crypto.Keccak256(input)
	chal1s256 := HashPointsToBytes(append(Comms, A))

	// chal1s256 := sha256.Sum256([]byte(A.X.String() + A.Y.String()))
	cy := new(big.Int).SetBytes(chal1s256[:])
	MRPResult.Cy = cy

	// input = append(PadTo32Bytes(S.X.Bytes()), PadTo32Bytes(S.Y.Bytes())...)
	// chal2s256 := crypto.Keccak256(input)
	chal2s256 := HashPointsToBytes(append(Comms, A, S))

	// chal2s256 := sha256.Sum256([]byte(S.X.String() + S.Y.String()))
	cz := new(big.Int).SetBytes(chal2s256[:])
	MRPResult.Cz = cz

	zPowersTimesTwoVec := make([]*big.Int, EC.V)
	for j := 0; j < m; j++ {
		zp := new(big.Int).Exp(cz, big.NewInt(2+int64(j)), EC.N)
		for i := 0; i < bitsPerValue; i++ {
			zPowersTimesTwoVec[j*bitsPerValue+i] = new(big.Int).Mod(new(big.Int).Mul(PowerOfTwos[i], zp), EC.N)
		}
	}

	PowerOfCY := PowerVector(EC.V, cy)
	// fmt.Println(PowerOfCY)
	l0 := VectorAddScalar(aLConcat, new(big.Int).Neg(cz))
	l1 := sL
	r0 := VectorAdd(
		VectorHadamard(
			PowerOfCY,
			VectorAddScalar(aRConcat, cz)),
		zPowersTimesTwoVec)
	r1 := VectorHadamard(sR, PowerOfCY)

	//calculate t0
	vz2 := big.NewInt(0)
	z2 := new(big.Int).Mod(new(big.Int).Mul(cz, cz), EC.N)
	PowerOfCZ := PowerVector(m, cz)
	for j := 0; j < m; j++ {
		vz2 = new(big.Int).Add(vz2,
			new(big.Int).Mul(
				PowerOfCZ[j],
				new(big.Int).Mul(values[j], z2)))
		vz2 = new(big.Int).Mod(vz2, EC.N)
	}

	t0 := new(big.Int).Mod(new(big.Int).Add(vz2, DeltaMRP(PowerOfCY, cz, m)), EC.N)

	t1 := new(big.Int).Mod(new(big.Int).Add(InnerProduct(l1, r0), InnerProduct(l0, r1)), EC.N)
	t2 := InnerProduct(l1, r1)

	// given the t_i values, we can generate commitments to them
	tau1, err := rand.Int(rand.Reader, EC.N)
	check(err)
	tau2, err := rand.Int(rand.Reader, EC.N)
	check(err)

	T1 := pedersenCommitment(tau1, t1) // EC.G.Mult(t1).Add(EC.H.Mult(tau1)) //commitment to t1
	T2 := pedersenCommitment(tau2, t2) //EC.G.Mult(t2).Add(EC.H.Mult(tau2)) //commitment to t2

	MRPResult.T1 = T1
	MRPResult.T2 = T2

	// t1Byte := append(PadTo32Bytes(T1.X.Bytes()), PadTo32Bytes(T1.Y.Bytes())...)
	// t2Byte := append(PadTo32Bytes(T2.X.Bytes()), PadTo32Bytes(T2.Y.Bytes())...)
	// input = append(t1Byte, t2Byte...)
	// chal3s256 := crypto.Keccak256(input)
	chal3s256 := HashPointsToBytes(append(Comms, A, S, T1, T2))

	// chal3s256 := sha256.Sum256([]byte(T1.X.String() + T1.Y.String() + T2.X.String() + T2.Y.String()))
	cx := new(big.Int).SetBytes(chal3s256[:])

	MRPResult.Cx = cx

	left := CalculateLMRP(aLConcat, sL, cz, cx)
	right := CalculateRMRP(aRConcat, sR, PowerOfCY, zPowersTimesTwoVec, cz, cx)

	thatPrime := new(big.Int).Mod( // t0 + t1*x + t2*x^2
		new(big.Int).Add(t0, new(big.Int).Add(new(big.Int).Mul(t1, cx), new(big.Int).Mul(new(big.Int).Mul(cx, cx), t2))), EC.N)

	that := InnerProduct(left, right) // NOTE: BP Java implementation calculates this from the t_i

	// thatPrime and that should be equal
	if thatPrime.Cmp(that) != 0 {
		fmt.Println("Proving -- Uh oh! Two diff ways to compute same value not working")
		fmt.Printf("\tthatPrime = %s\n", thatPrime.String())
		fmt.Printf("\tthat = %s \n", that.String())
	}

	MRPResult.Th = that

	vecRandomnessTotal := big.NewInt(0)
	for j := 0; j < m; j++ {
		zp := new(big.Int).Exp(cz, big.NewInt(2+int64(j)), EC.N)
		tmp1 := new(big.Int).Mul(gammas[j], zp)
		vecRandomnessTotal = new(big.Int).Mod(new(big.Int).Add(vecRandomnessTotal, tmp1), EC.N)
	}
	//fmt.Println(vecRandomnessTotal)
	taux1 := new(big.Int).Mod(new(big.Int).Mul(tau2, new(big.Int).Mul(cx, cx)), EC.N)
	taux2 := new(big.Int).Mod(new(big.Int).Mul(tau1, cx), EC.N)
	taux := new(big.Int).Mod(new(big.Int).Add(taux1, new(big.Int).Add(taux2, vecRandomnessTotal)), EC.N)

	MRPResult.Tau = taux

	mu := new(big.Int).Mod(new(big.Int).Add(alpha, new(big.Int).Mul(rho, cx)), EC.N)
	MRPResult.Mu = mu

	HPrime := make([]ECPoint, len(EC.BPH))

	for i := range HPrime {
		HPrime[i] = EC.BPH[i].Mult(new(big.Int).ModInverse(PowerOfCY[i], EC.N))
	}

	P := TwoVectorPCommitWithGens(EC.BPG, HPrime, left, right)
	//fmt.Println(P)

	MRPResult.IPP = InnerProductProve(left, right, that, P, EC.U, EC.BPG, HPrime)

	return MRPResult, nil
}

/*
MultiRangeProof Verify
Takes in a MultiRangeProof and verifies its correctness
*/
func MRPVerify(mrp *MultiRangeProof) bool {
	m := len(mrp.Comms)
	EC = genECPrimeGroupKey(m * bitsPerValue)

	//changes:
	// check 1 changes since it includes all commitments
	// check 2 commitment generation is also different

	// verify the challenges
	// input := append(PadTo32Bytes(mrp.A.X.Bytes()), PadTo32Bytes(mrp.A.Y.Bytes())...)
	// chal1s256 := crypto.Keccak256(input)
	chal1s256 := HashPointsToBytes(append(mrp.Comms, mrp.A))

	cy := new(big.Int).SetBytes(chal1s256[:])

	if cy.Cmp(mrp.Cy) != 0 {
		log.Debug("MRPVerify challenge failed!", "Cy", common.Bytes2Hex(mrp.Cy.Bytes()))
		return false
	}

	// input = append(PadTo32Bytes(mrp.S.X.Bytes()), PadTo32Bytes(mrp.S.Y.Bytes())...)
	// chal2s256 := crypto.Keccak256(input)
	chal2s256 := HashPointsToBytes(append(mrp.Comms, mrp.A, mrp.S))

	// chal2s256 := sha256.Sum256([]byte(mrp.S.X.String() + mrp.S.Y.String()))
	cz := new(big.Int).SetBytes(chal2s256[:])
	if cz.Cmp(mrp.Cz) != 0 {
		log.Debug("MRPVerify challenge failed!", "Cz", common.Bytes2Hex(mrp.Cz.Bytes()))
		return false
	}

	// t1Byte := append(PadTo32Bytes(mrp.T1.X.Bytes()), PadTo32Bytes(mrp.T1.Y.Bytes())...)
	// t2Byte := append(PadTo32Bytes(mrp.T2.X.Bytes()), PadTo32Bytes(mrp.T2.Y.Bytes())...)
	// input = append(t1Byte, t2Byte...)
	// chal3s256 := crypto.Keccak256(input)
	chal3s256 := HashPointsToBytes(append(mrp.Comms, mrp.A, mrp.S, mrp.T1, mrp.T2))

	// chal3s256 := sha256.Sum256([]byte(T1.X.String() + T1.Y.String() + T2.X.String() + T2.Y.String()))
	// cx := new(big.Int).SetBytes(chal3s256[:])
	//chal3s256 := sha256.Sum256([]byte(mrp.T1.X.String() + mrp.T1.Y.String() + mrp.T2.X.String() + mrp.T2.Y.String()))

	cx := new(big.Int).SetBytes(chal3s256[:])

	if cx.Cmp(mrp.Cx) != 0 {
		log.Debug("MRPVerify challenge failed!", "Cx", common.Bytes2Hex(mrp.Cx.Bytes()))
		return false
	}

	// given challenges are correct, very range proof
	PowersOfY := PowerVector(EC.V, cy)

	// t_hat * G + tau * H
	lhs := pedersenCommitment(mrp.Tau, mrp.Th) //EC.G.Mult(mrp.Th).Add(EC.H.Mult(mrp.Tau))

	// z^2 * \bold{z}^m \bold{V} + delta(y,z) * G + x * T1 + x^2 * T2
	CommPowers := EC.Zero()
	PowersOfZ := PowerVector(m, cz)
	z2 := new(big.Int).Mod(new(big.Int).Mul(cz, cz), EC.N)

	for j := 0; j < m; j++ {
		CommPowers = CommPowers.Add(mrp.Comms[j].Mult(new(big.Int).Mul(z2, PowersOfZ[j])))
	}

	// TODO i need to change how to calculate the commitment here also ?
	// need to compare and double check with privacy-client lib
	rhs := EC.G.Mult(DeltaMRP(PowersOfY, cz, m)).Add(
		mrp.T1.Mult(cx)).Add(
		mrp.T2.Mult(new(big.Int).Mul(cx, cx))).Add(CommPowers)

	if !lhs.Equal(rhs) {
		log.Debug("Rangeproof failed")
		return false
	}

	tmp1 := EC.Zero()
	zneg := new(big.Int).Mod(new(big.Int).Neg(cz), EC.N)
	for i := range EC.BPG {
		tmp1 = tmp1.Add(EC.BPG[i].Mult(zneg))
	}

	PowerOfTwos := PowerVector(bitsPerValue, big.NewInt(2))
	tmp2 := EC.Zero()
	// generate h'
	HPrime := make([]ECPoint, len(EC.BPH))

	for i := range HPrime {
		mi := new(big.Int).ModInverse(PowersOfY[i], EC.N)
		HPrime[i] = EC.BPH[i].Mult(mi)
	}

	for j := 0; j < m; j++ {
		for i := 0; i < bitsPerValue; i++ {
			val1 := new(big.Int).Mul(cz, PowersOfY[j*bitsPerValue+i])
			zp := new(big.Int).Exp(cz, big.NewInt(2+int64(j)), EC.N)
			val2 := new(big.Int).Mod(new(big.Int).Mul(zp, PowerOfTwos[i]), EC.N)
			tmp2 = tmp2.Add(HPrime[j*bitsPerValue+i].Mult(new(big.Int).Add(val1, val2)))
		}
	}

	// without subtracting this value should equal muCH + l[i]G[i] + r[i]H'[i]
	// we want to make sure that the innerproduct checks out, so we subtract it
	P := mrp.A.Add(mrp.S.Mult(cx)).Add(tmp1).Add(tmp2).Add(EC.H.Mult(mrp.Mu).Neg())
	//fmt.Println(P)

	if !InnerProductVerifyFast(mrp.Th, P, EC.U, EC.BPG, HPrime, mrp.IPP) {
		log.Debug("Range proof failed!")
		return false
	}

	return true
}

// NewECPrimeGroupKey returns the curve (field),
// Generator 1 x&y, Generator 2 x&y, order of the generators
func NewECPrimeGroupKey(n int) CryptoParams {
	curValue := btcec.S256().Gx
	s256 := sha256.New()
	gen1Vals := make([]ECPoint, n)
	gen2Vals := make([]ECPoint, n)
	u := ECPoint{big.NewInt(0), big.NewInt(0)}
	cg := ECPoint{}
	ch := ECPoint{}

	j := 0
	confirmed := 0
	for confirmed < (2*n + 3) {
		s256.Write(new(big.Int).Add(curValue, big.NewInt(int64(j))).Bytes())

		potentialXValue := make([]byte, 33)
		binary.LittleEndian.PutUint32(potentialXValue, 2)
		for i, elem := range s256.Sum(nil) {
			potentialXValue[i+1] = elem
		}

		gen2, err := btcec.ParsePubKey(potentialXValue, btcec.S256())
		if err == nil {
			if confirmed == 2*n { // once we've generated all g and h values then assign this to u
				u = ECPoint{gen2.X, gen2.Y}
				//fmt.Println("Got that U value")
			} else if confirmed == 2*n+1 {
				cg = ECPoint{gen2.X, gen2.Y}

			} else if confirmed == 2*n+2 {
				ch = ECPoint{gen2.X, gen2.Y}
			} else {
				if confirmed%2 == 0 {
					gen1Vals[confirmed/2] = ECPoint{gen2.X, gen2.Y}
					//fmt.Println("new G Value")
				} else {
					gen2Vals[confirmed/2] = ECPoint{gen2.X, gen2.Y}
					//fmt.Println("new H value")
				}
			}
			confirmed += 1
		}
		j += 1
	}

	return CryptoParams{
		btcec.S256(),
		btcec.S256(),
		gen1Vals,
		gen2Vals,
		btcec.S256().N,
		u,
		n,
		cg,
		ch}
}

func genECPrimeGroupKey(n int) CryptoParams {
	// curValue := btcec.S256().Gx
	// s256 := sha256.New()
	gen1Vals := make([]ECPoint, n)
	gen2Vals := make([]ECPoint, n)
	// u := ECPoint{big.NewInt(0), big.NewInt(0)}
	hx, _ := new(big.Int).SetString("50929b74c1a04954b78b4b6035e97a5e078a5a0f28ec96d547bfee9ace803ac0", 16)
	hy, _ := new(big.Int).SetString("31d3c6863973926e049e637cb1b5f40a36dac28af1766968c30c2313f3a38904", 16)
	ch := ECPoint{hx, hy}

	gx, _ := new(big.Int).SetString("79be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798", 16)
	gy, _ := new(big.Int).SetString("483ada7726a3c4655da4fbfc0e1108a8fd17b448a68554199c47d08ffb10d4b8", 16)
	cg := ECPoint{gx, gy}

	i := 0
	for i < n {
		gen2Vals[i] = ch.Mult(
			big.NewInt(int64(i*2 + 1)),
		)
		gen1Vals[i] = cg.Mult(
			big.NewInt(int64(i*2 + 2)),
		)
		i++
	}

	u := cg.Mult(
		big.NewInt(int64(n + 3)),
	)

	return CryptoParams{
		btcec.S256(),
		btcec.S256(),
		gen1Vals,
		gen2Vals,
		btcec.S256().N,
		u,
		n,
		ch,
		cg}
}

func init() {
	// just need the base parameter N, P, G, H for this init, ignore everything else
	EC = CryptoParams{
		btcec.S256(),
		btcec.S256(),
		make([]ECPoint, VecLength),
		make([]ECPoint, VecLength),
		btcec.S256().N,
		ECPoint{big.NewInt(0), big.NewInt(0)},
		VecLength,
		ECPoint{big.NewInt(0), big.NewInt(0)},
		ECPoint{big.NewInt(0), big.NewInt(0)}}
}
