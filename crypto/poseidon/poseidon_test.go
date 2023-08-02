// from github.com/iden3/go-iden3-crypto/ff/poseidon
package poseidon

import (
	"math/big"
	"testing"

	"github.com/iden3/go-iden3-crypto/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPoseidonHashFixed(t *testing.T) {
	b0 := big.NewInt(0)
	b1 := big.NewInt(1)
	b2 := big.NewInt(2)
	b3 := big.NewInt(3)
	b4 := big.NewInt(4)
	b5 := big.NewInt(5)
	b6 := big.NewInt(6)
	b7 := big.NewInt(7)
	b8 := big.NewInt(8)
	b9 := big.NewInt(9)
	b10 := big.NewInt(10)
	b11 := big.NewInt(11)
	b12 := big.NewInt(12)
	b13 := big.NewInt(13)
	b14 := big.NewInt(14)
	b15 := big.NewInt(15)
	b16 := big.NewInt(16)

	h, err := HashFixed([]*big.Int{b1})
	assert.Nil(t, err)
	assert.Equal(t,
		"18586133768512220936620570745912940619677854269274689475585506675881198879027",
		h.String())

	h, err = HashFixed([]*big.Int{b1, b2})
	assert.Nil(t, err)
	assert.Equal(t,
		"7853200120776062878684798364095072458815029376092732009249414926327459813530",
		h.String())

	h, err = HashFixed([]*big.Int{b1, b2, b0, b0, b0})
	assert.Nil(t, err)
	assert.Equal(t,
		"1018317224307729531995786483840663576608797660851238720571059489595066344487",
		h.String())
	h, err = HashFixed([]*big.Int{b1, b2, b0, b0, b0, b0})
	assert.Nil(t, err)
	assert.Equal(t,
		"15336558801450556532856248569924170992202208561737609669134139141992924267169",
		h.String())

	h, err = HashFixed([]*big.Int{b3, b4, b0, b0, b0})
	assert.Nil(t, err)
	assert.Equal(t,
		"5811595552068139067952687508729883632420015185677766880877743348592482390548",
		h.String())
	h, err = HashFixed([]*big.Int{b3, b4, b0, b0, b0, b0})
	assert.Nil(t, err)
	assert.Equal(t,
		"12263118664590987767234828103155242843640892839966517009184493198782366909018",
		h.String())

	h, err = HashFixed([]*big.Int{b1, b2, b3, b4, b5, b6})
	assert.Nil(t, err)
	assert.Equal(t,
		"20400040500897583745843009878988256314335038853985262692600694741116813247201",
		h.String())

	h, err = HashFixed([]*big.Int{b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13, b14})
	assert.Nil(t, err)
	assert.Equal(t,
		"8354478399926161176778659061636406690034081872658507739535256090879947077494",
		h.String())

	h, err = HashFixed([]*big.Int{b1, b2, b3, b4, b5, b6, b7, b8, b9, b0, b0, b0, b0, b0})
	assert.Nil(t, err)
	assert.Equal(t,
		"5540388656744764564518487011617040650780060800286365721923524861648744699539",
		h.String())

	h, err = HashFixed([]*big.Int{b1, b2, b3, b4, b5, b6, b7, b8, b9, b0, b0, b0, b0, b0, b0, b0})
	assert.Nil(t, err)
	assert.Equal(t,
		"11882816200654282475720830292386643970958445617880627439994635298904836126497",
		h.String())

	h, err = HashFixed([]*big.Int{b1, b2, b3, b4, b5, b6, b7, b8, b9, b10, b11, b12, b13, b14, b15, b16})
	assert.Nil(t, err)
	assert.Equal(t,
		"9989051620750914585850546081941653841776809718687451684622678807385399211877",
		h.String())

	h, err = HashFixedWithDomain([]*big.Int{b1, b2}, big.NewInt(256))
	assert.Nil(t, err)
	assert.Equal(t,
		"2362370911616048355006851495576377379220050231129891536935411970097789775493",
		h.String())
	h_ref, _ := HashFixed([]*big.Int{b1, b2})
	assert.NotEqual(t, h_ref, h)
}

func TestErrorInputs(t *testing.T) {
	b0 := big.NewInt(0)
	b1 := big.NewInt(1)
	b2 := big.NewInt(2)

	var err error

	_, err = HashFixed([]*big.Int{b1, b2, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0})
	require.Nil(t, err)

	_, err = HashFixed([]*big.Int{b1, b2, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0})
	require.NotNil(t, err)
	assert.Equal(t, "invalid inputs length 17, max 16", err.Error())

	_, err = HashFixed([]*big.Int{b1, b2, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0, b0})
	require.NotNil(t, err)
	assert.Equal(t, "invalid inputs length 18, max 16", err.Error())
}

func TestInputsNotInField(t *testing.T) {
	var err error

	// Very big number, should just return error and not go into endless loop
	b1 := utils.NewIntFromString("12242166908188651009877250812424843524687801523336557272219921456462821518061999999999999999999999999999999999999999999999999999999999") //nolint:lll
	_, err = HashFixed([]*big.Int{b1})
	require.Error(t, err, "inputs values not inside Finite Field")

	// Finite Field const Q, should return error
	b2 := utils.NewIntFromString("21888242871839275222246405745257275088548364400416034343698204186575808495617") //nolint:lll
	_, err = HashFixed([]*big.Int{b2})
	require.Error(t, err, "inputs values not inside Finite Field")
}

func TestPoseidonHash(t *testing.T) {
	ret, err := Hash(nil, 3)
	if err != nil {
		t.Fatal(err)
	}

	// Hash nil for width 3 equal to Hash([0, 0])
	retRef, err := HashFixed([]*big.Int{big.NewInt(0), big.NewInt(0)})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ret, retRef)

	// hash is different for the cap flag
	ret1, err := Hash([]*big.Int{big.NewInt(0)}, 3)
	if err != nil {
		t.Fatal(err)
	}

	assert.NotEqual(t, ret1, retRef)

	ret2, err := HashWithCap([]*big.Int{big.NewInt(0)}, 3, 0)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, ret2, retRef)
}

func BenchmarkPoseidonHash(b *testing.B) {
	b0 := big.NewInt(0)
	b1 := utils.NewIntFromString("12242166908188651009877250812424843524687801523336557272219921456462821518061") //nolint:lll
	b2 := utils.NewIntFromString("12242166908188651009877250812424843524687801523336557272219921456462821518061") //nolint:lll

	bigArray4 := []*big.Int{b1, b2, b0, b0, b0, b0}

	for i := 0; i < b.N; i++ {
		HashFixed(bigArray4) //nolint:errcheck,gosec
	}
}
