package privacy

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"math/big"
	"os"
	"testing"
)

func TestInnerProductProveLen1(t *testing.T) {
	fmt.Println("TestInnerProductProve1")
	EC = genECPrimeGroupKey(1)
	a := make([]*big.Int, 1)
	b := make([]*big.Int, 1)

	a[0] = big.NewInt(1)

	b[0] = big.NewInt(1)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerify(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
	}
}

func TestInnerProductProveLen2(t *testing.T) {
	fmt.Println("TestInnerProductProve2")
	EC = genECPrimeGroupKey(2)
	a := make([]*big.Int, 2)
	b := make([]*big.Int, 2)

	a[0] = big.NewInt(1)
	a[1] = big.NewInt(1)

	b[0] = big.NewInt(1)
	b[1] = big.NewInt(1)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	fmt.Println("P after two vector commitment with gen ", P)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerify(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
	}
}

func TestInnerProductProveLen4(t *testing.T) {
	fmt.Println("TestInnerProductProve4")
	EC = genECPrimeGroupKey(4)
	a := make([]*big.Int, 4)
	b := make([]*big.Int, 4)

	a[0] = big.NewInt(1)
	a[1] = big.NewInt(1)
	a[2] = big.NewInt(1)
	a[3] = big.NewInt(1)

	b[0] = big.NewInt(1)
	b[1] = big.NewInt(1)
	b[2] = big.NewInt(1)
	b[3] = big.NewInt(1)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerify(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
	}
}

func TestInnerProductProveLen8(t *testing.T) {
	fmt.Println("TestInnerProductProve8")
	EC = genECPrimeGroupKey(8)
	a := make([]*big.Int, 8)
	b := make([]*big.Int, 8)

	a[0] = big.NewInt(1)
	a[1] = big.NewInt(1)
	a[2] = big.NewInt(1)
	a[3] = big.NewInt(1)
	a[4] = big.NewInt(1)
	a[5] = big.NewInt(1)
	a[6] = big.NewInt(1)
	a[7] = big.NewInt(1)

	b[0] = big.NewInt(2)
	b[1] = big.NewInt(2)
	b[2] = big.NewInt(2)
	b[3] = big.NewInt(2)
	b[4] = big.NewInt(2)
	b[5] = big.NewInt(2)
	b[6] = big.NewInt(2)
	b[7] = big.NewInt(2)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerify(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
	}
}

func TestInnerProductProveLen64Rand(t *testing.T) {
	fmt.Println("TestInnerProductProveLen64Rand")
	EC = genECPrimeGroupKey(64)
	a := RandVector(64)
	b := RandVector(64)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerify(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
		fmt.Printf("Values Used: \n\ta = %s\n\tb = %s\n", a, b)
	}

}

func TestInnerProductVerifyFastLen1(t *testing.T) {
	fmt.Println("TestInnerProductProve1")
	EC = genECPrimeGroupKey(1)
	a := make([]*big.Int, 1)
	b := make([]*big.Int, 1)

	a[0] = big.NewInt(2)

	b[0] = big.NewInt(2)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerifyFast(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
	}
}

func TestInnerProductVerifyFastLen2(t *testing.T) {
	fmt.Println("TestInnerProductProve2")
	EC = genECPrimeGroupKey(2)
	a := make([]*big.Int, 2)
	b := make([]*big.Int, 2)

	a[0] = big.NewInt(2)
	a[1] = big.NewInt(3)

	b[0] = big.NewInt(2)
	b[1] = big.NewInt(3)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerifyFast(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
	}
}

func TestInnerProductVerifyFastLen4(t *testing.T) {
	fmt.Println("TestInnerProductProve4")
	EC = genECPrimeGroupKey(4)
	a := make([]*big.Int, 4)
	b := make([]*big.Int, 4)

	a[0] = big.NewInt(1)
	a[1] = big.NewInt(1)
	a[2] = big.NewInt(1)
	a[3] = big.NewInt(1)

	b[0] = big.NewInt(1)
	b[1] = big.NewInt(1)
	b[2] = big.NewInt(1)
	b[3] = big.NewInt(1)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerifyFast(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
	}
}

