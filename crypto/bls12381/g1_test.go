package bls12381

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
)

func (g *G1) one() *PointG1 {
	return g.New().Set(&g1One)
}

func (g *G1) rand() *PointG1 {
	p := &PointG1{}
	z, _ := new(fe).rand(rand.Reader)
	z6, bz6 := new(fe), new(fe)
	square(z6, z)
	square(z6, z6)
	mul(z6, z6, z)
	mul(z6, z6, z)
	mul(bz6, z6, b)
	for {
		x, _ := new(fe).rand(rand.Reader)
		y := new(fe)
		square(y, x)
		mul(y, y, x)
		add(y, y, bz6)
		if sqrt(y, y) {
			p.Set(&PointG1{*x, *y, *z})
			break
		}
	}
	if !g.IsOnCurve(p) {
		panic("rand point must be on curve")
	}
	if g.InCorrectSubgroup(p) {
		panic("rand point must be out of correct subgroup")
	}
	return p
}

func (g *G1) randCorrect() *PointG1 {
	p := g.ClearCofactor(g.rand())
	if !g.InCorrectSubgroup(p) {
		panic("must be in correct subgroup")
	}
	return p
}

func (g *G1) randAffine() *PointG1 {
	return g.Affine(g.randCorrect())
}

func (g *G1) new() *PointG1 {
	return g.Zero()
}

func TestG1Serialization(t *testing.T) {
	var err error
	g := NewG1()
	zero := g.Zero()
	b0 := g.ToUncompressed(zero)
	p0, err := g.FromUncompressed(b0)
	if err != nil {
		t.Fatal(err)
	}
	if !g.IsZero(p0) {
		t.Fatal("infinity serialization failed")
	}
	b0 = g.ToCompressed(zero)
	p0, err = g.FromCompressed(b0)
	if err != nil {
		t.Fatal(err)
	}
	if !g.IsZero(p0) {
		t.Fatal("infinity serialization failed")
	}
	b0 = g.ToBytes(zero)
	p0, err = g.FromBytes(b0)
	if err != nil {
		t.Fatal(err)
	}
	if !g.IsZero(p0) {
		t.Fatal("infinity serialization failed")
	}
	for i := 0; i < fuz; i++ {
		a := g.randAffine()
		uncompressed := g.ToUncompressed(a)
		b, err := g.FromUncompressed(uncompressed)
		if err != nil {
			t.Fatal(err)
		}
		if !g.Equal(a, b) {
			t.Fatal("serialization failed")
		}
		compressed := g.ToCompressed(b)
		a, err = g.FromCompressed(compressed)
		if err != nil {
			t.Fatal(err)
		}
		if !g.Equal(a, b) {
			t.Fatal("serialization failed")
		}
	}
	for i := 0; i < fuz; i++ {
		a := g.randAffine()
		uncompressed := g.ToBytes(a)
		b, err := g.FromBytes(uncompressed)
		if err != nil {
			t.Fatal(err)
		}
		if !g.Equal(a, b) {
			t.Fatal("serialization failed")
		}
	}
}

func TestG1IsOnCurve(t *testing.T) {
	g := NewG1()
	zero := g.Zero()
	if !g.IsOnCurve(zero) {
		t.Fatal("zero must be on curve")
	}
	one := new(fe).one()
	p := &PointG1{*one, *one, *one}
	if g.IsOnCurve(p) {
		t.Fatal("(1, 1) is not on curve")
	}
}

func TestG1BatchAffine(t *testing.T) {
	n := 20
	g := NewG1()
	points0 := make([]*PointG1, n)
	points1 := make([]*PointG1, n)
	for i := 0; i < n; i++ {
		points0[i] = g.rand()
		points1[i] = g.New().Set(points0[i])
		if g.IsAffine(points0[i]) {
			t.Fatal("expect non affine point")
		}
	}
	g.AffineBatch(points0)
	for i := 0; i < n; i++ {
		if !g.Equal(points0[i], points1[i]) {
			t.Fatal("batch affine failed")
		}
	}
}

