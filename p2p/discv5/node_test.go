// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package discv5

import (
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func ExampleNewNode() {
	id := MustHexID("1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439")

	// Complete nodes contain UDP and TCP endpoints:
	n1 := NewNode(id, net.ParseIP("2001:db8:3c4d:15::abcd:ef12"), 52150, 30303)
	fmt.Println("n1:", n1)
	fmt.Println("n1.Incomplete() ->", n1.Incomplete())

	// An incomplete node can be created by passing zero values
	// for all parameters except id.
	n2 := NewNode(id, nil, 0, 0)
	fmt.Println("n2:", n2)
	fmt.Println("n2.Incomplete() ->", n2.Incomplete())

	// Output:
	// n1: enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@[2001:db8:3c4d:15::abcd:ef12]:30303?discport=52150
	// n1.Incomplete() -> false
	// n2: enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439
	// n2.Incomplete() -> true
}

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
		rawurl:    "enode://01010101@123.124.125.126:3",
		wantError: `invalid node ID (wrong length, want 128 hex chars)`,
	},
	// Complete nodes with IP address.
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
		wantResult: NewNode(
			MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			net.IP{0x7f, 0x0, 0x0, 0x1},
			52150,
			52150,
		),
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@[::]:52150",
		wantResult: NewNode(
			MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			net.ParseIP("::"),
			52150,
			52150,
		),
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@[2001:db8:3c4d:15::abcd:ef12]:52150",
		wantResult: NewNode(
			MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			net.ParseIP("2001:db8:3c4d:15::abcd:ef12"),
			52150,
			52150,
		),
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439@127.0.0.1:52150?discport=22334",
		wantResult: NewNode(
			MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			net.IP{0x7f, 0x0, 0x0, 0x1},
			22334,
			52150,
		),
	},
	// Incomplete nodes with no address.
	{
		rawurl: "1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439",
		wantResult: NewNode(
			MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			nil, 0, 0,
		),
	},
	{
		rawurl: "enode://1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439",
		wantResult: NewNode(
			MustHexID("0x1dd9d65c4552b5eb43d5ad55a2ee3f56c6cbc1c64a5c8d659f51fcd51bace24351232b8d7821617d2b29b54b81cdefb9b3e9c37d7fd5f63270bcc9e1a6f6a439"),
			nil, 0, 0,
		),
	},
	// Invalid URLs
	{
		rawurl:    "01010101",
		wantError: `invalid node ID (wrong length, want 128 hex chars)`,
	},
	{
		rawurl:    "enode://01010101",
		wantError: `invalid node ID (wrong length, want 128 hex chars)`,
	},
	{
		// This test checks that errors from url.Parse are handled.
		rawurl:    "://foo",
		wantError: `missing protocol scheme`,
	},
}

func TestParseNode(t *testing.T) {
	for _, test := range parseNodeTests {
		n, err := ParseNode(test.rawurl)
		if test.wantError != "" {
			if err == nil {
				t.Errorf("test %q:\n  got nil error, expected %#q", test.rawurl, test.wantError)
				continue
			} else if !strings.Contains(err.Error(), test.wantError) {
				t.Errorf("test %q:\n  got error %#q, expected %#q", test.rawurl, err.Error(), test.wantError)
				continue
			}
		} else {
			if err != nil {
				t.Errorf("test %q:\n  unexpected error: %v", test.rawurl, err)
				continue
			}
			if !reflect.DeepEqual(n, test.wantResult) {
				t.Errorf("test %q:\n  result mismatch:\ngot:  %#v, want: %#v", test.rawurl, n, test.wantResult)
			}
		}
	}
}

func TestNodeString(t *testing.T) {
	for i, test := range parseNodeTests {
		if test.wantError == "" && strings.HasPrefix(test.rawurl, "enode://") {
			str := test.wantResult.String()
			if str != test.rawurl {
				t.Errorf("test %d: Node.String() mismatch:\ngot:  %s\nwant: %s", i, str, test.rawurl)
			}
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
	distcmpBig := func(target, a, b common.Hash) int {
		tbig := new(big.Int).SetBytes(target[:])
		abig := new(big.Int).SetBytes(a[:])
		bbig := new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(tbig, abig).Cmp(new(big.Int).Xor(tbig, bbig))
	}
	if err := quick.CheckEqual(distcmp, distcmpBig, quickcfg()); err != nil {
		t.Error(err)
	}
}

// the random tests is likely to miss the case where they're equal.
func TestNodeID_distcmpEqual(t *testing.T) {
	base := common.Hash{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	x := common.Hash{15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0}
	if distcmp(base, x, x) != 0 {
		t.Errorf("distcmp(base, x, x) != 0")
	}
}

func TestNodeID_logdist(t *testing.T) {
	logdistBig := func(a, b common.Hash) int {
		abig, bbig := new(big.Int).SetBytes(a[:]), new(big.Int).SetBytes(b[:])
		return new(big.Int).Xor(abig, bbig).BitLen()
	}
	if err := quick.CheckEqual(logdist, logdistBig, quickcfg()); err != nil {
		t.Error(err)
	}
}

// the random tests is likely to miss the case where they're equal.
func TestNodeID_logdistEqual(t *testing.T) {
	x := common.Hash{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	if logdist(x, x) != 0 {
		t.Errorf("logdist(x, x) != 0")
	}
}

func TestNodeID_hashAtDistance(t *testing.T) {
	// we don't use quick.Check here because its output isn't
	// very helpful when the test fails.
	cfg := quickcfg()
	for i := 0; i < cfg.MaxCount; i++ {
		a := gen(common.Hash{}, cfg.Rand).(common.Hash)
		dist := cfg.Rand.Intn(len(common.Hash{}) * 8)
		result := hashAtDistance(a, dist)
		actualdist := logdist(result, a)

		if dist != actualdist {
			t.Log("a:     ", a)
			t.Log("result:", result)
			t.Fatalf("#%d: distance of result is %d, want %d", i, actualdist, dist)
		}
	}
}

func quickcfg() *quick.Config {
	return &quick.Config{
		MaxCount: 5000,
		Rand:     rand.New(rand.NewSource(time.Now().Unix())),
	}
}

// TODO: The Generate method can be dropped when we require Go >= 1.5
// because testing/quick learned to generate arrays in 1.5.

func (NodeID) Generate(rand *rand.Rand, size int) reflect.Value {
	var id NodeID
	m := rand.Intn(len(id))
	for i := len(id) - 1; i > m; i-- {
		id[i] = byte(rand.Uint32())
	}
	return reflect.ValueOf(id)
}