func TestInnerProductVerifyFastLen8(t *testing.T) {
	fmt.Println("TestInnerProductProve8")
	EC = genECPrimeGroupKey(8)
	a := make([]*big.Int, 8)
	b := make([]*big.Int, 8)

	a[0] = big.NewInt(1)
	a[1] = big.NewInt(1)
	a[2] = big.NewInt(1)
	a[3] = big.NewInt(1)
	a[4] = big.NewInt(1)
	a[5] = big.NewInt(1)
	a[6] = big.NewInt(1)
	a[7] = big.NewInt(1)

	b[0] = big.NewInt(2)
	b[1] = big.NewInt(2)
	b[2] = big.NewInt(2)
	b[3] = big.NewInt(2)
	b[4] = big.NewInt(2)
	b[5] = big.NewInt(2)
	b[6] = big.NewInt(2)
	b[7] = big.NewInt(2)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerifyFast(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
	}
}

func TestInnerProductVerifyFastLen64Rand(t *testing.T) {
	fmt.Println("TestInnerProductProveLen64Rand")
	EC = genECPrimeGroupKey(64)
	a := RandVector(64)
	b := RandVector(64)

	c := InnerProduct(a, b)

	P := TwoVectorPCommitWithGens(EC.BPG, EC.BPH, a, b)

	ipp := InnerProductProve(a, b, c, P, EC.U, EC.BPG, EC.BPH)

	if InnerProductVerifyFast(c, P, EC.U, EC.BPG, EC.BPH, ipp) {
		fmt.Println("Inner Product Proof correct")
	} else {
		t.Error("Inner Product Proof incorrect")
		fmt.Printf("Values Used: \n\ta = %s\n\tb = %s\n", a, b)
	}

}

func TestMRPProveZERO(t *testing.T) {

	mRangeProof, _ := MRPProve([]*big.Int{
		new(big.Int).SetInt64(0),
	})
	mv := MRPVerify(&mRangeProof)
	assert.Equal(t, mv, true, " MRProof incorrect")
}

func TestMRPProve_MAX_2_POW_64(t *testing.T) {

	mRangeProof, _ := MRPProve([]*big.Int{
		new(big.Int).SetUint64(0xFFFFFFFFFFFFFFFF),
	})
	mv := MRPVerify(&mRangeProof)
	assert.Equal(t, mv, true, " MRProof incorrect")
}

func TestMRPProveOutOfSupportedRange(t *testing.T) {

	value, _ := new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFF", 16)
	_, err := MRPProve([]*big.Int{
		value,
	})
	assert.NotNil(t, err, " MRProof incorrect")
}

func TestMRPProve_RANDOM(t *testing.T) {

	mRangeProof, _ := MRPProve(Rand64Vector(1))
	mv := MRPVerify(&mRangeProof)
	assert.Equal(t, mv, true, " MRProof incorrect")

	mRangeProof, _ = MRPProve(Rand64Vector(2))
	mv = MRPVerify(&mRangeProof)
	assert.Equal(t, mv, true, " MRProof incorrect")

	mRangeProof, _ = MRPProve(Rand64Vector(4))
	mv = MRPVerify(&mRangeProof)
	assert.Equal(t, mv, true, " MRProof incorrect")

	mRangeProof, _ = MRPProve(Rand64Vector(8))
	mv = MRPVerify(&mRangeProof)
	assert.Equal(t, mv, true, " MRProof incorrect")
}

func Rand64Vector(l int) []*big.Int {
	result := make([]*big.Int, l)

	for i := 0; i < l; i++ {
		x, err := rand.Int(rand.Reader, big.NewInt(0xFFFFFFFFFFFFFFF))
		check(err)
		result[i] = x
	}

	return result
}

func TestMRPProveValueNumberNotSupported(t *testing.T) {

	_, err := MRPProve(Rand64Vector(3))
	assert.NotNil(t, err, "MRProof incorrect - accepted 3 inputs")

	_, err = MRPProve(Rand64Vector(5))
	assert.NotNil(t, err, "MRProof incorrect - accepted 5 inputs")

	_, err = MRPProve(Rand64Vector(6))
	assert.NotNil(t, err, "MRProof incorrect - accepted 6 inputs")

	_, err = MRPProve(Rand64Vector(7))
	assert.NotNil(t, err, "MRProof incorrect - accepted 7 inputs")

	_, err = MRPProve(Rand64Vector(10))
	assert.NotNil(t, err, "MRProof incorrect - accepted 10 inputs")

	_, err = MRPProve(Rand64Vector(1))
	assert.Nil(t, err, "MRProof incorrect - not accepted 1 inputs")

	_, err = MRPProve(Rand64Vector(2))
	assert.Nil(t, err, "MRProof incorrect - not accepted 2 inputs")

	_, err = MRPProve(Rand64Vector(4))
	fmt.Println(err)
	assert.Nil(t, err, "MRProof incorrect - not accepted 4 inputs")

	_, err = MRPProve(Rand64Vector(8))
	assert.Nil(t, err, "MRProof incorrect - not accepted 8 inputs")
}

