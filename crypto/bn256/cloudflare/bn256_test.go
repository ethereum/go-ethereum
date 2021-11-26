package bn256

import (
	"bytes"
	"crypto/rand"
	"testing"
)

func TestG1Marshal(t *testing.T) {
	_, Ga, err := RandomG1(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ma := Ga.Marshal()

	Gb := new(G1)
	_, err = Gb.Unmarshal(ma)
	if err != nil {
		t.Fatal(err)
	}
	mb := Gb.Marshal()

	if !bytes.Equal(ma, mb) {
		t.Fatal("bytes are different")
	}
}

func TestG2Marshal(t *testing.T) {
	_, Ga, err := RandomG2(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ma := Ga.Marshal()

	Gb := new(G2)
	_, err = Gb.Unmarshal(ma)
	if err != nil {
		t.Fatal(err)
	}
	mb := Gb.Marshal()

	if !bytes.Equal(ma, mb) {
		t.Fatal("bytes are different")
	}
}

func TestBilinearity(t *testing.T) {
	for i := 0; i < 2; i++ {
		a, p1, _ := RandomG1(rand.Reader)
		b, p2, _ := RandomG2(rand.Reader)
		e1 := Pair(p1, p2)

		e2 := Pair(&G1{curveGen}, &G2{twistGen})
		e2.ScalarMult(e2, a)
		e2.ScalarMult(e2, b)

		if *e1.p != *e2.p {
			t.Fatalf("bad pairing result: %s", e1)
		}
	}
}

func TestTripartiteDiffieHellman(t *testing.T) {
	a, _ := rand.Int(rand.Reader, Order)
	b, _ := rand.Int(rand.Reader, Order)
	c, _ := rand.Int(rand.Reader, Order)

	pa, pb, pc := new(G1), new(G1), new(G1)
	qa, qb, qc := new(G2), new(G2), new(G2)

	pa.Unmarshal(new(G1).ScalarBaseMult(a).Marshal())
	qa.Unmarshal(new(G2).ScalarBaseMult(a).Marshal())
	pb.Unmarshal(new(G1).ScalarBaseMult(b).Marshal())
	qb.Unmarshal(new(G2).ScalarBaseMult(b).Marshal())
	pc.Unmarshal(new(G1).ScalarBaseMult(c).Marshal())
	qc.Unmarshal(new(G2).ScalarBaseMult(c).Marshal())

	k1 := Pair(pb, qc)
	k1.ScalarMult(k1, a)
	k1Bytes := k1.Marshal()

	k2 := Pair(pc, qa)
	k2.ScalarMult(k2, b)
	k2Bytes := k2.Marshal()

	k3 := Pair(pa, qb)
	k3.ScalarMult(k3, c)
	k3Bytes := k3.Marshal()

	if !bytes.Equal(k1Bytes, k2Bytes) || !bytes.Equal(k2Bytes, k3Bytes) {
		t.Errorf("keys didn't agree")
	}
}

func TestBinaryEAA(t *testing.T) {
	for i := 0; i < 10000; i++ {
		_, Ga, err := RandomG1(rand.Reader)
		if err != nil {
			t.Fatal(err)
		}
		tmpLittleFermat := &gfP{}
		tmpLittleFermat.Invert(&Ga.p.x)

		tmpBinaryEAA := &gfP{}
		tmpBinaryEAA.InvertVariableTime(&Ga.p.x)

		tmpBinaryEAASelfSet := &gfP{}
		tmpBinaryEAASelfSet.Set(&Ga.p.x)
		tmpBinaryEAASelfSet.InvertVariableTime(tmpBinaryEAASelfSet)

		if *tmpLittleFermat != *tmpBinaryEAA {
			t.Fatalf("results of different inversion do not agree")
		}

		if *tmpLittleFermat != *tmpBinaryEAASelfSet {
			t.Fatalf("self-assigned inversion is invalid")
		}
	}
}

func BenchmarkLittleFermatInversion(b *testing.B) {
	el := gfP{0x0, 0x97816a916871ca8d, 0xb85045b68181585d, 0x30644e72e131a029}

	b.ResetTimer()

	tmp := &gfP{}
	for i := 0; i < b.N; i++ {
		tmp.Invert(&el)
	}
}

func BenchmarkBinaryEEAInversion(b *testing.B) {
	el := gfP{0x0, 0x97816a916871ca8d, 0xb85045b68181585d, 0x30644e72e131a029}

	b.ResetTimer()

	tmp := &gfP{}
	for i := 0; i < b.N; i++ {
		tmp.InvertVariableTime(&el)
	}
}

func BenchmarkG1AddAndMakeAffine(b *testing.B) {
	_, Ga, err := RandomG1(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	_, Gb, err := RandomG1(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		e := new(G1).Add(Ga, Gb)
		e.p.MakeAffine()
	}
}

func TestG2SelfAddition(t *testing.T) {
	s, _ := rand.Int(rand.Reader, Order)
	p := new(G2).ScalarBaseMult(s)

	if !p.p.IsOnCurve() {
		t.Fatal("p isn't on curve")
	}
	m := p.Add(p, p).Marshal()
	if _, err := p.Unmarshal(m); err != nil {
		t.Fatalf("p.Add(p, p) ∉ G₂: %v", err)
	}
}

func BenchmarkG1(b *testing.B) {
	x, _ := rand.Int(rand.Reader, Order)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		new(G1).ScalarBaseMult(x)
	}
}

func BenchmarkG2(b *testing.B) {
	x, _ := rand.Int(rand.Reader, Order)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		new(G2).ScalarBaseMult(x)
	}
}
func BenchmarkPairing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Pair(&G1{curveGen}, &G2{twistGen})
	}
}