func TestG1AdditiveProperties(t *testing.T) {
	g := NewG1()
	t0, t1 := g.New(), g.New()
	zero := g.Zero()
	for i := 0; i < fuz; i++ {
		a, b := g.rand(), g.rand()
		g.Add(t0, a, zero)
		if !g.Equal(t0, a) {
			t.Fatal("a + 0 == a")
		}
		g.Add(t0, zero, zero)
		if !g.Equal(t0, zero) {
			t.Fatal("0 + 0 == 0")
		}
		g.Sub(t0, a, zero)
		if !g.Equal(t0, a) {
			t.Fatal("a - 0 == a")
		}
		g.Sub(t0, zero, zero)
		if !g.Equal(t0, zero) {
			t.Fatal("0 - 0 == 0")
		}
		g.Neg(t0, zero)
		if !g.Equal(t0, zero) {
			t.Fatal("- 0 == 0")
		}
		g.Sub(t0, zero, a)
		g.Neg(t0, t0)
		if !g.Equal(t0, a) {
			t.Fatal(" - (0 - a) == a")
		}
		g.Double(t0, zero)
		if !g.Equal(t0, zero) {
			t.Fatal("2 * 0 == 0")
		}
		g.Double(t0, a)
		g.Sub(t0, t0, a)
		if !g.Equal(t0, a) || !g.IsOnCurve(t0) {
			t.Fatal(" (2 * a) - a == a")
		}
		g.Add(t0, a, b)
		g.Add(t1, b, a)
		if !g.Equal(t0, t1) {
			t.Fatal("a + b == b + a")
		}
		g.Sub(t0, a, b)
		g.Sub(t1, b, a)
		g.Neg(t1, t1)
		if !g.Equal(t0, t1) {
			t.Fatal("a - b == - ( b - a )")
		}
		c := g.rand()
		g.Add(t0, a, b)
		g.Add(t0, t0, c)
		g.Add(t1, a, c)
		g.Add(t1, t1, b)
		if !g.Equal(t0, t1) {
			t.Fatal("(a + b) + c == (a + c ) + b")
		}
		g.Sub(t0, a, b)
		g.Sub(t0, t0, c)
		g.Sub(t1, a, c)
		g.Sub(t1, t1, b)
		if !g.Equal(t0, t1) {
			t.Fatal("(a - b) - c == (a - c) -b")
		}
	}
}

func TestG1MixedAdd(t *testing.T) {
	g := NewG1()
	for i := 0; i < fuz; i++ {
		a, b := g.rand(), g.rand()
		if g.IsAffine(a) || g.IsAffine(b) {
			t.Fatal("expect non affine points")
		}
		bAffine := g.New().Set(b)
		g.Affine(bAffine)
		r0, r1 := g.New(), g.New()
		g.Add(r0, a, b)
		g.AddMixed(r1, a, bAffine)
		if !g.Equal(r0, r1) {
			t.Fatal("mixed addition failed")
		}
		aAffine := g.New().Set(a)
		g.Affine(aAffine)
		g.AddMixed(r0, a, aAffine)
		g.Double(r1, a)
		if !g.Equal(r0, r1) {
			t.Fatal("mixed addition must double where points are equal")
		}
	}
}

func TestG1MultiplicationCross(t *testing.T) {
	g := NewG1()
	for i := 0; i < fuz; i++ {

		a := g.randCorrect()
		s, _ := new(Fr).Rand(rand.Reader)
		sBig := s.ToBig()
		res0, res1, res2, res3, res4 := g.New(), g.New(), g.New(), g.New(), g.New()

		g.mulScalar(res0, a, s)
		g.glvMulFr(res1, a, s)
		g.glvMulBig(res2, a, sBig)
		g.wnafMulFr(res3, a, s)
		g.wnafMulBig(res4, a, sBig)

		if !g.Equal(res0, res1) {
			t.Fatal("cross multiplication failed (glv, fr)", i)
		}
		if !g.Equal(res0, res2) {
			t.Fatal("cross multiplication failed (glv, big)", i)
		}
		if !g.Equal(res0, res3) {
			t.Fatal("cross multiplication failed (wnaf, fr)", i)
		}
		if !g.Equal(res0, res4) {
			t.Fatal("cross multiplication failed (wnaf, big)", i)
		}
	}
}

