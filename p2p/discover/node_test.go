package discover

import (
	"math/big"
	"math/rand"
	"net"
	"reflect"
	"testing"
	"testing/quick"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	quickrand = rand.New(rand.NewSource(time.Now().Unix()))
	quickcfg  = &quick.Config{MaxCount: 5000, Rand: quickrand}
)

var parseNodeTests = []struct {
	rawurl     string
	wantError  string
	wantResult *Node
}{
	{
		rawurl:    "http://foobar",
		wantError: `invalid URL scheme, want "enode"`,
	},
	{
		rawurl:    "enode://foobar",
		wantError: `does not contain node ID`,
	},
	{
		rawurl:    "enode://01010101@123.124.125.126:3",
		wantError: `invalid node ID (wrong length, need 64 hex bytes)`,
	},
	{
		rawurl:    "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@hostname:3",
		wantError: `invalid IP address`,
	},
	{
		rawurl:    "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@127.0.0.1:foo",
		wantError: `invalid port`,
	},
	{
		rawurl:    "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@127.0.0.1:3?discport=foo",
		wantError: `invalid discport in query`,
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@127.0.0.1:52150",
		wantResult: &Node{
			ID:  MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			IP:  net.ParseIP("127.0.0.1"),
			UDP: 52150,
			TCP: 52150,
		},
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@[::]:52150",
		wantResult: &Node{
			ID:  MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			IP:  net.ParseIP("::"),
			UDP: 52150,
			TCP: 52150,
		},
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@127.0.0.1:52150?discport=22334",
		wantResult: &Node{
			ID:  MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			IP:  net.ParseIP("127.0.0.1"),
			UDP: 22334,
			TCP: 52150,
		},
	},
}

func TestParseNode(t *testing.T) {
	for i, test := range parseNodeTests {
		n, err := ParseNode(test.rawurl)
		if err == nil && test.wantError != "" {
			t.Errorf("test %d: got nil error, expected %#q", i, test.wantError)
			continue
		}
		if err != nil && err.Error() != test.wantError {
			t.Errorf("test %d: got error %#q, expected %#q", i, err.Error(), test.wantError)
			continue
		}
		if !reflect.DeepEqual(n, test.wantResult) {
			t.Errorf("test %d: result mismatch:\ngot:  %#v, want: %#v", i, n, test.wantResult)
		}
	}
}

func TestNodeString(t *testing.T) {
	for i, test := range parseNodeTests {
		if test.wantError != "" {
			continue
		}
		str := test.wantResult.String()
		if str != test.rawurl {
			t.Errorf("test %d: Node.String() mismatch:\ngot:  %s\nwant: %s", i, str, test.rawurl)
		}
	}
}

func TestHexID(t *testing.T) {
	ref := NodeID{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 128, 106, 217, 182, 31, 165, 174, 1, 67, 7, 235, 220, 150, 66, 83, 173, 205, 159, 44, 10, 57, 42, 161, 26, 188}
	id1 := MustHexID("0x000000000000000000000000000000000000000000000000000000000000000000000000000000806ad9b61fa5ae014307ebdc964253adcd9f2c0a392aa11abc")
	id2 := MustHexID("000000000000000000000000000000000000000000000000000000000000000000000000000000806ad9b61fa5ae014307ebdc964253adcd9f2c0a392aa11abc")

	if id1 != ref {
		t.Errorf("wrong id1\ngot  %v\nwant %v", id1[:], ref[:])
	}
	if id2 != ref {
		t.Errorf("wrong id2\ngot  %v\nwant %v", id2[:], ref[:])
	}
}

func TestNodeID_recover(t *testing.T) {
	prv := newkey()
	hash := make([]byte, 32)
	sig, err := crypto.Sign(hash, prv)
	if err != nil {
		t.Fatalf("signing error: %v", err)
	}

	pub := PubkeyID(&prv.PublicKey)
	recpub, err := recoverNodeID(hash, sig)
	if err != nil {
		t.Fatalf("recovery error: %v", err)
	}
	if pub != recpub {
		t.Errorf("recovered wrong pubkey:\ngot:  %v\nwant: %v", recpub, pub)
	}

	ecdsa, err := pub.Pubkey()
	if err != nil {
		t.Errorf("Pubkey error: %v", err)
	}
	if !reflect.DeepEqual(ecdsa, &prv.PublicKey) {
		t.Errorf("Pubkey mismatch:\n  got:  %#v\n  want: %#v", ecdsa, &prv.PublicKey)
	}
}

func TestNodeID_pubkeyBad(t *testing.T) {
	ecdsa, err := NodeID{}.Pubkey()
	if err == nil {
		t.Error("expected error for zero ID")
	}
	if ecdsa != nil {
		t.Error("expected nil result")
	}
}

func TestNodeID_distcmp(t *testing.T) {
	distcmpBig := func(target, a, b NodeID) int {
		tbig := new(big.Int).SetBytes(target[:])
		abig := new(big.Int).SetBytes(a[:])
		bbig := new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(tbig, abig).Cmp(new(big.Int).Xor(tbig, bbig))
	}
	if err := quick.CheckEqual(distcmp, distcmpBig, quickcfg); err != nil {
		t.Error(err)
	}
}

// the random tests is likely to miss the case where they're equal.
func TestNodeID_distcmpEqual(t *testing.T) {
	base := NodeID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	x := NodeID{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	if distcmp(base, x, x) != 0 {
		t.Errorf("distcmp(base, x, x) != 0")
	}
}

func TestNodeID_logdist(t *testing.T) {
	logdistBig := func(a, b NodeID) int {
		abig, bbig := new(big.Int).SetBytes(a[:]), new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(abig, bbig).BitLen()
	}
	if err := quick.CheckEqual(logdist, logdistBig, quickcfg); err != nil {
		t.Error(err)
	}
}

// the random tests is likely to miss the case where they're equal.
func TestNodeID_logdistEqual(t *testing.T) {
	x := NodeID{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	if logdist(x, x) != 0 {
		t.Errorf("logdist(x, x) != 0")
	}
}

func TestNodeID_randomID(t *testing.T) {
	// we don't use quick.Check here because its output isn't
	// very helpful when the test fails.
	for i := 0; i < quickcfg.MaxCount; i++ {
		a := gen(NodeID{}, quickrand).(NodeID)
		dist := quickrand.Intn(len(NodeID{}) * 8)
		result := randomID(a, dist)
		actualdist := logdist(result, a)

		if dist != actualdist {
			t.Log("a:     ", a)
			t.Log("result:", result)
			t.Fatalf("#%d: distance of result is %d, want %d", i, actualdist, dist)
		}
	}
}

func (NodeID) Generate(rand *rand.Rand, size int) reflect.Value {
	var id NodeID
	m := rand.Intn(len(id))
	for i := len(id) - 1; i > m; i-- {
		id[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(id)
}
