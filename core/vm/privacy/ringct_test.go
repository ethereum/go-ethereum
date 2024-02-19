package privacy

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/stretchr/testify/assert"

	"github.com/XinFinOrg/XDPoSChain/crypto/secp256k1"
)

func TestSign(t *testing.T) {
	/*for i := 14; i < 15; i++ {
	for j := 14; j < 15; j++ {
		for k := 0; k <= j; k++ {*/
	numRing := 5
	ringSize := 10
	s := 9
	fmt.Println("Generate random ring parameter ")
	rings, privkeys, m, err := GenerateMultiRingParams(numRing, ringSize, s)

	fmt.Println("numRing  ", numRing)
	fmt.Println("ringSize  ", ringSize)
	fmt.Println("index of real one  ", s)

	fmt.Println("Ring  ", rings)
	fmt.Println("privkeys  ", privkeys)
	fmt.Println("m  ", m)

	ringSignature, err := Sign(m, rings, privkeys, s)
	if err != nil {
		t.Error("Failed to create Ring signature")
	}

	sig, err := ringSignature.Serialize()
	if err != nil {
		t.Error("Failed to Serialize input Ring signature")
	}

	deserializedSig, err := Deserialize(sig)
	if err != nil {
		t.Error("Failed to Deserialize Ring signature")
	}
	verified := Verify(deserializedSig, false)

	if !verified {
		t.Error("Failed to verify Ring signature")
	}

}

func TestDeserialize(t *testing.T) {
	numRing := 5
	ringSize := 10
	s := 5
	rings, privkeys, m, err := GenerateMultiRingParams(numRing, ringSize, s)

	ringSignature, err := Sign(m, rings, privkeys, s)
	if err != nil {
		t.Error("Failed to create Ring signature")
	}

	// A normal signature.
	sig, err := ringSignature.Serialize()
	if err != nil {
		t.Error("Failed to Serialize input Ring signature")
	}

	// Modify the serialized signature s.t.
	// the new signature passes the length check
	// but triggers buffer overflow in Deserialize().
	// ringSize: 10 -> 56759212534490939
	// len(sig): 3495 -> 3804
	// 80 + 5 * (56759212534490939*65 + 33) = 18446744073709551616 + 3804
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, 56759212534490939)
	for i := 0; i < 8; i++ {
		sig[i+8] = bs[i]
	}
	tail := make([]byte, 3804-len(sig))
	sig = append(sig, tail...)

	_, err = Deserialize(sig)
	assert.EqualError(t, err, "incorrect ring size, len r: 3804, sig.NumRing: 5 sig.Size: 56759212534490939")
}

func TestVerify1(t *testing.T) {
	numRing := 5
	ringSize := 10
	s := 7

	rings, privkeys, m, err := GenerateMultiRingParams(numRing, ringSize, s)
	if err != nil {
		t.Error("fail to generate rings")
	}

	ringSignature, err := Sign(m, rings, privkeys, s)
	if err != nil {
		t.Error("fail to create ring signature")
	}

	sig, err := ringSignature.Serialize()
	if err != nil {
		t.Error("fail to serialize input ring signature")
	}

	deserializedSig, err := Deserialize(sig)
	if err != nil {
		t.Error("fail to deserialize ring signature")
	}

	assert.True(t, Verify(deserializedSig, false), "Verify should return true")
}

func TestDeserialize2(t *testing.T) {
	numRing := 5
	ringSize := 10
	s := 7

	rings, privkeys, m, err := GenerateMultiRingParams(numRing, ringSize, s)
	if err != nil {
		t.Error("fail to generate rings")
	}

	ringSignature, err := Sign(m, rings, privkeys, s)
	if err != nil {
		t.Error("fail to create ring signature")
	}

	// change one sig to the scalar field
	ringSignature.S[0][0] = curve.Params().N

	sig, err := ringSignature.Serialize()
	if err != nil {
		t.Error("fail to serialize input ring signature")
	}

	_, err = Deserialize(sig)
	assert.EqualError(t, err, "failed to deserialize, invalid ring signature")
}