func TestG1MultiplicativeProperties(t *testing.T) {
	g := NewG1()
	t0, t1 := g.New(), g.New()
	zero := g.Zero()
	for i := 0; i < fuz; i++ {
		a := g.randCorrect()
		s1, _ := new(Fr).Rand(rand.Reader)
		s2, _ := new(Fr).Rand(rand.Reader)
		s3, _ := new(Fr).Rand(rand.Reader)
		sone := &Fr{1}
		g.MulScalar(t0, zero, s1)
		if !g.Equal(t0, zero) {
			t.Fatal(" 0 ^ s == 0")
		}
		g.MulScalar(t0, a, sone)
		if !g.Equal(t0, a) {
			t.Fatal(" a ^ 1 == a")
		}
		g.MulScalar(t0, zero, s1)
		if !g.Equal(t0, zero) {
			t.Fatal(" 0 ^ s == a")
		}
		g.MulScalar(t0, a, s1)
		g.MulScalar(t0, t0, s2)
		s3.Mul(s1, s2)
		g.MulScalar(t1, a, s3)
		if !g.Equal(t0, t1) {
			t.Fatal(" (a ^ s1) ^ s2 == a ^ (s1 * s2)")
		}
		g.MulScalar(t0, a, s1)
		g.MulScalar(t1, a, s2)
		g.Add(t0, t0, t1)
		s3.Add(s1, s2)
		g.MulScalar(t1, a, s3)
		if !g.Equal(t0, t1) {
			t.Fatal(" (a ^ s1) + (a ^ s2) == a ^ (s1 + s2)")
		}
	}
}

func TestZKCryptoVectorsG1UncompressedValid(t *testing.T) {
	data, err := ioutil.ReadFile("tests/g1_uncompressed_valid_test_vectors.dat")
	if err != nil {
		panic(err)
	}
	g := NewG1()
	p1 := g.Zero()
	for i := 0; i < 1000; i++ {
		vector := data[i*2*fpByteSize : (i+1)*2*fpByteSize]
		p2, err := g.FromUncompressed(vector)
		if err != nil {
			t.Fatal("decoing fails", err, i)
		}
		uncompressed := g.ToUncompressed(p2)
		if !bytes.Equal(vector, uncompressed) || !g.Equal(p1, p2) {
			t.Fatal("serialization failed")
		}

		g.Add(p1, p1, &g1One)
	}
}

func TestZKCryptoVectorsG1CompressedValid(t *testing.T) {
	data, err := ioutil.ReadFile("tests/g1_compressed_valid_test_vectors.dat")
	if err != nil {
		panic(err)
	}
	g := NewG1()
	p1 := g.Zero()
	for i := 0; i < 1000; i++ {
		vector := data[i*fpByteSize : (i+1)*fpByteSize]
		p2, err := g.FromCompressed(vector)
		if err != nil {
			t.Fatal("decoing fails", err, i)
		}
		compressed := g.ToCompressed(p2)
		if !bytes.Equal(vector, compressed) || !g.Equal(p1, p2) {
			t.Fatal("serialization failed")
		}
		g.Add(p1, p1, &g1One)
	}
}

func TestG1MultiExpExpected(t *testing.T) {
	g := NewG1()
	one := g.one()
	var scalars [2]*Fr
	var bases [2]*PointG1
	scalars[0] = &Fr{2}
	scalars[1] = &Fr{3}
	bases[0], bases[1] = new(PointG1).Set(one), new(PointG1).Set(one)
	expected, result := g.New(), g.New()
	g.mulScalar(expected, one, &Fr{5})
	_, _ = g.MultiExp(result, bases[:], scalars[:])
	if !g.Equal(expected, result) {
		t.Fatal("multi-exponentiation failed")
	}
}

func TestG1MultiExpBigExpected(t *testing.T) {
	g := NewG1()
	one := g.one()
	var scalars [2]*big.Int
	var bases [2]*PointG1
	scalars[0] = big.NewInt(2)
	scalars[1] = big.NewInt(3)
	bases[0], bases[1] = new(PointG1).Set(one), new(PointG1).Set(one)
	expected, result := g.New(), g.New()
	g.mulScalarBig(expected, one, big.NewInt(5))
	_, _ = g.MultiExpBig(result, bases[:], scalars[:])
	if !g.Equal(expected, result) {
		t.Fatal("multi-exponentiation failed")
	}
}

