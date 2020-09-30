// Copyright 2020 The go-ethereum Authors
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

package v5wire

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

func TestVector_ECDH(t *testing.T) {
	var (
		staticKey = hexPrivkey("0xfb757dc581730490a1d7a00deea65e9b1936924caaea8f44d476014856b68736")
		publicKey = hexPubkey(crypto.S256(), "0x039961e4c2356d61bedb83052c115d311acb3a96f5777296dcf297351130266231")
		want      = hexutil.MustDecode("0x033b11a2a1f214567e1537ce5e509ffd9b21373247f2a3ff6841f4976f53165e7e")
	)
	result := ecdh(staticKey, publicKey)
	check(t, "shared-secret", result, want)
}

func TestVector_KDF(t *testing.T) {
	var (
		ephKey    = hexPrivkey("0xfb757dc581730490a1d7a00deea65e9b1936924caaea8f44d476014856b68736")
		net       = newHandshakeTest()
		challenge Whoareyou
	)
	copy(challenge.Header.IV[:], hexutil.MustDecode("0x01010101010101010101010101010101"))
	copy(challenge.IDNonce[:], hexutil.MustDecode("0x02020202020202020202020202020202"))
	defer net.close()

	destKey := &net.nodeB.c.privkey.PublicKey
	s := net.nodeA.c.deriveKeys(net.nodeA.id(), net.nodeB.id(), ephKey, destKey, &challenge)
	t.Logf("ephemeral-key = %#x", ephKey.D)
	t.Logf("dest-pubkey = %#x", EncodePubkey(destKey))
	t.Logf("node-id-a = %#x", net.nodeA.id().Bytes())
	t.Logf("node-id-b = %#x", net.nodeB.id().Bytes())
	t.Logf("whoareyou.masking-iv = %#x", challenge.Header.IV[:])
	t.Logf("whoareyou.id-nonce = %#x", challenge.IDNonce[:])
	check(t, "initiator-key", s.writeKey, hexutil.MustDecode("0xb10e94a89b34cfb87b65aa7f8902f40c"))
	check(t, "recipient-key", s.readKey, hexutil.MustDecode("0xce8db25ae599c9b2c4a9d60090c9efdd"))
}

func TestVector_IDSignature(t *testing.T) {
	var (
		key    = hexPrivkey("0xfb757dc581730490a1d7a00deea65e9b1936924caaea8f44d476014856b68736")
		destID = enode.HexID("0xbbbb9d047f0488c0b5a93c1c3f2d8bafc7c8ff337024a55434a0d0555de64db9")
		ephkey = hexutil.MustDecode("0x039961e4c2356d61bedb83052c115d311acb3a96f5777296dcf297351130266231")
		header = Header{
			AuthData: hexutil.MustDecode("0x0102030405060708090a0b0c0d0e0f100000000000000000"),
		}
	)
	copy(header.IV[:], hexutil.MustDecode("0x01010101010101010101010101010101"))

	sig, err := makeIDSignature(sha256.New(), key, destID, ephkey, &Header{})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("static-key = %#x", key.D)
	t.Logf("masking-iv = %#x", header.IV[:])
	t.Logf("authdata = %#x", header.AuthData)
	t.Logf("ephemeral-pubkey = %#x", ephkey)
	t.Logf("node-id-B = %#x", destID.Bytes())
	check(t, "id-signature", sig, hexutil.MustDecode("0xd82364cfffb18101355371de84ee0def3dca31191b9add79b21a14f4442b6df02dc26df6278f71c83d43645da13071881cacdb43b0aea1e256cdec73a73faf01"))
}

func check(t *testing.T, what string, x, y []byte) {
	t.Helper()

	if !bytes.Equal(x, y) {
		t.Errorf("wrong %s: %#x != %#x", what, x, y)
	} else {
		t.Logf("%s = %#x", what, x)
	}
}

func hexPrivkey(input string) *ecdsa.PrivateKey {
	key, err := crypto.HexToECDSA(strings.TrimPrefix(input, "0x"))
	if err != nil {
		panic(err)
	}
	return key
}

func hexPubkey(curve elliptic.Curve, input string) *ecdsa.PublicKey {
	key, err := DecodePubkey(curve, hexutil.MustDecode(input))
	if err != nil {
		panic(err)
	}
	return key
}