func TestPadTo32Bytes(t *testing.T) {
	arr := [44]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34}

	// test input slice is longer than 32 bytes
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[0:]), arr[0:32]), "Test PadTo32Bytes longer than 32 bytes #1")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[1:]), arr[1:33]), "Test PadTo32Bytes longer than 32 bytes #2")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[2:]), arr[2:34]), "Test PadTo32Bytes longer than 32 bytes #3")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[3:]), arr[3:35]), "Test PadTo32Bytes longer than 32 bytes #4")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[4:]), arr[4:36]), "Test PadTo32Bytes longer than 32 bytes #5")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[5:]), arr[5:37]), "Test PadTo32Bytes longer than 32 bytes #6")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[6:]), arr[6:38]), "Test PadTo32Bytes longer than 32 bytes #7")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[7:]), arr[7:39]), "Test PadTo32Bytes longer than 32 bytes #8")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[8:]), arr[8:40]), "Test PadTo32Bytes longer than 32 bytes #9")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[9:]), arr[9:41]), "Test PadTo32Bytes longer than 32 bytes #10")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:]), arr[10:42]), "Test PadTo32Bytes longer than 32 bytes #11")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[11:]), arr[11:43]), "Test PadTo32Bytes longer than 32 bytes #12")

	// test input slice is equal 32 bytes
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[0:32]), arr[0:32]), "Test PadTo32Bytes equal 32 bytes #1")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[1:33]), arr[1:33]), "Test PadTo32Bytes equal 32 bytes #2")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[2:34]), arr[2:34]), "Test PadTo32Bytes equal 32 bytes #3")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[3:35]), arr[3:35]), "Test PadTo32Bytes equal 32 bytes #4")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[4:36]), arr[4:36]), "Test PadTo32Bytes equal 32 bytes #5")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[5:37]), arr[5:37]), "Test PadTo32Bytes equal 32 bytes #6")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[6:38]), arr[6:38]), "Test PadTo32Bytes equal 32 bytes #7")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[7:39]), arr[7:39]), "Test PadTo32Bytes equal 32 bytes #8")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[8:40]), arr[8:40]), "Test PadTo32Bytes equal 32 bytes #9")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[9:41]), arr[9:41]), "Test PadTo32Bytes equal 32 bytes #10")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:42]), arr[10:42]), "Test PadTo32Bytes equal 32 bytes #11")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[11:43]), arr[11:43]), "Test PadTo32Bytes equal 32 bytes #12")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[12:44]), arr[12:44]), "Test PadTo32Bytes equal 32 bytes #13")

	// test input slice is shorter than 32 bytes
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:32]), arr[0:32]), "Test PadTo32Bytes shorter than 32 bytes #1")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:33]), arr[1:33]), "Test PadTo32Bytes shorter than 32 bytes #2")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:34]), arr[2:34]), "Test PadTo32Bytes shorter than 32 bytes #3")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:35]), arr[3:35]), "Test PadTo32Bytes shorter than 32 bytes #4")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:36]), arr[4:36]), "Test PadTo32Bytes shorter than 32 bytes #5")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:37]), arr[5:37]), "Test PadTo32Bytes shorter than 32 bytes #6")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:38]), arr[6:38]), "Test PadTo32Bytes shorter than 32 bytes #7")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:39]), arr[7:39]), "Test PadTo32Bytes shorter than 32 bytes #8")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:40]), arr[8:40]), "Test PadTo32Bytes shorter than 32 bytes #9")
	assert.True(t, bytes.Equal(PadTo32Bytes(arr[10:41]), arr[9:41]), "Test PadTo32Bytes shorter than 32 bytes #10")
}

func TestCurveAddNegative(t *testing.T) {
	curve := crypto.S256().(*secp256k1.BitCurve)

	x1, y1 := curve.ScalarBaseMult(new(big.Int).SetUint64(uint64(2)).Bytes())
	fmt.Printf("Point(%x, %x)\n", x1, y1)

	x2 := x1
	y2 := new(big.Int).Neg(y1) // negative of point (x1,y1)

	x3, y3 := curve.Add(x1, y1, x2, y2)
	fmt.Printf("Output is Point(%x, %x)\n", x3, y3)

	x0 := new(big.Int).SetUint64(uint64(0))
	y0 := new(big.Int).SetUint64(uint64(0)) // infinity

	if (x3.Cmp(x0) == 0) && (y3.Cmp(y0) == 0) {
		// fmt.Printf("Correct, add negative of self should yield (0,0)")
	} else {
		t.Error("Incorrect, add negative of self did not yield (0,0)")
	}
}