func TestG1MultiExpBig(t *testing.T) {
	g := NewG1()
	for n := 1; n < 1024+1; n = n * 2 {
		bases := make([]*PointG1, n)
		scalars := make([]*big.Int, n)
		var err error
		for i := 0; i < n; i++ {
			scalars[i], err = rand.Int(rand.Reader, qBig)
			if err != nil {
				t.Fatal(err)
			}
			bases[i] = g.randAffine()
		}
		expected, tmp := g.New(), g.New()
		for i := 0; i < n; i++ {
			g.mulScalarBig(tmp, bases[i], scalars[i])
			g.Add(expected, expected, tmp)
		}
		result := g.New()
		_, _ = g.MultiExpBig(result, bases, scalars)
		if !g.Equal(expected, result) {
			t.Fatal("multi-exponentiation failed")
		}
	}
}

func TestG1MultiExp(t *testing.T) {
	g := NewG1()
	for n := 1; n < 1024+1; n = n * 2 {
		bases := make([]*PointG1, n)
		scalars := make([]*Fr, n)
		var err error
		for i := 0; i < n; i++ {
			scalars[i], err = new(Fr).Rand(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			bases[i] = g.randAffine()
		}
		expected, tmp := g.New(), g.New()
		for i := 0; i < n; i++ {
			g.mulScalar(tmp, bases[i], scalars[i])
			g.Add(expected, expected, tmp)
		}
		result := g.New()
		_, _ = g.MultiExp(result, bases, scalars)
		if !g.Equal(expected, result) {
			t.Fatal("multi-exponentiation failed")
		}
	}
}

func TestG1ClearCofactor(t *testing.T) {
	g := NewG1()
	for i := 0; i < fuz; i++ {
		p0 := g.rand()
		if g.InCorrectSubgroup(p0) {
			t.Fatal("rand point should be out of correct subgroup")
		}
		g.ClearCofactor(p0)
		if !g.InCorrectSubgroup(p0) {
			t.Fatal("cofactor clearing is failed")
		}
	}
}

func TestG1MapToCurve(t *testing.T) {
	for i, v := range []struct {
		u        []byte
		expected []byte
	}{
		{
			u: make([]byte, fpByteSize),
			expected: fromHex(-1,
				"11a9a0372b8f332d5c30de9ad14e50372a73fa4c45d5f2fa5097f2d6fb93bcac592f2e1711ac43db0519870c7d0ea415",
				"092c0f994164a0719f51c24ba3788de240ff926b55f58c445116e8bc6a47cd63392fd4e8e22bdf9feaa96ee773222133",
			),
		},
		{
			u: fromHex(-1, "07fdf49ea58e96015d61f6b5c9d1c8f277146a533ae7fbca2a8ef4c41055cd961fbc6e26979b5554e4b4f22330c0e16d"),
			expected: fromHex(-1,
				"1223effdbb2d38152495a864d78eee14cb0992d89a241707abb03819a91a6d2fd65854ab9a69e9aacb0cbebfd490732c",
				"0f925d61e0b235ecd945cbf0309291878df0d06e5d80d6b84aa4ff3e00633b26f9a7cb3523ef737d90e6d71e8b98b2d5",
			),
		},
		{
			u: fromHex(-1, "1275ab3adbf824a169ed4b1fd669b49cf406d822f7fe90d6b2f8c601b5348436f89761bb1ad89a6fb1137cd91810e5d2"),
			expected: fromHex(-1,
				"179d3fd0b4fb1da43aad06cea1fb3f828806ddb1b1fa9424b1e3944dfdbab6e763c42636404017da03099af0dcca0fd6",
				"0d037cb1c6d495c0f5f22b061d23f1be3d7fe64d3c6820cfcd99b6b36fa69f7b4c1f4addba2ae7aa46fb25901ab483e4",
			),
		},
		{
			u: fromHex(-1, "0e93d11d30de6d84b8578827856f5c05feef36083eef0b7b263e35ecb9b56e86299614a042e57d467fa20948e8564909"),
			expected: fromHex(-1,
				"15aa66c77eded1209db694e8b1ba49daf8b686733afaa7b68c683d0b01788dfb0617a2e2d04c0856db4981921d3004af",
				"0952bb2f61739dd1d201dd0a79d74cda3285403d47655ee886afe860593a8a4e51c5b77a22d2133e3a4280eaaaa8b788",
			),
		},
		{
			u: fromHex(-1, "015a41481155d17074d20be6d8ec4d46632a51521cd9c916e265bd9b47343b3689979b50708c8546cbc2916b86cb1a3a"),
			expected: fromHex(-1,
				"06328ce5106e837935e8da84bd9af473422e62492930aa5f460369baad9545defa468d9399854c23a75495d2a80487ee",
				"094bfdfe3e552447433b5a00967498a3f1314b86ce7a7164c8a8f4131f99333b30a574607e301d5f774172c627fd0bca",
			),
		},
	} {
		g := NewG1()
		p0, err := g.MapToCurve(v.u)
		if err != nil {
			t.Fatal("map to curve fails", i, err)
		}
		if !bytes.Equal(g.ToBytes(p0), v.expected) {
			t.Fatal("map to curve fails", i)
		}
	}
}

func TestG1EncodeToCurve(t *testing.T) {
	domain := []byte("BLS12381G1_XMD:SHA-256_SSWU_NU_TESTGEN")
	for i, v := range []struct {
		msg      []byte
		expected []byte
	}{
		{
			msg: []byte(""),
			expected: fromHex(-1,
				"1223effdbb2d38152495a864d78eee14cb0992d89a241707abb03819a91a6d2fd65854ab9a69e9aacb0cbebfd490732c",
				"0f925d61e0b235ecd945cbf0309291878df0d06e5d80d6b84aa4ff3e00633b26f9a7cb3523ef737d90e6d71e8b98b2d5",
			),
		},
		{
			msg: []byte("abc"),
			expected: fromHex(-1,
				"179d3fd0b4fb1da43aad06cea1fb3f828806ddb1b1fa9424b1e3944dfdbab6e763c42636404017da03099af0dcca0fd6",
				"0d037cb1c6d495c0f5f22b061d23f1be3d7fe64d3c6820cfcd99b6b36fa69f7b4c1f4addba2ae7aa46fb25901ab483e4",
			),
		},
		{
			msg: []byte("abcdef0123456789"),
			expected: fromHex(-1,
				"15aa66c77eded1209db694e8b1ba49daf8b686733afaa7b68c683d0b01788dfb0617a2e2d04c0856db4981921d3004af",
				"0952bb2f61739dd1d201dd0a79d74cda3285403d47655ee886afe860593a8a4e51c5b77a22d2133e3a4280eaaaa8b788",
			),
		},
		{
			msg: []byte("a512_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			expected: fromHex(-1,
				"06328ce5106e837935e8da84bd9af473422e62492930aa5f460369baad9545defa468d9399854c23a75495d2a80487ee",
				"094bfdfe3e552447433b5a00967498a3f1314b86ce7a7164c8a8f4131f99333b30a574607e301d5f774172c627fd0bca",
			),
		},
	} {
		g := NewG1()
		p0, err := g.EncodeToCurve(v.msg, domain)
		if err != nil {
			t.Fatal("encode to point fails", i, err)
		}
		if !bytes.Equal(g.ToBytes(p0), v.expected) {
			t.Fatal("encode to point fails", i)
		}
	}
}

func TestG1HashToCurve(t *testing.T) {
	domain := []byte("BLS12381G1_XMD:SHA-256_SSWU_RO_TESTGEN")
	for i, v := range []struct {
		msg      []byte
		expected []byte
	}{
		{
			msg: []byte(""),
			expected: fromHex(-1,
				"0576730ab036cbac1d95b38dca905586f28d0a59048db4e8778782d89bff856ddef89277ead5a21e2975c4a6e3d8c79e",
				"1273e568bebf1864393c517f999b87c1eaa1b8432f95aea8160cd981b5b05d8cd4a7cf00103b6ef87f728e4b547dd7ae",
			),
		},
		{
			msg: []byte("abc"),
			expected: fromHex(-1,
				"061daf0cc00d8912dac1d4cf5a7c32fca97f8b3bf3f805121888e5eb89f77f9a9f406569027ac6d0e61b1229f42c43d6",
				"0de1601e5ba02cb637c1d35266f5700acee9850796dc88e860d022d7b9e7e3dce5950952e97861e5bb16d215c87f030d",
			),
		},
		{
			msg: []byte("abcdef0123456789"),
			expected: fromHex(-1,
				"0fb3455436843e76079c7cf3dfef75e5a104dfe257a29a850c145568d500ad31ccfe79be9ae0ea31a722548070cf98cd",
				"177989f7e2c751658df1b26943ee829d3ebcf131d8f805571712f3a7527ee5334ecff8a97fc2a50cea86f5e6212e9a57",
			),
		},
		{
			msg: []byte("a512_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			expected: fromHex(-1,
				"0514af2137c1ae1d78d5cb97ee606ea142824c199f0f25ac463a0c78200de57640d34686521d3e9cf6b3721834f8a038",
				"047a85d6898416a0899e26219bca7c4f0fa682717199de196b02b95eaf9fb55456ac3b810e78571a1b7f5692b7c58ab6",
			),
		},
	} {
		g := NewG1()
		p0, err := g.HashToCurve(v.msg, domain)
		if err != nil {
			t.Fatal("hash to point fails", i, err)
		}
		if !bytes.Equal(g.ToBytes(p0), v.expected) {
			t.Fatal("hash to point fails", i)
		}
	}
}

func BenchmarkG1Add(t *testing.B) {
	g := NewG1()
	a, b, c := g.rand(), g.rand(), PointG1{}
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		g.Add(&c, a, b)
	}
}

