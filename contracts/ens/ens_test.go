// Copyright 2016 The go-ethereum Authors
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

package ens

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/contracts/ens/contract"
	"github.com/ethereum/go-ethereum/contracts/ens/fallback_contract"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multibase"
	mh "github.com/multiformats/go-multihash"
)

var (
	key, _       = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	name         = "my name on ENS"
	hash         = crypto.Keccak256Hash([]byte("my content"))
	fallbackHash = crypto.Keccak256Hash([]byte("my content hash"))
	addr         = crypto.PubkeyToAddress(key.PublicKey)
	testAddr     = common.HexToAddress("0x1234123412341234123412341234123412341234")
)

func TestENS(t *testing.T) {
	contractBackend := backends.NewSimulatedBackend(core.GenesisAlloc{addr: {Balance: big.NewInt(1000000000)}}, 10000000)
	transactOpts := bind.NewKeyedTransactor(key)

	ensAddr, ens, err := DeployENS(transactOpts, contractBackend)
	if err != nil {
		t.Fatalf("can't deploy root registry: %v", err)
	}
	contractBackend.Commit()

	// Set ourself as the owner of the name.
	if _, err := ens.Register(name); err != nil {
		t.Fatalf("can't register: %v", err)
	}
	contractBackend.Commit()

	// Deploy a resolver and make it responsible for the name.
	resolverAddr, _, _, err := contract.DeployPublicResolver(transactOpts, contractBackend, ensAddr)
	if err != nil {
		t.Fatalf("can't deploy resolver: %v", err)
	}
	if _, err := ens.SetResolver(EnsNode(name), resolverAddr); err != nil {
		t.Fatalf("can't set resolver: %v", err)
	}
	contractBackend.Commit()

	// Set the content hash for the name.
	if _, err = ens.SetContentHash(name, hash.Bytes()); err != nil {
		t.Fatalf("can't set content hash: %v", err)
	}
	contractBackend.Commit()

	// Try to resolve the name.
	vhost, err := ens.Resolve(name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if vhost != hash {
		t.Fatalf("resolve error, expected %v, got %v", hash.Hex(), vhost.Hex())
	}

	// set the address for the name
	if _, err = ens.SetAddr(name, testAddr); err != nil {
		t.Fatalf("can't set address: %v", err)
	}
	contractBackend.Commit()

	// Try to resolve the name to an address
	recoveredAddr, err := ens.Addr(name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if testAddr != recoveredAddr {
		t.Fatalf("resolve error, expected %v, got %v", testAddr.Hex(), recoveredAddr.Hex())
	}

	// deploy the fallback contract and see that the fallback mechanism works
	fallbackResolverAddr, _, _, err := fallback_contract.DeployPublicResolver(transactOpts, contractBackend, ensAddr)
	if err != nil {
		t.Fatalf("can't deploy resolver: %v", err)
	}
	if _, err := ens.SetResolver(EnsNode(name), fallbackResolverAddr); err != nil {
		t.Fatalf("can't set resolver: %v", err)
	}
	contractBackend.Commit()

	// Set the content hash for the name.
	if _, err = ens.SetContentHash(name, fallbackHash.Bytes()); err != nil {
		t.Fatalf("can't set content hash: %v", err)
	}
	contractBackend.Commit()

	// Try to resolve the name.
	vhost, err = ens.Resolve(name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if vhost != fallbackHash {
		t.Fatalf("resolve error, expected %v, got %v", hash.Hex(), vhost.Hex())
	}
	t.Fatal("todo: try to set old contract with new multicodec stuff and assert fail, set new contract with multicodec stuff, encode, decode and assert returns correct hash")
}
func TestManuelCidDecode(t *testing.T) {
	// call cid encode method with hash. expect byte slice returned, compare according to spec
	bb := []byte{}
	buf := make([]byte, binary.MaxVarintLen64)

	for _, v := range []byte{0xe4, 0x01, 0x99, 0x1b, 0x20} {
		n := binary.PutUvarint(buf, uint64(v))
		bb = append(bb, buf[:n]...)
	}
	h := common.HexToHash("29f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f")
	bb = append(bb, h[:]...)
	str := hex.EncodeToString(bb)
	fmt.Println(str)
	decodedHash, e := manualDecode(bb)
	if e != nil {
		t.Fatal(e)
	}

	if !bytes.Equal(decodedHash[:], h[:]) {
		t.Fatal("hashes not equal")
	}
	/* from the EIP documentation
	   storage system: Swarm (0xe4)
	   CID version: 1 (0x01)
	   content type: swarm-manifest (0x??)
	   hash function: keccak-256 (0x1B)
	   hash length: 32 bytes (0x20)
	   hash: 29f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f
	*/

}

func TestManuelCidEncode(t *testing.T) {
	// call cid encode method with hash. expect byte slice returned, compare according to spec

	/* from the EIP documentation
	   storage system: Swarm (0xe4)
	   CID version: 1 (0x01)
	   content type: swarm-manifest (0x??)
	   hash function: keccak-256 (0x1B)
	   hash length: 32 bytes (0x20)
	   hash: 29f2d17be6139079dc48696d1f582a8530eb9805b561eda517e22a892c7e3f1f
	*/

}

func TestCIDSanity(t *testing.T) {
	hashStr := "d1de9994b4d039f6548d191eb26786769f580809256b4685ef316805265ea162"
	hash := common.HexToHash(hashStr) //this always yields a 32 byte long hash
	cc, err := encodeCid(hash)
	if err != nil {
		t.Fatal(err)
	}

	if cc.Prefix().MhLength != 32 {
		t.Fatal("w00t")
	}
	decoded, err := mh.Decode(cc.Hash())
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Length != 32 {
		t.Fatal("invalid length")
	}
	if !bytes.Equal(decoded.Digest, hash[:]) {
		t.Fatalf("hashes not equal")
	}

	if decoded.Length != 32 {
		t.Fatal("wrong length")
	}
	fmt.Println(cc.StringOfBase(multibase.Base16))

	bbbb, e := cc.StringOfBase(multibase.Base16)
	if e != nil {
		t.Fatal(e)
	}
	fmt.Println(bbbb)
	//create the CID string artificially
	hashStr = "f01551b20" + hashStr

	c, err := cid.Decode(hashStr)
	if err != nil {
		t.Fatalf("Error decoding CID: %v", err)
	}

	fmt.Sprintf("Got CID: %v", c)
	fmt.Println("Got CID:", c.Prefix())

}