func TestCurveAddZero(t *testing.T) {
	// curve := crypto.S256()
	curve := crypto.S256().(*secp256k1.BitCurve)

	x1, y1 := curve.ScalarBaseMult(new(big.Int).SetUint64(uint64(1)).Bytes())
	fmt.Printf("Point(%x, %x)\n", x1, y1)

	x0 := new(big.Int).SetUint64(uint64(0))
	y0 := new(big.Int).SetUint64(uint64(0)) // infinity
	fmt.Printf("Is point (%d,%d) on the curve: %t \n", x0, y0, curve.IsOnCurve(x0, y0))

	x2, y2 := curve.Add(x1, y1, x0, y0)
	fmt.Printf("Output is Point(%x, %x)\n", x2, y2)

	if (x1.Cmp(x2) == 0) && (y1.Cmp(y2) == 0) {
		// fmt.Printf("Correct, Point on curve is the same after Zero addition\n")
	} else {
		t.Error("Incorrect, Point on curve changed after Zero addition\n")
	}
}

func TestOnCurveVerify(t *testing.T) {
	numRing := 5
	ringSize := 10
	s := 5
	rings, privkeys, m, err := GenerateMultiRingParams(numRing, ringSize, s)
	ringSignature, err := Sign(m, rings, privkeys, s)
	if err != nil {
		t.Error("Failed to create Ring signature")
	}

	valid := Verify(ringSignature, false)
	if !valid {
		t.Error("Incorrect, unmodified ringSignature should be valid")
	}

	ringsModified := ringSignature.Ring
	ringsModified[0][0].X = big.NewInt(1)
	ringsModified[0][0].Y = big.NewInt(1)
	valid = Verify(ringSignature, false)
	if valid {
		t.Error("Incorrect, modified ringSignature should be invalid")
	}
}

func TestOnCurveDeserialize(t *testing.T) {
	numRing := 5
	ringSize := 10
	s := 5
	rings, privkeys, m, err := GenerateMultiRingParams(numRing, ringSize, s)
	ringSignature, err := Sign(m, rings, privkeys, s)
	if err != nil {
		t.Error("Failed to create Ring signature")
	}

	sig, err := ringSignature.Serialize()
	if err != nil {
		t.Error("Failed to Serialize input Ring signature")
	}
	_, err = Deserialize(sig)
	if err != nil {
		t.Error("Failed to Deserialize")
	}

	ringsModified := ringSignature.Ring
	ringsModified[0][0].X = big.NewInt(1)
	ringsModified[0][0].Y = big.NewInt(1)

	sig, err = ringSignature.Serialize()
	if err != nil {
		t.Error("Failed to Serialize input Ring signature")
	}
	_, err = Deserialize(sig)
	assert.EqualError(t, err, "failed to deserialize, invalid ring signature")
}

func TestCurveScalarMult(t *testing.T) {
	curve := crypto.S256().(*secp256k1.BitCurve)

	x, y := curve.ScalarBaseMult(curve.Params().N.Bytes())
	if x == nil && y == nil {
		fmt.Println("Scalar multiplication with base point returns nil when scalar is the scalar field")
	}

	x2, y2 := curve.ScalarMult(new(big.Int).SetUint64(uint64(100)), new(big.Int).SetUint64(uint64(2)), curve.Params().N.Bytes())
	if x2 == nil && y2 == nil {
		fmt.Println("Scalar multiplication with a point (not necessarily on curve) returns nil when scalar is the scalar field")
	}
}

func TestNilPointerDereferencePanic(t *testing.T) {
	numRing := 5
	ringSize := 10
	s := 7
	rings, privkeys, m, err := GenerateMultiRingParams(numRing, ringSize, s)

	ringSig, err := Sign(m, rings, privkeys, s)
	if err != nil {
		fmt.Println("Failed to set up")
	}

	ringSig.S[0][0] = curve.Params().N // change one sig to the scalar field

	sig, err := ringSig.Serialize()
	if err != nil {
		t.Error("Failed to Serialize input Ring signature")
	}

	_ , err = Deserialize(sig)
	// Should failed to verify Ring signature as the signature is invalid
	assert.EqualError(t, err, "failed to deserialize, invalid ring signature")
}