func BenchmarkG1MulWNAF(t *testing.B) {
	g := NewG1()
	p := new(PointG1).Set(&g1One)
	s, _ := new(Fr).Rand(rand.Reader)
	sBig := s.ToBig()
	res := new(PointG1)
	t.Run("Naive", func(t *testing.B) {
		t.ResetTimer()
		for i := 0; i < t.N; i++ {
			g.mulScalar(res, p, s)
		}
	})
	for i := 1; i < 8; i++ {
		wnafMulWindowG1 = uint(i)
		t.Run(fmt.Sprintf("Fr, window: %d", i), func(t *testing.B) {
			t.ResetTimer()
			for i := 0; i < t.N; i++ {
				g.wnafMulFr(res, p, s)
			}
		})
		t.Run(fmt.Sprintf("Big, window: %d", i), func(t *testing.B) {
			t.ResetTimer()
			for i := 0; i < t.N; i++ {
				g.wnafMulBig(res, p, sBig)
			}
		})
	}
}

func BenchmarkG1MulGLV(t *testing.B) {

	g := NewG1()
	p := new(PointG1).Set(&g1One)
	s, _ := new(Fr).Rand(rand.Reader)
	sBig := s.ToBig()
	res := new(PointG1)
	t.Run("Naive", func(t *testing.B) {
		t.ResetTimer()
		for i := 0; i < t.N; i++ {
			g.mulScalar(res, p, s)
		}
	})
	for i := 1; i < 8; i++ {
		glvMulWindowG1 = uint(i)
		t.Run(fmt.Sprintf("Fr, window: %d", i), func(t *testing.B) {
			t.ResetTimer()
			for i := 0; i < t.N; i++ {
				g.glvMulFr(res, p, s)
			}
		})
		t.Run(fmt.Sprintf("Big, window: %d", i), func(t *testing.B) {
			t.ResetTimer()
			for i := 0; i < t.N; i++ {
				g.glvMulBig(res, p, sBig)
			}
		})
	}
}