type Point struct {
	x string
	y string
}

type IPP struct {
	L          []map[string]string `json:"L"`
	R          []map[string]string `json:"R"`
	A          string              `json:"A"`
	B          string              `json:"B"`
	Challenges []string            `json:"Challenges"`
}

type BulletProof struct {
	Comms []string `json:"Comms"`
	A     string   `json:"A"`
	S     string   `json:"S"`
	Cx    string   `json:"Cx"`
	Cy    string   `json:"Cy"`
	Cz    string   `json:"Cz"`
	T1    string   `json:"T1"`
	T2    string   `json:"T2"`
	Th    string   `json:"Th"`
	Tau   string   `json:"Tau"`
	Mu    string   `json:"Mu"`
	Ipp   IPP      `json:"Ipp"`
}

func parseTestData(filePath string) MultiRangeProof {
	jsonFile, err := os.Open(filePath)

	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize our Users array
	// var result map[string]interface{}
	result := BulletProof{}

	json.Unmarshal([]byte(byteValue), &result)

	fmt.Println("result ", result.Tau)
	fmt.Println("result ", result.Th)
	fmt.Println("result.Ipp ", result.Ipp)

	ipp := result.Ipp

	proof := MultiRangeProof{
		Comms: MapECPointFromHex(result.Comms, ECPointFromHex),
		A:     ECPointFromHex(result.A),
		S:     ECPointFromHex(result.S),
		T1:    ECPointFromHex(result.T1),
		T2:    ECPointFromHex(result.T2),
		Th:    bigIFromHex(result.Th),
		Tau:   bigIFromHex(result.Tau),
		Mu:    bigIFromHex(result.Mu),
		Cx:    bigIFromHex(result.Cx),
		Cy:    bigIFromHex(result.Cy),
		Cz:    bigIFromHex(result.Cz),
		IPP: InnerProdArg{
			L:          MapECPoint(ipp.L, ECPointFromPoint),
			R:          MapECPoint(ipp.R, ECPointFromPoint),
			A:          bigIFromHex(ipp.A),
			B:          bigIFromHex(ipp.B),
			Challenges: MapBigI(ipp.Challenges, bigIFromHex),
		},
	}

	fmt.Println(proof)
	return proof
}

/**
Utils for parsing data from json
*/
func MapBigI(list []string, f func(string) *big.Int) []*big.Int {
	result := make([]*big.Int, len(list))

	for i, item := range list {
		result[i] = f(item)
	}
	return result
}

func MapECPointFromHex(list []string, f func(string) ECPoint) []ECPoint {
	result := make([]ECPoint, len(list))

	for i, item := range list {
		result[i] = f(item)
	}
	return result
}

func MapECPoint(list []map[string]string, f func(Point) ECPoint) []ECPoint {
	result := make([]ECPoint, len(list))

	for i, item := range list {
		result[i] = f(Point{
			x: item["x"],
			y: item["y"],
		})
	}
	return result
}

func bigIFromHex(hex string) *big.Int {
	tmp, _ := new(big.Int).SetString(hex, 16)
	return tmp
}

func ECPointFromHex(hex string) ECPoint {
	Px, _ := new(big.Int).SetString(hex[:64], 16)
	Py, _ := new(big.Int).SetString(hex[64:], 16)
	P := ECPoint{Px, Py}
	return P
}

func ECPointFromPoint(ecpoint Point) ECPoint {
	Px, _ := new(big.Int).SetString(ecpoint.x, 16)
	Py, _ := new(big.Int).SetString(ecpoint.y, 16)
	P := ECPoint{Px, Py}
	return P
}

func TestMRPGeneration(t *testing.T) {
	values := make([]*big.Int, 2)
	values[0] = big.NewInt(1000)
	values[1] = big.NewInt(100000)
	mrp, err := MRPProve(values)
	if err != nil {
		t.Error("failed to generate bulletproof")
	}

	v := MRPVerify(&mrp)
	serilizedBp := mrp.Serialize()

	newMRP := new(MultiRangeProof)
	if newMRP.Deserialize(serilizedBp) != nil {
		t.Error("failed to deserialized bulletproof")
	}

	v = v && MRPVerify(newMRP)

	if !v {
		t.Error("failed to verify bulletproof")
	}
}
