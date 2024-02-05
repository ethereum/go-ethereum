package bls12381

import (
	"math/big"
	"testing"
)

func TestPairingExpected(t *testing.T) {
	bls := NewEngine()
	G1, G2 := bls.G1, bls.G2
	GT := bls.GT()
	expected, err := GT.FromBytes(
		fromHex(
			fpByteSize,
			"0x0f41e58663bf08cf068672cbd01a7ec73baca4d72ca93544deff686bfd6df543d48eaa24afe47e1efde449383b676631",
			"0x04c581234d086a9902249b64728ffd21a189e87935a954051c7cdba7b3872629a4fafc05066245cb9108f0242d0fe3ef",
			"0x03350f55a7aefcd3c31b4fcb6ce5771cc6a0e9786ab5973320c806ad360829107ba810c5a09ffdd9be2291a0c25a99a2",
			"0x11b8b424cd48bf38fcef68083b0b0ec5c81a93b330ee1a677d0d15ff7b984e8978ef48881e32fac91b93b47333e2ba57",
			"0x06fba23eb7c5af0d9f80940ca771b6ffd5857baaf222eb95a7d2809d61bfe02e1bfd1b68ff02f0b8102ae1c2d5d5ab1a",
			"0x19f26337d205fb469cd6bd15c3d5a04dc88784fbb3d0b2dbdea54d43b2b73f2cbb12d58386a8703e0f948226e47ee89d",
			"0x018107154f25a764bd3c79937a45b84546da634b8f6be14a8061e55cceba478b23f7dacaa35c8ca78beae9624045b4b6",
			"0x01b2f522473d171391125ba84dc4007cfbf2f8da752f7c74185203fcca589ac719c34dffbbaad8431dad1c1fb597aaa5",
			"0x193502b86edb8857c273fa075a50512937e0794e1e65a7617c90d8bd66065b1fffe51d7a579973b1315021ec3c19934f",
			"0x1368bb445c7c2d209703f239689ce34c0378a68e72a6b3b216da0e22a5031b54ddff57309396b38c881c4c849ec23e87",
			"0x089a1c5b46e5110b86750ec6a532348868a84045483c92b7af5af689452eafabf1a8943e50439f1d59882a98eaa0170f",
			"0x1250ebd871fc0a92a7b2d83168d0d727272d441befa15c503dd8e90ce98db3e7b6d194f60839c508a84305aaca1789b6",
		),
	)
	if err != nil {
		t.Fatal(err)
	}
	r := bls.AddPair(G1.One(), G2.One()).Result()
	if !r.Equal(expected) {
		t.Fatal("expected pairing failed")
	}
	if !GT.IsValid(r) {
		t.Fatal("element is not in correct subgroup")
	}
}

func TestPairingNonDegeneracy(t *testing.T) {
	bls := NewEngine()
	G1, G2 := bls.G1, bls.G2
	g1Zero, g2Zero, g1One, g2One := G1.Zero(), G2.Zero(), G1.One(), G2.One()
	GT := bls.GT()
	// e(g1^a, g2^b) != 1
	bls.Reset()
	{
		bls.AddPair(g1One, g2One)
		e := bls.Result()
		if e.IsOne() {
			t.Fatal("pairing result is not expected to be one")
		}
		if !GT.IsValid(e) {
			t.Fatal("pairing result is not valid")
		}
	}
	// e(g1^a, 0) == 1
	bls.Reset()
	{
		bls.AddPair(g1One, g2Zero)
		e := bls.Result()
		if !e.IsOne() {
			t.Fatal("pairing result is expected to be one")
		}
	}
	// e(0, g2^b) == 1
	bls.Reset()
	{
		bls.AddPair(g1Zero, g2One)
		e := bls.Result()
		if !e.IsOne() {
			t.Fatal("pairing result is expected to be one")
		}
	}
	//
	bls.Reset()
	{
		bls.AddPair(g1Zero, g2One)
		bls.AddPair(g1One, g2Zero)
		bls.AddPair(g1Zero, g2Zero)
		e := bls.Result()
		if !e.IsOne() {
			t.Fatal("pairing result is expected to be one")
		}
	}
	//
	bls.Reset()
	{
		expected, err := GT.FromBytes(
			fromHex(
				fpByteSize,
				"0x0f41e58663bf08cf068672cbd01a7ec73baca4d72ca93544deff686bfd6df543d48eaa24afe47e1efde449383b676631",
				"0x04c581234d086a9902249b64728ffd21a189e87935a954051c7cdba7b3872629a4fafc05066245cb9108f0242d0fe3ef",
				"0x03350f55a7aefcd3c31b4fcb6ce5771cc6a0e9786ab5973320c806ad360829107ba810c5a09ffdd9be2291a0c25a99a2",
				"0x11b8b424cd48bf38fcef68083b0b0ec5c81a93b330ee1a677d0d15ff7b984e8978ef48881e32fac91b93b47333e2ba57",
				"0x06fba23eb7c5af0d9f80940ca771b6ffd5857baaf222eb95a7d2809d61bfe02e1bfd1b68ff02f0b8102ae1c2d5d5ab1a",
				"0x19f26337d205fb469cd6bd15c3d5a04dc88784fbb3d0b2dbdea54d43b2b73f2cbb12d58386a8703e0f948226e47ee89d",
				"0x018107154f25a764bd3c79937a45b84546da634b8f6be14a8061e55cceba478b23f7dacaa35c8ca78beae9624045b4b6",
				"0x01b2f522473d171391125ba84dc4007cfbf2f8da752f7c74185203fcca589ac719c34dffbbaad8431dad1c1fb597aaa5",
				"0x193502b86edb8857c273fa075a50512937e0794e1e65a7617c90d8bd66065b1fffe51d7a579973b1315021ec3c19934f",
				"0x1368bb445c7c2d209703f239689ce34c0378a68e72a6b3b216da0e22a5031b54ddff57309396b38c881c4c849ec23e87",
				"0x089a1c5b46e5110b86750ec6a532348868a84045483c92b7af5af689452eafabf1a8943e50439f1d59882a98eaa0170f",
				"0x1250ebd871fc0a92a7b2d83168d0d727272d441befa15c503dd8e90ce98db3e7b6d194f60839c508a84305aaca1789b6",
			),
		)
		if err != nil {
			t.Fatal(err)
		}
		bls.AddPair(g1Zero, g2One)
		bls.AddPair(g1One, g2Zero)
		bls.AddPair(g1Zero, g2Zero)
		bls.AddPair(g1One, g2One)
		e := bls.Result()
		if !e.Equal(expected) {
			t.Fatal("pairing failed")
		}
	}
}