func BenchmarkG1MultiExp(t *testing.B) {
	g := NewG1()
	v := func(n int) ([]*PointG1, []*Fr) {
		bases := make([]*PointG1, n)
		scalars := make([]*Fr, n)
		var err error
		for i := 0; i < n; i++ {
			scalars[i], err = new(Fr).Rand(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			bases[i] = g.randAffine()
		}
		return bases, scalars
	}
	for _, i := range []int{2, 10, 100, 1000} {
		t.Run(fmt.Sprint(i), func(t *testing.B) {
			bases, scalars := v(i)
			result := g.New()
			t.ResetTimer()
			for i := 0; i < t.N; i++ {
				_, _ = g.MultiExp(result, bases, scalars)
			}
		})
	}
}

func BenchmarkG1ClearCofactor(t *testing.B) {
	g := NewG1()
	a := g.rand()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		g.ClearCofactor(a)
	}
}

func BenchmarkG1SubgroupCheck(t *testing.B) {
	g := NewG1()
	a := g.rand()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		g.InCorrectSubgroup(a)
	}
}

func BenchmarkG1MapToCurve(t *testing.B) {
	a := fromHex(fpByteSize, "0x1234")
	g := NewG1()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_, err := g.MapToCurve(a)
		if err != nil {
			t.Fatal(err)
		}
	}
}
