package bls12381

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io/ioutil"
	"math/big"
	"testing"
)

func (g *G2) one() *PointG2 {
	return g.New().Set(&g2One)
}

func (g *G2) rand() *PointG2 {
	p := &PointG2{}
	z, _ := new(fe2).rand(rand.Reader)
	z6, bz6 := new(fe2), new(fe2)
	g.f.square(z6, z)
	g.f.square(z6, z6)
	g.f.mul(z6, z6, z)
	g.f.mul(z6, z6, z)
	g.f.mul(bz6, z6, b2)
	for {
		x, _ := new(fe2).rand(rand.Reader)
		y := new(fe2)
		g.f.square(y, x)
		g.f.mul(y, y, x)
		fp2Add(y, y, bz6)
		if g.f.sqrt(y, y) {
			p.Set(&PointG2{*x, *y, *z})
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

func (g *G2) randCorrect() *PointG2 {
	p := g.ClearCofactor(g.rand())
	if !g.InCorrectSubgroup(p) {
		panic("must be in correct subgroup")
	}
	return p
}

func (g *G2) randAffine() *PointG2 {
	return g.Affine(g.randCorrect())
}

func (g *G2) new() *PointG2 {
	return g.Zero()
}

func TestG2Serialization(t *testing.T) {
	var err error
	g2 := NewG2()
	zero := g2.Zero()
	b0 := g2.ToUncompressed(zero)
	p0, err := g2.FromUncompressed(b0)
	if err != nil {
		t.Fatal(err)
	}
	if !g2.IsZero(p0) {
		t.Fatal("infinity serialization failed")
	}
	b0 = g2.ToCompressed(zero)
	p0, err = g2.FromCompressed(b0)
	if err != nil {
		t.Fatal(err)
	}
	if !g2.IsZero(p0) {
		t.Fatal("infinity serialization failed")
	}
	b0 = g2.ToBytes(zero)
	p0, err = g2.FromBytes(b0)
	if err != nil {
		t.Fatal(err)
	}
	if !g2.IsZero(p0) {
		t.Fatal("infinity serialization failed")
	}
	for i := 0; i < fuz; i++ {
		a := g2.randAffine()
		uncompressed := g2.ToUncompressed(a)
		b, err := g2.FromUncompressed(uncompressed)
		if err != nil {
			t.Fatal(err)
		}
		if !g2.Equal(a, b) {
			t.Fatal("serialization failed")
		}
		compressed := g2.ToCompressed(b)
		a, err = g2.FromCompressed(compressed)
		if err != nil {
			t.Fatal(err)
		}
		if !g2.Equal(a, b) {
			t.Fatal("serialization failed")
		}
	}
	for i := 0; i < fuz; i++ {
		a := g2.rand()
		uncompressed := g2.ToBytes(a)
		b, err := g2.FromBytes(uncompressed)
		if err != nil {
			t.Fatal(err)
		}
		if !g2.Equal(a, b) {
			t.Fatal("serialization failed")
		}
	}
}

func TestG2IsOnCurve(t *testing.T) {
	g := NewG2()
	zero := g.Zero()
	if !g.IsOnCurve(zero) {
		t.Fatal("zero must be on curve")
	}
	one := new(fe2).one()
	p := &PointG2{*one, *one, *one}
	if g.IsOnCurve(p) {
		t.Fatal("(1, 1) is not on curve")
	}
}

func TestG2BatchAffine(t *testing.T) {
	n := 20
	g := NewG2()
	points0 := make([]*PointG2, n)
	points1 := make([]*PointG2, n)
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

func TestG2AdditiveProperties(t *testing.T) {
	g := NewG2()
	t0, t1 := g.New(), g.New()
	zero := g.Zero()
	for i := 0; i < fuz; i++ {
		a, b := g.rand(), g.rand()
		_, _, _ = b, t1, zero
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

func TestG2MixedAdd(t *testing.T) {
	g := NewG2()
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

func TestG2MultiplicationCross(t *testing.T) {
	g := NewG2()
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

func TestG2MultiplicativeProperties(t *testing.T) {
	g := NewG2()
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

func TestZKCryptoVectorsG2UncompressedValid(t *testing.T) {
	data, err := ioutil.ReadFile("tests/g2_uncompressed_valid_test_vectors.dat")
	if err != nil {
		panic(err)
	}
	g := NewG2()
	p1 := g.Zero()
	for i := 0; i < 1000; i++ {
		vector := data[i*192 : (i+1)*192]
		p2, err := g.FromUncompressed(vector)
		if err != nil {
			t.Fatal("decoing fails", err, i)
		}
		uncompressed := g.ToUncompressed(p2)
		if !bytes.Equal(vector, uncompressed) || !g.Equal(p1, p2) {
			t.Fatal("serialization failed")
		}
		g.Add(p1, p1, &g2One)
	}
}

func TestZKCryptoVectorsG2CompressedValid(t *testing.T) {
	data, err := ioutil.ReadFile("tests/g2_compressed_valid_test_vectors.dat")
	if err != nil {
		panic(err)
	}
	g := NewG2()
	p1 := g.Zero()
	for i := 0; i < 1000; i++ {
		vector := data[i*2*fpByteSize : (i+1)*2*fpByteSize]
		p2, err := g.FromCompressed(vector)
		if err != nil {
			t.Fatal("decoing fails", err, i)
		}
		compressed := g.ToCompressed(p2)
		if !bytes.Equal(vector, compressed) || !g.Equal(p1, p2) {
			t.Fatal("serialization failed")
		}

		g.Add(p1, p1, &g2One)
	}
}

func TestG2MultiExpExpected(t *testing.T) {
	g := NewG2()
	one := g.one()
	var scalars [2]*Fr
	var bases [2]*PointG2
	scalars[0] = &Fr{2}
	scalars[1] = &Fr{3}
	bases[0], bases[1] = new(PointG2).Set(one), new(PointG2).Set(one)
	expected, result := g.New(), g.New()
	g.mulScalar(expected, one, &Fr{5})
	_, _ = g.MultiExp(result, bases[:], scalars[:])
	if !g.Equal(expected, result) {
		t.Fatal("multi-exponentiation failed")
	}
}

func TestG2MultiExpBigExpected(t *testing.T) {
	g := NewG2()
	one := g.one()
	var scalars [2]*big.Int
	var bases [2]*PointG2
	scalars[0] = big.NewInt(2)
	scalars[1] = big.NewInt(3)
	bases[0], bases[1] = new(PointG2).Set(one), new(PointG2).Set(one)
	expected, result := g.New(), g.New()
	g.mulScalarBig(expected, one, big.NewInt(5))
	_, _ = g.MultiExpBig(result, bases[:], scalars[:])
	if !g.Equal(expected, result) {
		t.Fatal("multi-exponentiation failed")
	}
}

func TestG2MultiExp(t *testing.T) {
	g := NewG2()
	for n := 1; n < 1024+1; n = n * 2 {
		bases := make([]*PointG2, n)
		scalars := make([]*Fr, n)
		var err error
		for i := 0; i < n; i++ {
			scalars[i], err = new(Fr).Rand(rand.Reader)
			if err != nil {
				t.Fatal(err)
			}
			bases[i] = g.rand()
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

func TestG2MultiExpBig(t *testing.T) {
	g := NewG2()
	for n := 1; n < 1024+1; n = n * 2 {
		bases := make([]*PointG2, n)
		scalars := make([]*big.Int, n)
		var err error
		for i := 0; i < n; i++ {
			scalars[i], err = rand.Int(rand.Reader, qBig)
			if err != nil {
				t.Fatal(err)
			}
			bases[i] = g.rand()
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

func TestG2ClearCofactor(t *testing.T) {
	g := NewG2()
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

func TestG2MapToCurve(t *testing.T) {
	for i, v := range []struct {
		u        []byte
		expected []byte
	}{
		{
			u: make([]byte, 2*fpByteSize),
			expected: fromHex(-1, "0a67d12118b5a35bb02d2e86b3ebfa7e23410db93de39fb06d7025fa95e96ffa428a7a27c3ae4dd4b40bd251ac658892",
				"018320896ec9eef9d5e619848dc29ce266f413d02dd31d9b9d44ec0c79cd61f18b075ddba6d7bd20b7ff27a4b324bfce",
				"04c69777a43f0bda07679d5805e63f18cf4e0e7c6112ac7f70266d199b4f76ae27c6269a3ceebdae30806e9a76aadf5c",
				"0260e03644d1a2c321256b3246bad2b895cad13890cbe6f85df55106a0d334604fb143c7a042d878006271865bc35941",
			),
		},
		{
			u: fromHex(-1,
				"025fbc07711ba267b7e70c82caa70a16fbb1d470ae24ceef307f5e2000751677820b7013ad4e25492dcf30052d3e5eca",
				"0e775d7827adf385b83e20e4445bd3fab21d7b4498426daf3c1d608b9d41e9edb5eda0df022e753b8bb4bc3bb7db4914",
			),
			expected: fromHex(-1,
				"0d4333b77becbf9f9dfa3ca928002233d1ecc854b1447e5a71f751c9042d000f42db91c1d6649a5e0ad22bd7bf7398b8",
				"027e4bfada0b47f9f07e04aec463c7371e68f2fd0c738cd517932ea3801a35acf09db018deda57387b0f270f7a219e4d",
				"0cc76dc777ea0d447e02a41004f37a0a7b1fafb6746884e8d9fc276716ccf47e4e0899548a2ec71c2bdf1a2a50e876db",
				"053674cba9ef516ddc218fedb37324e6c47de27f88ab7ef123b006127d738293c0277187f7e2f80a299a24d84ed03da7",
			),
		},
		{
			u: fromHex(-1,
				"1870a7dbfd2a1deb74015a3546b20f598041bf5d5202997956a94a368d30d3f70f18cdaa1d33ce970a4e16af961cbdcb",
				"045ab31ce4b5a8ba7c4b2851b64f063a66cd1223d3c85005b78e1beee65e33c90ceef0244e45fc45a5e1d6eab6644fdb",
			),
			expected: fromHex(-1,
				"18f0f87b40af67c056915dbaf48534c592524e82c1c2b50c3734d02c0172c80df780a60b5683759298a3303c5d942778",
				"09349f1cb5b2e55489dcd45a38545343451cc30a1681c57acd4fb0a6db125f8352c09f4a67eb7d1d8242cb7d3405f97b",
				"10a2ba341bc689ab947b7941ce6ef39be17acaab067bd32bd652b471ab0792c53a2bd03bdac47f96aaafe96e441f63c0",
				"02f2d9deb2c7742512f5b8230bf0fd83ea42279d7d39779543c1a43b61c885982b611f6a7a24b514995e8a098496b811",
			),
		},
		{
			u: fromHex(-1,
				"088fe329b054db8a6474f21a7fbfdf17b4c18044db299d9007af582c3d5f17d00e56d99921d4b5640fce44b05219b5de",
				"0b6e6135a4cd31ba980ddbd115ac48abef7ec60e226f264d7befe002c165f3a496f36f76dd524efd75d17422558d10b4",
			),
			expected: fromHex(-1,
				"19808ec5930a53c7cf5912ccce1cc33f1b3dcff24a53ce1cc4cba41fd6996dbed4843ccdd2eaf6a0cd801e562718d163",
				"149fe43777d34f0d25430dea463889bd9393bdfb4932946db23671727081c629ebb98a89604f3433fba1c67d356a4af7",
				"04783e391c30c83f805ca271e353582fdf19d159f6a4c39b73acbb637a9b8ac820cfbe2738d683368a7c07ad020e3e33",
				"04c0d6793a766233b2982087b5f4a254f261003ccb3262ea7c50903eecef3e871d1502c293f9e063d7d293f6384f4551",
			),
		},
		{
			u: fromHex(-1,
				"03df16a66a05e4c1188c234788f43896e0565bfb64ac49b9639e6b284cc47dad73c47bb4ea7e677db8d496beb907fbb6",
				"0f45b50647d67485295aa9eb2d91a877b44813677c67c8d35b2173ff3ba95f7bd0806f9ca8a1436b8b9d14ee81da4d7e",
			),
			expected: fromHex(-1,
				"0b8e0094c886487870372eb6264613a6a087c7eb9804fab789be4e47a57b29eb19b1983a51165a1b5eb025865e9fc63a",
				"0804152cbf8474669ad7d1796ab92d7ca21f32d8bed70898a748ed4e4e0ec557069003732fc86866d938538a2ae95552",
				"14c80f068ece15a3936bb00c3c883966f75b4e8d9ddde809c11f781ab92d23a2d1d103ad48f6f3bb158bf3e3a4063449",
				"09e5c8242dd7281ad32c03fe4af3f19167770016255fb25ad9b67ec51d62fade31a1af101e8f6172ec2ee8857662be3a",
			),
		},
	} {
		g := NewG2()
		p0, err := g.MapToCurve(v.u)
		if err != nil {
			t.Fatal("map to curve fails", i, err)
		}
		if !bytes.Equal(g.ToBytes(p0), v.expected) {
			t.Fatal("map to curve fails", i)
		}
	}
}

func TestG2EncodeToCurve(t *testing.T) {
	domain := []byte("BLS12381G2_XMD:SHA-256_SSWU_NU_TESTGEN")
	for i, v := range []struct {
		msg      []byte
		expected []byte
	}{
		{
			msg: []byte(""),
			expected: fromHex(-1,
				"0d4333b77becbf9f9dfa3ca928002233d1ecc854b1447e5a71f751c9042d000f42db91c1d6649a5e0ad22bd7bf7398b8",
				"027e4bfada0b47f9f07e04aec463c7371e68f2fd0c738cd517932ea3801a35acf09db018deda57387b0f270f7a219e4d",
				"0cc76dc777ea0d447e02a41004f37a0a7b1fafb6746884e8d9fc276716ccf47e4e0899548a2ec71c2bdf1a2a50e876db",
				"053674cba9ef516ddc218fedb37324e6c47de27f88ab7ef123b006127d738293c0277187f7e2f80a299a24d84ed03da7",
			),
		},
		{
			msg: []byte("abc"),
			expected: fromHex(-1,
				"18f0f87b40af67c056915dbaf48534c592524e82c1c2b50c3734d02c0172c80df780a60b5683759298a3303c5d942778",
				"09349f1cb5b2e55489dcd45a38545343451cc30a1681c57acd4fb0a6db125f8352c09f4a67eb7d1d8242cb7d3405f97b",
				"10a2ba341bc689ab947b7941ce6ef39be17acaab067bd32bd652b471ab0792c53a2bd03bdac47f96aaafe96e441f63c0",
				"02f2d9deb2c7742512f5b8230bf0fd83ea42279d7d39779543c1a43b61c885982b611f6a7a24b514995e8a098496b811",
			),
		},
		{
			msg: []byte("abcdef0123456789"),
			expected: fromHex(-1,
				"19808ec5930a53c7cf5912ccce1cc33f1b3dcff24a53ce1cc4cba41fd6996dbed4843ccdd2eaf6a0cd801e562718d163",
				"149fe43777d34f0d25430dea463889bd9393bdfb4932946db23671727081c629ebb98a89604f3433fba1c67d356a4af7",
				"04783e391c30c83f805ca271e353582fdf19d159f6a4c39b73acbb637a9b8ac820cfbe2738d683368a7c07ad020e3e33",
				"04c0d6793a766233b2982087b5f4a254f261003ccb3262ea7c50903eecef3e871d1502c293f9e063d7d293f6384f4551",
			),
		},
		{
			msg: []byte("a512_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			expected: fromHex(-1,
				"0b8e0094c886487870372eb6264613a6a087c7eb9804fab789be4e47a57b29eb19b1983a51165a1b5eb025865e9fc63a",
				"0804152cbf8474669ad7d1796ab92d7ca21f32d8bed70898a748ed4e4e0ec557069003732fc86866d938538a2ae95552",
				"14c80f068ece15a3936bb00c3c883966f75b4e8d9ddde809c11f781ab92d23a2d1d103ad48f6f3bb158bf3e3a4063449",
				"09e5c8242dd7281ad32c03fe4af3f19167770016255fb25ad9b67ec51d62fade31a1af101e8f6172ec2ee8857662be3a",
			),
		},
	} {
		g := NewG2()
		p0, err := g.EncodeToCurve(v.msg, domain)
		if err != nil {
			t.Fatal("encode to point fails", i, err)
		}
		if !bytes.Equal(g.ToBytes(p0), v.expected) {
			t.Fatal("encode to point fails x", i)
		}
	}
}

func TestG2HashToCurve(t *testing.T) {
	domain := []byte("BLS12381G2_XMD:SHA-256_SSWU_RO_TESTGEN")
	for i, v := range []struct {
		msg      []byte
		expected []byte
	}{
		{
			msg: []byte(""),
			expected: fromHex(-1,
				"0fbdae26f9f9586a46d4b0b70390d09064ef2afe5c99348438a3c7d9756471e015cb534204c1b6824617a85024c772dc",
				"0a650bd36ae7455cb3fe5d8bb1310594551456f5c6593aec9ee0c03d2f6cb693bd2c5e99d4e23cbaec767609314f51d3",
				"02e5cf8f9b7348428cc9e66b9a9b36fe45ba0b0a146290c3a68d92895b1af0e1f2d9f889fb412670ae8478d8abd4c5aa",
				"0d8d49e7737d8f9fc5cef7c4b8817633103faf2613016cb86a1f3fc29968fe2413e232d9208d2d74a89bf7a48ac36f83",
			),
		},
		{
			msg: []byte("abc"),
			expected: fromHex(-1,
				"03578447618463deb106b60e609c6f7cc446dc6035f84a72801ba17c94cd800583b493b948eff0033f09086fdd7f6175",
				"1953ce6d4267939c7360756d9cca8eb34aac4633ef35369a7dc249445069888e7d1b3f9d2e75fbd468fbcbba7110ea02",
				"0184d26779ae9d4670aca9b267dbd4d3b30443ad05b8546d36a195686e1ccc3a59194aea05ed5bce7c3144a29ec047c4",
				"0882ab045b8fe4d7d557ebb59a63a35ac9f3d312581b509af0f8eaa2960cbc5e1e36bb969b6e22980b5cbdd0787fcf4e",
			),
		},
		{
			msg: []byte("abcdef0123456789"),
			expected: fromHex(-1,
				"195fad48982e186ce3c5c82133aefc9b26d55979b6f530992a8849d4263ec5d57f7a181553c8799bcc83da44847bdc8d",
				"17b461fc3b96a30c2408958cbfa5f5927b6063a8ad199d5ebf2d7cdeffa9c20c85487204804fab53f950b2f87db365aa",
				"005cdf3d984e3391e7e969276fb4bc02323c5924a4449af167030d855acc2600cf3d4fab025432c6d868c79571a95bef",
				"174a3473a3af2d0302b9065e895ca4adba4ece6ce0b41148ba597001abb152f852dd9a96fb45c9de0a43d944746f833e",
			),
		},
		{
			msg: []byte("a512_aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
			expected: fromHex(-1,
				"123b6bd9feeba26dd4ad00f8bfda2718c9700dc093ea5287d7711844644eb981848316d3f3f57d5d3a652c6cdc816aca",
				"0a162306f3b0f2bb326f0c4fb0e1fea020019c3af796dcd1d7264f50ddae94cacf3cade74603834d44b9ab3d5d0a6c98",
				"05483f3b96d9252dd4fc0868344dfaf3c9d145e3387db23fa8e449304fab6a7b6ec9c15f05c0a1ea66ff0efcc03e001a",
				"15c1d4f1a685bb63ee67ca1fd96155e3d091e852a684b78d085fd34f6091e5249ddddbdcf2e7ec82ce6c04c63647eeb7",
			),
		},
	} {
		g := NewG2()
		p0, err := g.HashToCurve(v.msg, domain)
		if err != nil {
			t.Fatal("encode to point fails", i, err)
		}
		if !bytes.Equal(g.ToBytes(p0), v.expected) {
			t.Fatal("encode to point fails x", i)
		}
	}
}

func BenchmarkG2Add(t *testing.B) {
	g2 := NewG2()
	a, b, c := g2.rand(), g2.rand(), PointG2{}
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		g2.Add(&c, a, b)
	}
}

func BenchmarkG2MulWNAF(t *testing.B) {
	g := NewG2()
	p := new(PointG2).Set(&g2One)
	s, _ := new(Fr).Rand(rand.Reader)
	sBig := s.ToBig()
	res := new(PointG2)
	t.Run("Naive", func(t *testing.B) {
		t.ResetTimer()
		for i := 0; i < t.N; i++ {
			g.mulScalar(res, p, s)
		}
	})
	for i := 1; i < 8; i++ {
		wnafMulWindowG2 = uint(i)
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

func BenchmarkG2MulGLV(t *testing.B) {

	g := NewG2()
	p := new(PointG2).Set(&g2One)
	s, _ := new(Fr).Rand(rand.Reader)
	sBig := s.ToBig()
	res := new(PointG2)
	t.Run("Naive", func(t *testing.B) {
		t.ResetTimer()
		for i := 0; i < t.N; i++ {
			g.mulScalar(res, p, s)
		}
	})
	for i := 1; i < 8; i++ {
		glvMulWindowG2 = uint(i)
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

func BenchmarkG2MultiExp(t *testing.B) {
	g := NewG2()
	v := func(n int) ([]*PointG2, []*Fr) {
		bases := make([]*PointG2, n)
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

func BenchmarkG2ClearCofactor(t *testing.B) {
	g2 := NewG2()
	a := g2.rand()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		g2.ClearCofactor(a)
	}
}

func BenchmarkG2SubgroupCheck(t *testing.B) {
	g2 := NewG2()
	a := g2.rand()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		g2.InCorrectSubgroup(a)
	}
}

func BenchmarkG2SWUMap(t *testing.B) {
	a := fromHex(2*fpByteSize, "0x1234")
	g2 := NewG2()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		_, err := g2.MapToCurve(a)
		if err != nil {
			t.Fatal(err)
		}
	}
}