func TestPairingBilinearity(t *testing.T) {
	bls := NewEngine()
	g1, g2 := bls.G1, bls.G2
	gt := bls.GT()
	// e(a*G1, b*G2) = e(G1, G2)^c
	{
		a, b := big.NewInt(17), big.NewInt(117)
		c := new(big.Int).Mul(a, b)
		G1, G2 := g1.One(), g2.One()
		e0 := bls.AddPair(G1, G2).Result()
		P1, P2 := g1.New(), g2.New()
		g1.MulScalarBig(P1, G1, a)
		g2.MulScalarBig(P2, G2, b)
		e1 := bls.AddPair(P1, P2).Result()
		gt.Exp(e0, e0, c)
		if !e0.Equal(e1) {
			t.Fatal("pairing failed")
		}
	}
	// e(a * G1, b * G2) = e((a * b) * G1, G2)
	{
		// scalars
		a, b := big.NewInt(17), big.NewInt(117)
		c := new(big.Int).Mul(a, b)
		// LHS
		G1, G2 := g1.One(), g2.One()
		g1.MulScalarBig(G1, G1, c)
		bls.AddPair(G1, G2)
		// RHS
		P1, P2 := g1.One(), g2.One()
		g1.MulScalarBig(P1, P1, a)
		g2.MulScalarBig(P2, P2, b)
		bls.AddPairInv(P1, P2)
		// should be one
		if !bls.Check() {
			t.Fatal("pairing failed")
		}
	}
	// e(a * G1, b * G2) = e(G1, (a * b) * G2)
	{
		// scalars
		a, b := big.NewInt(17), big.NewInt(117)
		c := new(big.Int).Mul(a, b)
		// LHS
		G1, G2 := g1.One(), g2.One()
		g2.MulScalarBig(G2, G2, c)
		bls.AddPair(G1, G2)
		// RHS
		H1, H2 := g1.One(), g2.One()
		g1.MulScalarBig(H1, H1, a)
		g2.MulScalarBig(H2, H2, b)
		bls.AddPairInv(H1, H2)
		// should be one
		if !bls.Check() {
			t.Fatal("pairing failed")
		}
	}
}

func TestPairingMulti(t *testing.T) {
	// e(G1, G2) ^ t == e(a01 * G1, a02 * G2) * e(a11 * G1, a12 * G2) * ... * e(an1 * G1, an2 * G2)
	// where t = sum(ai1 * ai2)
	bls := NewEngine()
	g1, g2 := bls.G1, bls.G2
	numOfPair := 100
	targetExp := new(big.Int)
	// RHS
	for i := 0; i < numOfPair; i++ {
		// (ai1 * G1, ai2 * G2)
		a1, a2 := randScalar(qBig), randScalar(qBig)
		P1, P2 := g1.One(), g2.One()
		g1.MulScalarBig(P1, P1, a1)
		g2.MulScalarBig(P2, P2, a2)
		bls.AddPair(P1, P2)
		// accumulate targetExp
		// t += (ai1 * ai2)
		a1.Mul(a1, a2)
		targetExp.Add(targetExp, a1)
	}
	// LHS
	// e(t * G1, G2)
	T1, T2 := g1.One(), g2.One()
	g1.MulScalarBig(T1, T1, targetExp)
	bls.AddPairInv(T1, T2)
	if !bls.Check() {
		t.Fatal("fail multi pairing")
	}
}

func TestPairingEmpty(t *testing.T) {
	bls := NewEngine()
	if !bls.Check() {
		t.Fatal("empty check should be accepted")
	}
	if !bls.Result().IsOne() {
		t.Fatal("empty pairing result should be one")
	}
}

func BenchmarkPairing(t *testing.B) {
	bls := NewEngine()
	g1, g2, gt := bls.G1, bls.G2, bls.GT()
	bls.AddPair(g1.One(), g2.One())
	e := gt.New()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		e = bls.calculate()
	}
	_ = e
}

func BenchmarkMillerLoop(t *testing.B) {
	bls := NewEngine()
	g1, g2, gt := bls.G1, bls.G2, bls.GT()
	bls.AddPair(g1.One(), g2.One())
	f := gt.New().one()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		bls.millerLoop(f)
	}
}

func BenchmarkFinalExp(t *testing.B) {
	bls := NewEngine()
	g1, g2, gt := bls.G1, bls.G2, bls.GT()
	bls.AddPair(g1.One(), g2.One())
	f := gt.New().one()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		bls.finalExp(f)
	}
}
