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

package trie

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	mrand "math/rand"
	"slices"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Prng is a pseudo random number generator seeded by strong randomness.
// The randomness is printed on startup in order to make failures reproducible.
var prng = initRnd()

func initRnd() *mrand.Rand {
	var seed [8]byte
	crand.Read(seed[:])
	rnd := mrand.New(mrand.NewSource(int64(binary.LittleEndian.Uint64(seed[:]))))
	fmt.Printf("Seed: %x\n", seed)
	return rnd
}

func randBytes(n int) []byte {
	r := make([]byte, n)
	prng.Read(r)
	return r
}

// makeProvers creates Merkle trie provers based on different implementations to
// test all variations.
func makeProvers(trie *Trie) []func(key []byte) *memorydb.Database {
	var provers []func(key []byte) *memorydb.Database

	// Create a direct trie based Merkle prover
	provers = append(provers, func(key []byte) *memorydb.Database {
		proof := memorydb.New()
		trie.Prove(key, proof)
		return proof
	})
	// Create a leaf iterator based Merkle prover
	provers = append(provers, func(key []byte) *memorydb.Database {
		proof := memorydb.New()
		if it := NewIterator(trie.MustNodeIterator(key)); it.Next() && bytes.Equal(key, it.Key) {
			for _, p := range it.Prove() {
				proof.Put(crypto.Keccak256(p), p)
			}
		}
		return proof
	})
	return provers
}

func TestProof(t *testing.T) {
	trie, vals := randomTrie(500)
	root := trie.Hash()
	for i, prover := range makeProvers(trie) {
		for _, kv := range vals {
			proof := prover(kv.k)
			if proof == nil {
				t.Fatalf("prover %d: missing key %x while constructing proof", i, kv.k)
			}
			val, err := VerifyProof(root, kv.k, proof)
			if err != nil {
				t.Fatalf("prover %d: failed to verify proof for key %x: %v\nraw proof: %x", i, kv.k, err, proof)
			}
			if !bytes.Equal(val, kv.v) {
				t.Fatalf("prover %d: verified value mismatch for key %x: have %x, want %x", i, kv.k, val, kv.v)
			}
		}
	}
}

func TestVerifyProofWithLastNode(t *testing.T) {
	testCase := []struct {
		desc  string
		root  string
		key   string
		proof []string
		value string

		expectedError               error
		expextedValue               string
		expectedLongestPrefixPrefix string
		expectedLastNode            string
	}{
		{
			desc: "Valid inclusion proof",
			root: "0x6f39539da0b571e36e04cdee1ef9273ce168644d63822352f3a18c0504220166",
			key:  "0xab14d68802a763f7db875346d03fbf86f137de55814b191c069e721f47474733",
			proof: []string{
				"0xf90211a026b1f06f4804f169f036ab0cbe42d00ac13da34d6ff9af107d1af41d425eef8aa097d5cf7afeb561c87f064e5055cb5d6634cb7f6e93636b5d2c819b73db9f60f2a05d21cb1cb916ce9e57594a3c31058d2a936a5d7f790da95c5fab850252ff93a1a0fcf8847652abe31e2c5096e87128d93fb2de640534deffd530c82dff3d8f30fba026930d0a381d96f36abdb8857d6d8e91c0acd22b2ba78399e573798622638067a0417e2530f32e6c01a6cbdda472b853b98076436110df109ffad71e9b1a685feaa04f9c68013dc5f94ec91621bda78d233590b5d91f846698e60b42b28e0c87d935a0a06147fa3e0b0272661f8402f048b2cf7061617684bf44d4154d889c3f5119faa039164ab7071740ceeeb687f999251decfbe781ac0fd31cb63624c2f4b99bca7fa089443797be9cb0c265caeabcef452c23fbfcbac12cfe58a31f7d3ad6fb0d53f1a08357d7d9cf194e90858d2186918f396ed971062fcf939b78cd5b43e7fb44eb9ca0f0e23f2080dcd0af55e7ef37a9ccab09716a4c31e0b1b2172ac416c53df675a4a0ba840647ec3e2f7d38317c7dc0a6f8e3ae14b398c454ce9e76b0232d782f66caa081b03c3a644f130eb88395dac6ecad40b1ba14826fa4fef38f82ef2fa63e2aeea01b356b63266c0771c8167162e5472815b7836bf50dc1b5d70ee982ddc619b719a08878b527ef493b8b571b73b5747370e12e1fe09dd003546b9755e39c9f98050880",
				"0xf90211a09a94b4b8ab62c73bec14fd530e704e997b09f51aa9b316cc9a3b5ad0164e2530a0dca0351f0790d336e42aca22aabdbb4be8ac126c7e95fba25b55a78179cf54eba004b4813a8cad85b721321b6738f362f79911db78db50ec2571873cc1e4b00512a0bcc74ae61638d722a57cb0507bd0f30dbef47f591df032a5422eb8b3aaa015d1a0a4f53957bf35159735d8902e208471ecfa0e31ccc6e577e9b56f81d061719de6a0e77e0548e98157dc2f18b9f694965df88abf3c78b3090990c329264f2ac84325a0672727f7669561b1ddd33117f51146d4712af029e26cf1fce30608a7e8f4291ea088827c096ac6ce1fd783b9e418bb37711e6b0b34b429e5b7a321fac508d1a8aea03030c43045ea3ac75803b189a7132d88d0956a86424e3207ee38a46f2f850566a0298e2ca265c46fd3096368b732a77977211a0c55d1246dbd990111cb26fc4c66a07afd6cb69666868b214b594819385a9c51ba41ce927b5e83e0282f258be4ec9ea090d5e48e6b82c7d3963a1a944575160cf6297e03589096defc75ee8ce3cae787a0d65839513fe22a30a61cd19481e3c0ccae670d0c5de3ce6427f862fa74b8f5b9a0e3ba073d7f8cbcc33cb8a9d7254eff98d4230960e8e02eaf189da50afc52550fa0bfb620535240603e78e23975fa04ec78737125c57ef0e1295ae4791dbd72eb98a0dc02979f215e06a5c38819bd42767a8e1856ac98d0eebddd136bc18395e01f9780",
				"0xf90211a0b45de0172eb2119f86e9a53414150582d4052e90d9c6bbdd124d17e90a8118c5a05da2d2141759d4ef34f007c423b60b148d190d35d04f261cf5e657e4549782e8a0acbb3eef7fd8fd5f484aed84d2cb3416e46599f6d29efc6be6c63666f72e8e9ca02ab8b018174045d2c79ae86ef00d70c7cdd964cfe225ce36c14506b88a2f1a96a031d430d208f212e6100cfe74c4c694fa18c55bc0834f6ebec8a7d413039bfb63a05dec69eb4414309d3e7cd09c929692fe28b1a0cb116635d0b57e9f712716ebb0a01205d9623ad31d97cbd6ef3c20008284386ddf66aee79984a8c006b9d411d46ea089f83ffcdedd51ca1ac27376be77c5c1ca9aa5712aba25e0f82c8c39f2e8b6c4a08a9dcb56eb25df8cbca5f7b18e43b7f6bbf70571c9ad924a1791e5aa55df2331a0e1fca8262813be4837902a1fb24584e3f504af383dc84aa7838d0369c942217aa0adfc616da1e64c42b4cb11886ccf6642222b43bff8d266caeefc572ca3e0fb65a01f5c53e0c2a0109df11c313845b7e46910004981044de1772aabba746d2ce9a5a0dd357d528f9f34a8a3ae37358496f91675b47f6133e6bd0568d581a840c6eeb4a09ece58670426075fec57d32fbac503973288de9eeb3545175b64ebe1cfad9930a0d3c0d1c48df816d9a5dc772d7f464e8872d49dbe6fcea0944b56663514237f84a066ab6c2f3748861ad36c6d19f8b2cc0ee17fca96ea862e4d3ce3beac233a75c980",
				"0xf90211a042ab718a6fbb71add03da39ed57525bddd6f8455f9e77ce7f33d603cd86b791aa080b40e9fc893fc8725853fb11b6e2665376a790ff68ef273c93f0b7025fccb28a00defb48ea51e2fdb1d6f353dc5724796c073d8d80cb38d517020200e39b23051a0236e4a72db3ea239bf7edc82e7f2e9276a0f05e6bd6309023a65eb2b8fa2559fa0bcae7ef6000f79902f615489ceb4a727b39326acdd4462664503fceb4a87d36ba06a1b1da69a20a57324962130a82d2de2ad7783f87e49c1536ae23912c9e8c866a098792dc09ed13762e2bcb1c21590395500fe3162e8b9b02433e821cdeff478b5a05bcf60cfe56c1d9d6a2d63a9e0a8e91f35ac851526159bc27ef3b989daa68b4aa03d93b7e3802a49e207f8aaa619dedd7a1d8f2be43ace5b641611c66a78dde10aa077a649d238a2c0fdcd00fa8058601d1d72437ee4fafd948c34d82475fbea902aa0fa62b7a999a0584fe8a57056c60257fc24d5c5bd59a6ac04cc0e23c2356555f4a0570a6bb6e76afeb523d7536b08a5f064675e8731daf902d12bd0c4472626f6aea07fdcf35ae85861ed95f3375ae18dc5ea4518b056a8ac157c32bb8abd2a147d53a04a413bcb2c143a4038755095c087593a197d15f31ee4c11399021d0928f7d50ea0be95324a2306fcf30b974a1283e2df2d7562c55875343ae1e29fa7a67e835fe8a0e05a31ba76f7ddd817b7010f427234b38fbd2f12a03c222dde19ecee94dcb43080",
				"0xf90211a056527244566aed48654b0ca3e95225defe8b8eaee17657bace6698d7744b06cda0118221e9076b2fdf85d2371af95c10b9e9539a9aa3e3f6947acc0c7958bbdeeda0eed1a6e3b203d643cf0b3349816a78974046a45972cce621ea4a1979c29c632ca0414fd647c29c1323fe866af7f73d88c7ba93ba3d3eedea38aebd7a800c34ac75a0595cc2c4b2d314f1c12e7407732c4040bec3b269f499011c733b88f63007ae75a0836b374e20ca7ed03b5c5a447b6e15a39c0d037a82a429f8e099a06a24a372eaa085bcaf2bf96c47456365f4b28869670dee8d910c69ef7c75facc1c6e8e8983bfa0540629174895cf121ea76f7b63647bd9c46378184aeff99c81be646e0e6b80bca03048c27521797b94c190972758f2592a298af84bc14a2af2d1cfdcd00dc4b4a3a04456762d4c42229917c26b66106d1b354d12de086b65ad9c4ac0c97ae1fee28ba07b5ec4d7acfb65d85ce3c2c5c8528ea71e03997ddfe4a849cf02af2b61334276a0d49449f71c85a405d1c3ac16e903f8e2333c5e86c2b294e624fb0084317e4183a09d66aac7c383422398f421567ad2b103ce002961e973c8321e03b8e26bbb4c9da0cbccafd20b1c1162663b03bf8387639db8b1472d344dbda2f85805ea3f7061dca0402abdf44ab8e0971f58fadece8c64c1d9fabe5b2bc9ffaa30c847df16613520a03e3e95cd292c1fc017d5386e0c65923e423cddafe918bc50d6cd4ba96e8a4fe580",
				"0xf90211a054ecd08900814f3b9e373cbc4ea3701007e84703bc5af9cfaa9e1711fd3f0319a09f9261744412d1946bcd933e1a6d53102186b2d99ecff30e403feea9841cf053a067f9200eef6cd28b708d6292b57e2a42b577874e5b1cbd406741e61a483e6b7aa081b4a58834d68fdb58de1b2f4b422b06f6094ceea952f8d0bf4746d743eefd70a02f59d7a6832a735d3116657cb0941143aa82d9c6d38efa250bf5827071bf63e4a09007eccc09dc7929e3574e1612a8867660accce899f70546173cbe30b109ea94a08d3f6cbecc09531cb79bdde35403a8b1192cfcdf6f0aca0a4429e5708920a254a031f06093af171b8380db7bff2e61463f6c4dc01c8d7e7181cf642bad24202357a0a60e1315be595edd2f36b0c8070638278ebfc3d1b02d9ad4828a690564f98c8ea0e4e3830e486521cc62bc05be33fb82dc976be5f77031cb9cff945c6fd9c07f6ba043a6e474ef36831d726f8260533cde7ee60409a3ebfc4660f0a0ec15f36098dda01af29c4ae492c1ef17d49e759c535e83a0f10e5be08e9218ad50a9a18a058604a096d3e05bb50ee4b1cec89a4eefea21fc8939fa5d1d698dddee320266d3245853a0ad45735c19f6fdd0d89dbbaa7aae4599b0e3fb099ae36b48410fb55231a1facca09cd3816e190ebd4bfefeb9cbff3ac539ad05ace88ca3dbae324326b61b52b2b4a0084ed25dc92cd64bf00f64818b2d4a3655c038035abe7daf5b04b9949522263780",
				"0xf90191a00a7a0118e00981ab321049c9d340cd52c3a4781037540f7c48d0fdc27e899b3280a08537f2e248702a6ae2a57e9110a5740f5772c876389739ac90debd6a0692713ea00b3a26a05b5494fb3ff6f0b3897688a5581066b20b07ebab9252d169d928717fa0a9a54d84976d134d6dba06a65064c7f3a964a75947d452db6f6bb4b6c47b43aaa01e2a1ed3d1572b872bbf09ee44d2ed737da31f01de3c0f4b4e1f046740066461a098be0d129102f96ae636f9928911ed66103ac7c62235241aa0ef7e5b4b7e289ea07da2bce701255847cf5169ba5a7578a9700133f7ce13fa26a1d4097c20d1e0fda00631e0c3a589b999513f1d44a9b11e53d81027c1019553b4100542dbcb21031fa0c8d71dd13d2806e2865a5c2cfa447f626471bf0b66182a8fd07230434e1cad2680a0e9864fdfaf3693b2602f56cd938ccd494b8634b1f91800ef02203a3609ca4c21a0c69d174ad6b6e58b0bd05914352839ec60915cd066dd2bee2a48016139687f21a0513dd5514fd6bad56871711441d38de2821cc6913cb192416b0385f025650731808080",
				"0xf8669d3802a763f7db875346d03fbf86f137de55814b191c069e721f47474733b846f8440101a065d17ccfe8328a42712f5dcd7a8827eebab3341e1a8bd6a4cb741495b83bd026a0b44fb4e949d0f78f87f79ee46428f23a2a5713ce6fc6e0beb3dda78c2ac1ea55",
			},
			expextedValue:               "0xf8440101a065d17ccfe8328a42712f5dcd7a8827eebab3341e1a8bd6a4cb741495b83bd026a0b44fb4e949d0f78f87f79ee46428f23a2a5713ce6fc6e0beb3dda78c2ac1ea55",
			expectedLongestPrefixPrefix: "ab14d68",
			expectedLastNode:            "0xf8669d3802a763f7db875346d03fbf86f137de55814b191c069e721f47474733b846f8440101a065d17ccfe8328a42712f5dcd7a8827eebab3341e1a8bd6a4cb741495b83bd026a0b44fb4e949d0f78f87f79ee46428f23a2a5713ce6fc6e0beb3dda78c2ac1ea55",
		},
		{
			desc: "Valid exclusion proof #1",
			root: "0x6f39539da0b571e36e04cdee1ef9273ce168644d63822352f3a18c0504220166",
			key:  "0xab14d68802a763f7db875346d000000000000000000000000000000000000000",
			proof: []string{
				"0xf90211a026b1f06f4804f169f036ab0cbe42d00ac13da34d6ff9af107d1af41d425eef8aa097d5cf7afeb561c87f064e5055cb5d6634cb7f6e93636b5d2c819b73db9f60f2a05d21cb1cb916ce9e57594a3c31058d2a936a5d7f790da95c5fab850252ff93a1a0fcf8847652abe31e2c5096e87128d93fb2de640534deffd530c82dff3d8f30fba026930d0a381d96f36abdb8857d6d8e91c0acd22b2ba78399e573798622638067a0417e2530f32e6c01a6cbdda472b853b98076436110df109ffad71e9b1a685feaa04f9c68013dc5f94ec91621bda78d233590b5d91f846698e60b42b28e0c87d935a0a06147fa3e0b0272661f8402f048b2cf7061617684bf44d4154d889c3f5119faa039164ab7071740ceeeb687f999251decfbe781ac0fd31cb63624c2f4b99bca7fa089443797be9cb0c265caeabcef452c23fbfcbac12cfe58a31f7d3ad6fb0d53f1a08357d7d9cf194e90858d2186918f396ed971062fcf939b78cd5b43e7fb44eb9ca0f0e23f2080dcd0af55e7ef37a9ccab09716a4c31e0b1b2172ac416c53df675a4a0ba840647ec3e2f7d38317c7dc0a6f8e3ae14b398c454ce9e76b0232d782f66caa081b03c3a644f130eb88395dac6ecad40b1ba14826fa4fef38f82ef2fa63e2aeea01b356b63266c0771c8167162e5472815b7836bf50dc1b5d70ee982ddc619b719a08878b527ef493b8b571b73b5747370e12e1fe09dd003546b9755e39c9f98050880",
				"0xf90211a09a94b4b8ab62c73bec14fd530e704e997b09f51aa9b316cc9a3b5ad0164e2530a0dca0351f0790d336e42aca22aabdbb4be8ac126c7e95fba25b55a78179cf54eba004b4813a8cad85b721321b6738f362f79911db78db50ec2571873cc1e4b00512a0bcc74ae61638d722a57cb0507bd0f30dbef47f591df032a5422eb8b3aaa015d1a0a4f53957bf35159735d8902e208471ecfa0e31ccc6e577e9b56f81d061719de6a0e77e0548e98157dc2f18b9f694965df88abf3c78b3090990c329264f2ac84325a0672727f7669561b1ddd33117f51146d4712af029e26cf1fce30608a7e8f4291ea088827c096ac6ce1fd783b9e418bb37711e6b0b34b429e5b7a321fac508d1a8aea03030c43045ea3ac75803b189a7132d88d0956a86424e3207ee38a46f2f850566a0298e2ca265c46fd3096368b732a77977211a0c55d1246dbd990111cb26fc4c66a07afd6cb69666868b214b594819385a9c51ba41ce927b5e83e0282f258be4ec9ea090d5e48e6b82c7d3963a1a944575160cf6297e03589096defc75ee8ce3cae787a0d65839513fe22a30a61cd19481e3c0ccae670d0c5de3ce6427f862fa74b8f5b9a0e3ba073d7f8cbcc33cb8a9d7254eff98d4230960e8e02eaf189da50afc52550fa0bfb620535240603e78e23975fa04ec78737125c57ef0e1295ae4791dbd72eb98a0dc02979f215e06a5c38819bd42767a8e1856ac98d0eebddd136bc18395e01f9780",
				"0xf90211a0b45de0172eb2119f86e9a53414150582d4052e90d9c6bbdd124d17e90a8118c5a05da2d2141759d4ef34f007c423b60b148d190d35d04f261cf5e657e4549782e8a0acbb3eef7fd8fd5f484aed84d2cb3416e46599f6d29efc6be6c63666f72e8e9ca02ab8b018174045d2c79ae86ef00d70c7cdd964cfe225ce36c14506b88a2f1a96a031d430d208f212e6100cfe74c4c694fa18c55bc0834f6ebec8a7d413039bfb63a05dec69eb4414309d3e7cd09c929692fe28b1a0cb116635d0b57e9f712716ebb0a01205d9623ad31d97cbd6ef3c20008284386ddf66aee79984a8c006b9d411d46ea089f83ffcdedd51ca1ac27376be77c5c1ca9aa5712aba25e0f82c8c39f2e8b6c4a08a9dcb56eb25df8cbca5f7b18e43b7f6bbf70571c9ad924a1791e5aa55df2331a0e1fca8262813be4837902a1fb24584e3f504af383dc84aa7838d0369c942217aa0adfc616da1e64c42b4cb11886ccf6642222b43bff8d266caeefc572ca3e0fb65a01f5c53e0c2a0109df11c313845b7e46910004981044de1772aabba746d2ce9a5a0dd357d528f9f34a8a3ae37358496f91675b47f6133e6bd0568d581a840c6eeb4a09ece58670426075fec57d32fbac503973288de9eeb3545175b64ebe1cfad9930a0d3c0d1c48df816d9a5dc772d7f464e8872d49dbe6fcea0944b56663514237f84a066ab6c2f3748861ad36c6d19f8b2cc0ee17fca96ea862e4d3ce3beac233a75c980",
				"0xf90211a042ab718a6fbb71add03da39ed57525bddd6f8455f9e77ce7f33d603cd86b791aa080b40e9fc893fc8725853fb11b6e2665376a790ff68ef273c93f0b7025fccb28a00defb48ea51e2fdb1d6f353dc5724796c073d8d80cb38d517020200e39b23051a0236e4a72db3ea239bf7edc82e7f2e9276a0f05e6bd6309023a65eb2b8fa2559fa0bcae7ef6000f79902f615489ceb4a727b39326acdd4462664503fceb4a87d36ba06a1b1da69a20a57324962130a82d2de2ad7783f87e49c1536ae23912c9e8c866a098792dc09ed13762e2bcb1c21590395500fe3162e8b9b02433e821cdeff478b5a05bcf60cfe56c1d9d6a2d63a9e0a8e91f35ac851526159bc27ef3b989daa68b4aa03d93b7e3802a49e207f8aaa619dedd7a1d8f2be43ace5b641611c66a78dde10aa077a649d238a2c0fdcd00fa8058601d1d72437ee4fafd948c34d82475fbea902aa0fa62b7a999a0584fe8a57056c60257fc24d5c5bd59a6ac04cc0e23c2356555f4a0570a6bb6e76afeb523d7536b08a5f064675e8731daf902d12bd0c4472626f6aea07fdcf35ae85861ed95f3375ae18dc5ea4518b056a8ac157c32bb8abd2a147d53a04a413bcb2c143a4038755095c087593a197d15f31ee4c11399021d0928f7d50ea0be95324a2306fcf30b974a1283e2df2d7562c55875343ae1e29fa7a67e835fe8a0e05a31ba76f7ddd817b7010f427234b38fbd2f12a03c222dde19ecee94dcb43080",
				"0xf90211a056527244566aed48654b0ca3e95225defe8b8eaee17657bace6698d7744b06cda0118221e9076b2fdf85d2371af95c10b9e9539a9aa3e3f6947acc0c7958bbdeeda0eed1a6e3b203d643cf0b3349816a78974046a45972cce621ea4a1979c29c632ca0414fd647c29c1323fe866af7f73d88c7ba93ba3d3eedea38aebd7a800c34ac75a0595cc2c4b2d314f1c12e7407732c4040bec3b269f499011c733b88f63007ae75a0836b374e20ca7ed03b5c5a447b6e15a39c0d037a82a429f8e099a06a24a372eaa085bcaf2bf96c47456365f4b28869670dee8d910c69ef7c75facc1c6e8e8983bfa0540629174895cf121ea76f7b63647bd9c46378184aeff99c81be646e0e6b80bca03048c27521797b94c190972758f2592a298af84bc14a2af2d1cfdcd00dc4b4a3a04456762d4c42229917c26b66106d1b354d12de086b65ad9c4ac0c97ae1fee28ba07b5ec4d7acfb65d85ce3c2c5c8528ea71e03997ddfe4a849cf02af2b61334276a0d49449f71c85a405d1c3ac16e903f8e2333c5e86c2b294e624fb0084317e4183a09d66aac7c383422398f421567ad2b103ce002961e973c8321e03b8e26bbb4c9da0cbccafd20b1c1162663b03bf8387639db8b1472d344dbda2f85805ea3f7061dca0402abdf44ab8e0971f58fadece8c64c1d9fabe5b2bc9ffaa30c847df16613520a03e3e95cd292c1fc017d5386e0c65923e423cddafe918bc50d6cd4ba96e8a4fe580",
				"0xf90211a054ecd08900814f3b9e373cbc4ea3701007e84703bc5af9cfaa9e1711fd3f0319a09f9261744412d1946bcd933e1a6d53102186b2d99ecff30e403feea9841cf053a067f9200eef6cd28b708d6292b57e2a42b577874e5b1cbd406741e61a483e6b7aa081b4a58834d68fdb58de1b2f4b422b06f6094ceea952f8d0bf4746d743eefd70a02f59d7a6832a735d3116657cb0941143aa82d9c6d38efa250bf5827071bf63e4a09007eccc09dc7929e3574e1612a8867660accce899f70546173cbe30b109ea94a08d3f6cbecc09531cb79bdde35403a8b1192cfcdf6f0aca0a4429e5708920a254a031f06093af171b8380db7bff2e61463f6c4dc01c8d7e7181cf642bad24202357a0a60e1315be595edd2f36b0c8070638278ebfc3d1b02d9ad4828a690564f98c8ea0e4e3830e486521cc62bc05be33fb82dc976be5f77031cb9cff945c6fd9c07f6ba043a6e474ef36831d726f8260533cde7ee60409a3ebfc4660f0a0ec15f36098dda01af29c4ae492c1ef17d49e759c535e83a0f10e5be08e9218ad50a9a18a058604a096d3e05bb50ee4b1cec89a4eefea21fc8939fa5d1d698dddee320266d3245853a0ad45735c19f6fdd0d89dbbaa7aae4599b0e3fb099ae36b48410fb55231a1facca09cd3816e190ebd4bfefeb9cbff3ac539ad05ace88ca3dbae324326b61b52b2b4a0084ed25dc92cd64bf00f64818b2d4a3655c038035abe7daf5b04b9949522263780",
				"0xf90191a00a7a0118e00981ab321049c9d340cd52c3a4781037540f7c48d0fdc27e899b3280a08537f2e248702a6ae2a57e9110a5740f5772c876389739ac90debd6a0692713ea00b3a26a05b5494fb3ff6f0b3897688a5581066b20b07ebab9252d169d928717fa0a9a54d84976d134d6dba06a65064c7f3a964a75947d452db6f6bb4b6c47b43aaa01e2a1ed3d1572b872bbf09ee44d2ed737da31f01de3c0f4b4e1f046740066461a098be0d129102f96ae636f9928911ed66103ac7c62235241aa0ef7e5b4b7e289ea07da2bce701255847cf5169ba5a7578a9700133f7ce13fa26a1d4097c20d1e0fda00631e0c3a589b999513f1d44a9b11e53d81027c1019553b4100542dbcb21031fa0c8d71dd13d2806e2865a5c2cfa447f626471bf0b66182a8fd07230434e1cad2680a0e9864fdfaf3693b2602f56cd938ccd494b8634b1f91800ef02203a3609ca4c21a0c69d174ad6b6e58b0bd05914352839ec60915cd066dd2bee2a48016139687f21a0513dd5514fd6bad56871711441d38de2821cc6913cb192416b0385f025650731808080",
				"0xf8669d3802a763f7db875346d03fbf86f137de55814b191c069e721f47474733b846f8440101a065d17ccfe8328a42712f5dcd7a8827eebab3341e1a8bd6a4cb741495b83bd026a0b44fb4e949d0f78f87f79ee46428f23a2a5713ce6fc6e0beb3dda78c2ac1ea55",
			},
			expextedValue:               "0x",
			expectedLongestPrefixPrefix: "ab14d68",
			expectedLastNode:            "0xf8669d3802a763f7db875346d03fbf86f137de55814b191c069e721f47474733b846f8440101a065d17ccfe8328a42712f5dcd7a8827eebab3341e1a8bd6a4cb741495b83bd026a0b44fb4e949d0f78f87f79ee46428f23a2a5713ce6fc6e0beb3dda78c2ac1ea55",
		},
		{
			desc: "Valid exclusion proof #2",
			root: "0x6f39539da0b571e36e04cdee1ef9273ce168644d63822352f3a18c0504220166",
			key:  "0xab14d6f000000000000000000000000000000000000000000000000000000000",
			proof: []string{
				"0xf90211a026b1f06f4804f169f036ab0cbe42d00ac13da34d6ff9af107d1af41d425eef8aa097d5cf7afeb561c87f064e5055cb5d6634cb7f6e93636b5d2c819b73db9f60f2a05d21cb1cb916ce9e57594a3c31058d2a936a5d7f790da95c5fab850252ff93a1a0fcf8847652abe31e2c5096e87128d93fb2de640534deffd530c82dff3d8f30fba026930d0a381d96f36abdb8857d6d8e91c0acd22b2ba78399e573798622638067a0417e2530f32e6c01a6cbdda472b853b98076436110df109ffad71e9b1a685feaa04f9c68013dc5f94ec91621bda78d233590b5d91f846698e60b42b28e0c87d935a0a06147fa3e0b0272661f8402f048b2cf7061617684bf44d4154d889c3f5119faa039164ab7071740ceeeb687f999251decfbe781ac0fd31cb63624c2f4b99bca7fa089443797be9cb0c265caeabcef452c23fbfcbac12cfe58a31f7d3ad6fb0d53f1a08357d7d9cf194e90858d2186918f396ed971062fcf939b78cd5b43e7fb44eb9ca0f0e23f2080dcd0af55e7ef37a9ccab09716a4c31e0b1b2172ac416c53df675a4a0ba840647ec3e2f7d38317c7dc0a6f8e3ae14b398c454ce9e76b0232d782f66caa081b03c3a644f130eb88395dac6ecad40b1ba14826fa4fef38f82ef2fa63e2aeea01b356b63266c0771c8167162e5472815b7836bf50dc1b5d70ee982ddc619b719a08878b527ef493b8b571b73b5747370e12e1fe09dd003546b9755e39c9f98050880",
				"0xf90211a09a94b4b8ab62c73bec14fd530e704e997b09f51aa9b316cc9a3b5ad0164e2530a0dca0351f0790d336e42aca22aabdbb4be8ac126c7e95fba25b55a78179cf54eba004b4813a8cad85b721321b6738f362f79911db78db50ec2571873cc1e4b00512a0bcc74ae61638d722a57cb0507bd0f30dbef47f591df032a5422eb8b3aaa015d1a0a4f53957bf35159735d8902e208471ecfa0e31ccc6e577e9b56f81d061719de6a0e77e0548e98157dc2f18b9f694965df88abf3c78b3090990c329264f2ac84325a0672727f7669561b1ddd33117f51146d4712af029e26cf1fce30608a7e8f4291ea088827c096ac6ce1fd783b9e418bb37711e6b0b34b429e5b7a321fac508d1a8aea03030c43045ea3ac75803b189a7132d88d0956a86424e3207ee38a46f2f850566a0298e2ca265c46fd3096368b732a77977211a0c55d1246dbd990111cb26fc4c66a07afd6cb69666868b214b594819385a9c51ba41ce927b5e83e0282f258be4ec9ea090d5e48e6b82c7d3963a1a944575160cf6297e03589096defc75ee8ce3cae787a0d65839513fe22a30a61cd19481e3c0ccae670d0c5de3ce6427f862fa74b8f5b9a0e3ba073d7f8cbcc33cb8a9d7254eff98d4230960e8e02eaf189da50afc52550fa0bfb620535240603e78e23975fa04ec78737125c57ef0e1295ae4791dbd72eb98a0dc02979f215e06a5c38819bd42767a8e1856ac98d0eebddd136bc18395e01f9780",
				"0xf90211a0b45de0172eb2119f86e9a53414150582d4052e90d9c6bbdd124d17e90a8118c5a05da2d2141759d4ef34f007c423b60b148d190d35d04f261cf5e657e4549782e8a0acbb3eef7fd8fd5f484aed84d2cb3416e46599f6d29efc6be6c63666f72e8e9ca02ab8b018174045d2c79ae86ef00d70c7cdd964cfe225ce36c14506b88a2f1a96a031d430d208f212e6100cfe74c4c694fa18c55bc0834f6ebec8a7d413039bfb63a05dec69eb4414309d3e7cd09c929692fe28b1a0cb116635d0b57e9f712716ebb0a01205d9623ad31d97cbd6ef3c20008284386ddf66aee79984a8c006b9d411d46ea089f83ffcdedd51ca1ac27376be77c5c1ca9aa5712aba25e0f82c8c39f2e8b6c4a08a9dcb56eb25df8cbca5f7b18e43b7f6bbf70571c9ad924a1791e5aa55df2331a0e1fca8262813be4837902a1fb24584e3f504af383dc84aa7838d0369c942217aa0adfc616da1e64c42b4cb11886ccf6642222b43bff8d266caeefc572ca3e0fb65a01f5c53e0c2a0109df11c313845b7e46910004981044de1772aabba746d2ce9a5a0dd357d528f9f34a8a3ae37358496f91675b47f6133e6bd0568d581a840c6eeb4a09ece58670426075fec57d32fbac503973288de9eeb3545175b64ebe1cfad9930a0d3c0d1c48df816d9a5dc772d7f464e8872d49dbe6fcea0944b56663514237f84a066ab6c2f3748861ad36c6d19f8b2cc0ee17fca96ea862e4d3ce3beac233a75c980",
				"0xf90211a042ab718a6fbb71add03da39ed57525bddd6f8455f9e77ce7f33d603cd86b791aa080b40e9fc893fc8725853fb11b6e2665376a790ff68ef273c93f0b7025fccb28a00defb48ea51e2fdb1d6f353dc5724796c073d8d80cb38d517020200e39b23051a0236e4a72db3ea239bf7edc82e7f2e9276a0f05e6bd6309023a65eb2b8fa2559fa0bcae7ef6000f79902f615489ceb4a727b39326acdd4462664503fceb4a87d36ba06a1b1da69a20a57324962130a82d2de2ad7783f87e49c1536ae23912c9e8c866a098792dc09ed13762e2bcb1c21590395500fe3162e8b9b02433e821cdeff478b5a05bcf60cfe56c1d9d6a2d63a9e0a8e91f35ac851526159bc27ef3b989daa68b4aa03d93b7e3802a49e207f8aaa619dedd7a1d8f2be43ace5b641611c66a78dde10aa077a649d238a2c0fdcd00fa8058601d1d72437ee4fafd948c34d82475fbea902aa0fa62b7a999a0584fe8a57056c60257fc24d5c5bd59a6ac04cc0e23c2356555f4a0570a6bb6e76afeb523d7536b08a5f064675e8731daf902d12bd0c4472626f6aea07fdcf35ae85861ed95f3375ae18dc5ea4518b056a8ac157c32bb8abd2a147d53a04a413bcb2c143a4038755095c087593a197d15f31ee4c11399021d0928f7d50ea0be95324a2306fcf30b974a1283e2df2d7562c55875343ae1e29fa7a67e835fe8a0e05a31ba76f7ddd817b7010f427234b38fbd2f12a03c222dde19ecee94dcb43080",
				"0xf90211a056527244566aed48654b0ca3e95225defe8b8eaee17657bace6698d7744b06cda0118221e9076b2fdf85d2371af95c10b9e9539a9aa3e3f6947acc0c7958bbdeeda0eed1a6e3b203d643cf0b3349816a78974046a45972cce621ea4a1979c29c632ca0414fd647c29c1323fe866af7f73d88c7ba93ba3d3eedea38aebd7a800c34ac75a0595cc2c4b2d314f1c12e7407732c4040bec3b269f499011c733b88f63007ae75a0836b374e20ca7ed03b5c5a447b6e15a39c0d037a82a429f8e099a06a24a372eaa085bcaf2bf96c47456365f4b28869670dee8d910c69ef7c75facc1c6e8e8983bfa0540629174895cf121ea76f7b63647bd9c46378184aeff99c81be646e0e6b80bca03048c27521797b94c190972758f2592a298af84bc14a2af2d1cfdcd00dc4b4a3a04456762d4c42229917c26b66106d1b354d12de086b65ad9c4ac0c97ae1fee28ba07b5ec4d7acfb65d85ce3c2c5c8528ea71e03997ddfe4a849cf02af2b61334276a0d49449f71c85a405d1c3ac16e903f8e2333c5e86c2b294e624fb0084317e4183a09d66aac7c383422398f421567ad2b103ce002961e973c8321e03b8e26bbb4c9da0cbccafd20b1c1162663b03bf8387639db8b1472d344dbda2f85805ea3f7061dca0402abdf44ab8e0971f58fadece8c64c1d9fabe5b2bc9ffaa30c847df16613520a03e3e95cd292c1fc017d5386e0c65923e423cddafe918bc50d6cd4ba96e8a4fe580",
				"0xf90211a054ecd08900814f3b9e373cbc4ea3701007e84703bc5af9cfaa9e1711fd3f0319a09f9261744412d1946bcd933e1a6d53102186b2d99ecff30e403feea9841cf053a067f9200eef6cd28b708d6292b57e2a42b577874e5b1cbd406741e61a483e6b7aa081b4a58834d68fdb58de1b2f4b422b06f6094ceea952f8d0bf4746d743eefd70a02f59d7a6832a735d3116657cb0941143aa82d9c6d38efa250bf5827071bf63e4a09007eccc09dc7929e3574e1612a8867660accce899f70546173cbe30b109ea94a08d3f6cbecc09531cb79bdde35403a8b1192cfcdf6f0aca0a4429e5708920a254a031f06093af171b8380db7bff2e61463f6c4dc01c8d7e7181cf642bad24202357a0a60e1315be595edd2f36b0c8070638278ebfc3d1b02d9ad4828a690564f98c8ea0e4e3830e486521cc62bc05be33fb82dc976be5f77031cb9cff945c6fd9c07f6ba043a6e474ef36831d726f8260533cde7ee60409a3ebfc4660f0a0ec15f36098dda01af29c4ae492c1ef17d49e759c535e83a0f10e5be08e9218ad50a9a18a058604a096d3e05bb50ee4b1cec89a4eefea21fc8939fa5d1d698dddee320266d3245853a0ad45735c19f6fdd0d89dbbaa7aae4599b0e3fb099ae36b48410fb55231a1facca09cd3816e190ebd4bfefeb9cbff3ac539ad05ace88ca3dbae324326b61b52b2b4a0084ed25dc92cd64bf00f64818b2d4a3655c038035abe7daf5b04b9949522263780",
				"0xf90191a00a7a0118e00981ab321049c9d340cd52c3a4781037540f7c48d0fdc27e899b3280a08537f2e248702a6ae2a57e9110a5740f5772c876389739ac90debd6a0692713ea00b3a26a05b5494fb3ff6f0b3897688a5581066b20b07ebab9252d169d928717fa0a9a54d84976d134d6dba06a65064c7f3a964a75947d452db6f6bb4b6c47b43aaa01e2a1ed3d1572b872bbf09ee44d2ed737da31f01de3c0f4b4e1f046740066461a098be0d129102f96ae636f9928911ed66103ac7c62235241aa0ef7e5b4b7e289ea07da2bce701255847cf5169ba5a7578a9700133f7ce13fa26a1d4097c20d1e0fda00631e0c3a589b999513f1d44a9b11e53d81027c1019553b4100542dbcb21031fa0c8d71dd13d2806e2865a5c2cfa447f626471bf0b66182a8fd07230434e1cad2680a0e9864fdfaf3693b2602f56cd938ccd494b8634b1f91800ef02203a3609ca4c21a0c69d174ad6b6e58b0bd05914352839ec60915cd066dd2bee2a48016139687f21a0513dd5514fd6bad56871711441d38de2821cc6913cb192416b0385f025650731808080",
				"0xf8669d3802a763f7db875346d03fbf86f137de55814b191c069e721f47474733b846f8440101a065d17ccfe8328a42712f5dcd7a8827eebab3341e1a8bd6a4cb741495b83bd026a0b44fb4e949d0f78f87f79ee46428f23a2a5713ce6fc6e0beb3dda78c2ac1ea55",
			},
			expextedValue:               "0x",
			expectedLongestPrefixPrefix: "ab14d6",
			expectedLastNode:            "0xf90191a00a7a0118e00981ab321049c9d340cd52c3a4781037540f7c48d0fdc27e899b3280a08537f2e248702a6ae2a57e9110a5740f5772c876389739ac90debd6a0692713ea00b3a26a05b5494fb3ff6f0b3897688a5581066b20b07ebab9252d169d928717fa0a9a54d84976d134d6dba06a65064c7f3a964a75947d452db6f6bb4b6c47b43aaa01e2a1ed3d1572b872bbf09ee44d2ed737da31f01de3c0f4b4e1f046740066461a098be0d129102f96ae636f9928911ed66103ac7c62235241aa0ef7e5b4b7e289ea07da2bce701255847cf5169ba5a7578a9700133f7ce13fa26a1d4097c20d1e0fda00631e0c3a589b999513f1d44a9b11e53d81027c1019553b4100542dbcb21031fa0c8d71dd13d2806e2865a5c2cfa447f626471bf0b66182a8fd07230434e1cad2680a0e9864fdfaf3693b2602f56cd938ccd494b8634b1f91800ef02203a3609ca4c21a0c69d174ad6b6e58b0bd05914352839ec60915cd066dd2bee2a48016139687f21a0513dd5514fd6bad56871711441d38de2821cc6913cb192416b0385f025650731808080",
		},
	}

	for _, tc := range testCase {
		t.Run(tc.desc, func(t *testing.T) {
			// Create a proofDB compatible with trie.VerifyProof
			proofDB := memorydb.New()
			err := StoreHexProofs(tc.proof, proofDB)
			require.NoError(t, err)

			// Verify the proof
			val, longestPrefix, lastNode, err := VerifyProofWithLastNode(common.HexToHash(tc.root), hexutil.MustDecode(tc.key), proofDB)
			if tc.expectedError != nil {
				require.Error(t, err)
				assert.Equal(t, tc.expectedError, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.expectedLongestPrefixPrefix, encodeKey(longestPrefix), "Last key prefix mismatch")
				assert.Equal(t, tc.expextedValue, hexutil.Encode(val), "Value mismatch")
				assert.Equal(t, tc.expectedLastNode, hexutil.Encode(lastNode), "Last node mismatch")
			}
		})
	}
}

const hextable = "0123456789abcdef"

func encodeKey(hex []byte) string {
	if hasTerm(hex) {
		hex = hex[:len(hex)-1]
	}

	key := make([]byte, len(hex))
	for i, b := range hex {
		key[i] = hextable[b]
	}

	return string(key)
}

func TestOneElementProof(t *testing.T) {
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	updateString(trie, "k", "v")
	for i, prover := range makeProvers(trie) {
		proof := prover([]byte("k"))
		if proof == nil {
			t.Fatalf("prover %d: nil proof", i)
		}
		if proof.Len() != 1 {
			t.Errorf("prover %d: proof should have one element", i)
		}
		val, err := VerifyProof(trie.Hash(), []byte("k"), proof)
		if err != nil {
			t.Fatalf("prover %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if !bytes.Equal(val, []byte("v")) {
			t.Fatalf("prover %d: verified value mismatch: have %x, want 'k'", i, val)
		}
	}
}

func TestBadProof(t *testing.T) {
	trie, vals := randomTrie(800)
	root := trie.Hash()
	for i, prover := range makeProvers(trie) {
		for _, kv := range vals {
			proof := prover(kv.k)
			if proof == nil {
				t.Fatalf("prover %d: nil proof", i)
			}
			it := proof.NewIterator(nil, nil)
			for i, d := 0, mrand.Intn(proof.Len()); i <= d; i++ {
				it.Next()
			}
			key := it.Key()
			val, _ := proof.Get(key)
			proof.Delete(key)
			it.Release()

			mutateByte(val)
			proof.Put(crypto.Keccak256(val), val)

			if _, err := VerifyProof(root, kv.k, proof); err == nil {
				t.Fatalf("prover %d: expected proof to fail for key %x", i, kv.k)
			}
		}
	}
}

// Tests that missing keys can also be proven. The test explicitly uses a single
// entry trie and checks for missing keys both before and after the single entry.
func TestMissingKeyProof(t *testing.T) {
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	updateString(trie, "k", "v")

	for i, key := range []string{"a", "j", "l", "z"} {
		proof := memorydb.New()
		trie.Prove([]byte(key), proof)

		if proof.Len() != 1 {
			t.Errorf("test %d: proof should have one element", i)
		}
		val, err := VerifyProof(trie.Hash(), []byte(key), proof)
		if err != nil {
			t.Fatalf("test %d: failed to verify proof: %v\nraw proof: %x", i, err, proof)
		}
		if val != nil {
			t.Fatalf("test %d: verified value mismatch: have %x, want nil", i, val)
		}
	}
}

// TestRangeProof tests normal range proof with both edge proofs
// as the existent proof. The test cases are generated randomly.
func TestRangeProof(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)
	for i := 0; i < 500; i++ {
		start := mrand.Intn(len(entries))
		end := mrand.Intn(len(entries)-start) + start + 1

		proof := memorydb.New()
		if err := trie.Prove(entries[start].k, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		if err := trie.Prove(entries[end-1].k, proof); err != nil {
			t.Fatalf("Failed to prove the last node %v", err)
		}
		var keys [][]byte
		var vals [][]byte
		for i := start; i < end; i++ {
			keys = append(keys, entries[i].k)
			vals = append(vals, entries[i].v)
		}
		_, err := VerifyRangeProof(trie.Hash(), keys[0], keys, vals, proof)
		if err != nil {
			t.Fatalf("Case %d(%d->%d) expect no error, got %v", i, start, end-1, err)
		}
	}
}

// TestRangeProofWithNonExistentProof tests normal range proof with two non-existent proofs.
// The test cases are generated randomly.
func TestRangeProofWithNonExistentProof(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)
	for i := 0; i < 500; i++ {
		start := mrand.Intn(len(entries))
		end := mrand.Intn(len(entries)-start) + start + 1
		proof := memorydb.New()

		// Short circuit if the decreased key is same with the previous key
		first := decreaseKey(common.CopyBytes(entries[start].k))
		if start != 0 && bytes.Equal(first, entries[start-1].k) {
			continue
		}
		// Short circuit if the decreased key is underflow
		if bytes.Compare(first, entries[start].k) > 0 {
			continue
		}
		if err := trie.Prove(first, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		if err := trie.Prove(entries[end-1].k, proof); err != nil {
			t.Fatalf("Failed to prove the last node %v", err)
		}
		var keys [][]byte
		var vals [][]byte
		for i := start; i < end; i++ {
			keys = append(keys, entries[i].k)
			vals = append(vals, entries[i].v)
		}
		_, err := VerifyRangeProof(trie.Hash(), first, keys, vals, proof)
		if err != nil {
			t.Fatalf("Case %d(%d->%d) expect no error, got %v", i, start, end-1, err)
		}
	}
}

// TestRangeProofWithInvalidNonExistentProof tests such scenarios:
// - There exists a gap between the first element and the left edge proof
func TestRangeProofWithInvalidNonExistentProof(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	// Case 1
	start, end := 100, 200
	first := decreaseKey(common.CopyBytes(entries[start].k))

	proof := memorydb.New()
	if err := trie.Prove(first, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[end-1].k, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	start = 105 // Gap created
	k := make([][]byte, 0)
	v := make([][]byte, 0)
	for i := start; i < end; i++ {
		k = append(k, entries[i].k)
		v = append(v, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), first, k, v, proof)
	if err == nil {
		t.Fatalf("Expected to detect the error, got nil")
	}
}

// TestOneElementRangeProof tests the proof with only one
// element. The first edge proof can be existent one or
// non-existent one.
func TestOneElementRangeProof(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	// One element with existent edge proof, both edge proofs
	// point to the SAME key.
	start := 1000
	proof := memorydb.New()
	if err := trie.Prove(entries[start].k, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	_, err := VerifyRangeProof(trie.Hash(), entries[start].k, [][]byte{entries[start].k}, [][]byte{entries[start].v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// One element with left non-existent edge proof
	start = 1000
	first := decreaseKey(common.CopyBytes(entries[start].k))
	proof = memorydb.New()
	if err := trie.Prove(first, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[start].k, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), first, [][]byte{entries[start].k}, [][]byte{entries[start].v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// One element with right non-existent edge proof
	start = 1000
	last := increaseKey(common.CopyBytes(entries[start].k))
	proof = memorydb.New()
	if err := trie.Prove(entries[start].k, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), entries[start].k, [][]byte{entries[start].k}, [][]byte{entries[start].v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// One element with two non-existent edge proofs
	start = 1000
	first, last = decreaseKey(common.CopyBytes(entries[start].k)), increaseKey(common.CopyBytes(entries[start].k))
	proof = memorydb.New()
	if err := trie.Prove(first, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), first, [][]byte{entries[start].k}, [][]byte{entries[start].v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Test the mini trie with only a single element.
	tinyTrie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	entry := &kv{randBytes(32), randBytes(20), false}
	tinyTrie.MustUpdate(entry.k, entry.v)

	first = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000").Bytes()
	last = entry.k
	proof = memorydb.New()
	if err := tinyTrie.Prove(first, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := tinyTrie.Prove(last, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(tinyTrie.Hash(), first, [][]byte{entry.k}, [][]byte{entry.v}, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// TestAllElementsProof tests the range proof with all elements.
// The edge proofs can be nil.
func TestAllElementsProof(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	var k [][]byte
	var v [][]byte
	for i := 0; i < len(entries); i++ {
		k = append(k, entries[i].k)
		v = append(v, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), nil, k, v, nil)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// With edge proofs, it should still work.
	proof := memorydb.New()
	if err := trie.Prove(entries[0].k, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[len(entries)-1].k, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), k[0], k, v, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Even with non-existent edge proofs, it should still work.
	proof = memorydb.New()
	first := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000").Bytes()
	if err := trie.Prove(first, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[len(entries)-1].k, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), first, k, v, proof)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

// TestSingleSideRangeProof tests the range starts from zero.
func TestSingleSideRangeProof(t *testing.T) {
	for i := 0; i < 64; i++ {
		trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
		var entries []*kv
		for i := 0; i < 4096; i++ {
			value := &kv{randBytes(32), randBytes(20), false}
			trie.MustUpdate(value.k, value.v)
			entries = append(entries, value)
		}
		slices.SortFunc(entries, (*kv).cmp)

		var cases = []int{0, 1, 50, 100, 1000, 2000, len(entries) - 1}
		for _, pos := range cases {
			proof := memorydb.New()
			if err := trie.Prove(common.Hash{}.Bytes(), proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
			if err := trie.Prove(entries[pos].k, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
			k := make([][]byte, 0)
			v := make([][]byte, 0)
			for i := 0; i <= pos; i++ {
				k = append(k, entries[i].k)
				v = append(v, entries[i].v)
			}
			_, err := VerifyRangeProof(trie.Hash(), common.Hash{}.Bytes(), k, v, proof)
			if err != nil {
				t.Fatalf("Expected no error, got %v", err)
			}
		}
	}
}

// TestBadRangeProof tests a few cases which the proof is wrong.
// The prover is expected to detect the error.
func TestBadRangeProof(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	for i := 0; i < 500; i++ {
		start := mrand.Intn(len(entries))
		end := mrand.Intn(len(entries)-start) + start + 1
		proof := memorydb.New()
		if err := trie.Prove(entries[start].k, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		if err := trie.Prove(entries[end-1].k, proof); err != nil {
			t.Fatalf("Failed to prove the last node %v", err)
		}
		var keys [][]byte
		var vals [][]byte
		for i := start; i < end; i++ {
			keys = append(keys, entries[i].k)
			vals = append(vals, entries[i].v)
		}
		var first = keys[0]
		testcase := mrand.Intn(6)
		var index int
		switch testcase {
		case 0:
			// Modified key
			index = mrand.Intn(end - start)
			keys[index] = randBytes(32) // In theory it can't be same
		case 1:
			// Modified val
			index = mrand.Intn(end - start)
			vals[index] = randBytes(20) // In theory it can't be same
		case 2:
			// Gapped entry slice
			index = mrand.Intn(end - start)
			if (index == 0 && start < 100) || (index == end-start-1) {
				continue
			}
			keys = append(keys[:index], keys[index+1:]...)
			vals = append(vals[:index], vals[index+1:]...)
		case 3:
			// Out of order
			index1 := mrand.Intn(end - start)
			index2 := mrand.Intn(end - start)
			if index1 == index2 {
				continue
			}
			keys[index1], keys[index2] = keys[index2], keys[index1]
			vals[index1], vals[index2] = vals[index2], vals[index1]
		case 4:
			// Set random key to nil, do nothing
			index = mrand.Intn(end - start)
			keys[index] = nil
		case 5:
			// Set random value to nil, deletion
			index = mrand.Intn(end - start)
			vals[index] = nil
		}
		_, err := VerifyRangeProof(trie.Hash(), first, keys, vals, proof)
		if err == nil {
			t.Fatalf("%d Case %d index %d range: (%d->%d) expect error, got nil", i, testcase, index, start, end-1)
		}
	}
}

// TestGappedRangeProof focuses on the small trie with embedded nodes.
// If the gapped node is embedded in the trie, it should be detected too.
func TestGappedRangeProof(t *testing.T) {
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	var entries []*kv // Sorted entries
	for i := byte(0); i < 10; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		trie.MustUpdate(value.k, value.v)
		entries = append(entries, value)
	}
	first, last := 2, 8
	proof := memorydb.New()
	if err := trie.Prove(entries[first].k, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[last-1].k, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	var keys [][]byte
	var vals [][]byte
	for i := first; i < last; i++ {
		if i == (first+last)/2 {
			continue
		}
		keys = append(keys, entries[i].k)
		vals = append(vals, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), keys[0], keys, vals, proof)
	if err == nil {
		t.Fatal("expect error, got nil")
	}
}

// TestSameSideProofs tests the element is not in the range covered by proofs
func TestSameSideProofs(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	pos := 1000
	first := common.CopyBytes(entries[0].k)

	proof := memorydb.New()
	if err := trie.Prove(first, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[2000].k, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	_, err := VerifyRangeProof(trie.Hash(), first, [][]byte{entries[pos].k}, [][]byte{entries[pos].v}, proof)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}

	first = increaseKey(common.CopyBytes(entries[pos].k))
	last := increaseKey(common.CopyBytes(entries[pos].k))
	last = increaseKey(last)

	proof = memorydb.New()
	if err := trie.Prove(first, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(last, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	_, err = VerifyRangeProof(trie.Hash(), first, [][]byte{entries[pos].k}, [][]byte{entries[pos].v}, proof)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
}

func TestHasRightElement(t *testing.T) {
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	var entries []*kv
	for i := 0; i < 4096; i++ {
		value := &kv{randBytes(32), randBytes(20), false}
		trie.MustUpdate(value.k, value.v)
		entries = append(entries, value)
	}
	slices.SortFunc(entries, (*kv).cmp)

	var cases = []struct {
		start   int
		end     int
		hasMore bool
	}{
		{-1, 1, true}, // single element with non-existent left proof
		{0, 1, true},  // single element with existent left proof
		{0, 10, true},
		{50, 100, true},
		{50, len(entries), false},               // No more element expected
		{len(entries) - 1, len(entries), false}, // Single last element with two existent proofs(point to same key)
		{0, len(entries), false},                // The whole set with existent left proof
		{-1, len(entries), false},               // The whole set with non-existent left proof
	}
	for _, c := range cases {
		var (
			firstKey []byte
			start    = c.start
			end      = c.end
			proof    = memorydb.New()
		)
		if c.start == -1 {
			firstKey, start = common.Hash{}.Bytes(), 0
			if err := trie.Prove(firstKey, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
		} else {
			firstKey = entries[c.start].k
			if err := trie.Prove(entries[c.start].k, proof); err != nil {
				t.Fatalf("Failed to prove the first node %v", err)
			}
		}
		if err := trie.Prove(entries[c.end-1].k, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		k := make([][]byte, 0)
		v := make([][]byte, 0)
		for i := start; i < end; i++ {
			k = append(k, entries[i].k)
			v = append(v, entries[i].v)
		}
		hasMore, err := VerifyRangeProof(trie.Hash(), firstKey, k, v, proof)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if hasMore != c.hasMore {
			t.Fatalf("Wrong hasMore indicator, want %t, got %t", c.hasMore, hasMore)
		}
	}
}

// TestEmptyRangeProof tests the range proof with "no" element.
// The first edge proof must be a non-existent proof.
func TestEmptyRangeProof(t *testing.T) {
	trie, vals := randomTrie(4096)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	var cases = []struct {
		pos int
		err bool
	}{
		{len(entries) - 1, false},
		{500, true},
	}
	for _, c := range cases {
		proof := memorydb.New()
		first := increaseKey(common.CopyBytes(entries[c.pos].k))
		if err := trie.Prove(first, proof); err != nil {
			t.Fatalf("Failed to prove the first node %v", err)
		}
		_, err := VerifyRangeProof(trie.Hash(), first, nil, nil, proof)
		if c.err && err == nil {
			t.Fatalf("Expected error, got nil")
		}
		if !c.err && err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
	}
}

// TestBloatedProof tests a malicious proof, where the proof is more or less the
// whole trie. Previously we didn't accept such packets, but the new APIs do, so
// lets leave this test as a bit weird, but present.
func TestBloatedProof(t *testing.T) {
	// Use a small trie
	trie, kvs := nonRandomTrie(100)
	var entries []*kv
	for _, kv := range kvs {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)
	var keys [][]byte
	var vals [][]byte

	proof := memorydb.New()
	// In the 'malicious' case, we add proofs for every single item
	// (but only one key/value pair used as leaf)
	for i, entry := range entries {
		trie.Prove(entry.k, proof)
		if i == 50 {
			keys = append(keys, entry.k)
			vals = append(vals, entry.v)
		}
	}
	// For reference, we use the same function, but _only_ prove the first
	// and last element
	want := memorydb.New()
	trie.Prove(keys[0], want)
	trie.Prove(keys[len(keys)-1], want)

	if _, err := VerifyRangeProof(trie.Hash(), keys[0], keys, vals, proof); err != nil {
		t.Fatalf("expected bloated proof to succeed, got %v", err)
	}
}

// TestEmptyValueRangeProof tests normal range proof with both edge proofs
// as the existent proof, but with an extra empty value included, which is a
// noop technically, but practically should be rejected.
func TestEmptyValueRangeProof(t *testing.T) {
	trie, values := randomTrie(512)
	var entries []*kv
	for _, kv := range values {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	// Create a new entry with a slightly modified key
	mid := len(entries) / 2
	key := common.CopyBytes(entries[mid-1].k)
	for n := len(key) - 1; n >= 0; n-- {
		if key[n] < 0xff {
			key[n]++
			break
		}
	}
	noop := &kv{key, []byte{}, false}
	entries = append(append(append([]*kv{}, entries[:mid]...), noop), entries[mid:]...)

	start, end := 1, len(entries)-1

	proof := memorydb.New()
	if err := trie.Prove(entries[start].k, proof); err != nil {
		t.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[end-1].k, proof); err != nil {
		t.Fatalf("Failed to prove the last node %v", err)
	}
	var keys [][]byte
	var vals [][]byte
	for i := start; i < end; i++ {
		keys = append(keys, entries[i].k)
		vals = append(vals, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), keys[0], keys, vals, proof)
	if err == nil {
		t.Fatalf("Expected failure on noop entry")
	}
}

// TestAllElementsEmptyValueRangeProof tests the range proof with all elements,
// but with an extra empty value included, which is a noop technically, but
// practically should be rejected.
func TestAllElementsEmptyValueRangeProof(t *testing.T) {
	trie, values := randomTrie(512)
	var entries []*kv
	for _, kv := range values {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	// Create a new entry with a slightly modified key
	mid := len(entries) / 2
	key := common.CopyBytes(entries[mid-1].k)
	for n := len(key) - 1; n >= 0; n-- {
		if key[n] < 0xff {
			key[n]++
			break
		}
	}
	noop := &kv{key, []byte{}, false}
	entries = append(append(append([]*kv{}, entries[:mid]...), noop), entries[mid:]...)

	var keys [][]byte
	var vals [][]byte
	for i := 0; i < len(entries); i++ {
		keys = append(keys, entries[i].k)
		vals = append(vals, entries[i].v)
	}
	_, err := VerifyRangeProof(trie.Hash(), nil, keys, vals, nil)
	if err == nil {
		t.Fatalf("Expected failure on noop entry")
	}
}

// mutateByte changes one byte in b.
func mutateByte(b []byte) {
	for r := mrand.Intn(len(b)); ; {
		new := byte(mrand.Intn(255))
		if new != b[r] {
			b[r] = new
			break
		}
	}
}

func increaseKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]++
		if key[i] != 0x0 {
			break
		}
	}
	return key
}

func decreaseKey(key []byte) []byte {
	for i := len(key) - 1; i >= 0; i-- {
		key[i]--
		if key[i] != 0xff {
			break
		}
	}
	return key
}

func BenchmarkProve(b *testing.B) {
	trie, vals := randomTrie(100)
	var keys []string
	for k := range vals {
		keys = append(keys, k)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		kv := vals[keys[i%len(keys)]]
		proofs := memorydb.New()
		if trie.Prove(kv.k, proofs); proofs.Len() == 0 {
			b.Fatalf("zero length proof for %x", kv.k)
		}
	}
}

func BenchmarkVerifyProof(b *testing.B) {
	trie, vals := randomTrie(100)
	root := trie.Hash()
	var keys []string
	var proofs []*memorydb.Database
	for k := range vals {
		keys = append(keys, k)
		proof := memorydb.New()
		trie.Prove([]byte(k), proof)
		proofs = append(proofs, proof)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		im := i % len(keys)
		if _, err := VerifyProof(root, []byte(keys[im]), proofs[im]); err != nil {
			b.Fatalf("key %x: %v", keys[im], err)
		}
	}
}

func BenchmarkVerifyRangeProof10(b *testing.B)   { benchmarkVerifyRangeProof(b, 10) }
func BenchmarkVerifyRangeProof100(b *testing.B)  { benchmarkVerifyRangeProof(b, 100) }
func BenchmarkVerifyRangeProof1000(b *testing.B) { benchmarkVerifyRangeProof(b, 1000) }
func BenchmarkVerifyRangeProof5000(b *testing.B) { benchmarkVerifyRangeProof(b, 5000) }

func benchmarkVerifyRangeProof(b *testing.B, size int) {
	trie, vals := randomTrie(8192)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	start := 2
	end := start + size
	proof := memorydb.New()
	if err := trie.Prove(entries[start].k, proof); err != nil {
		b.Fatalf("Failed to prove the first node %v", err)
	}
	if err := trie.Prove(entries[end-1].k, proof); err != nil {
		b.Fatalf("Failed to prove the last node %v", err)
	}
	var keys [][]byte
	var values [][]byte
	for i := start; i < end; i++ {
		keys = append(keys, entries[i].k)
		values = append(values, entries[i].v)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := VerifyRangeProof(trie.Hash(), keys[0], keys, values, proof)
		if err != nil {
			b.Fatalf("Case %d(%d->%d) expect no error, got %v", i, start, end-1, err)
		}
	}
}

func BenchmarkVerifyRangeNoProof10(b *testing.B)   { benchmarkVerifyRangeNoProof(b, 100) }
func BenchmarkVerifyRangeNoProof500(b *testing.B)  { benchmarkVerifyRangeNoProof(b, 500) }
func BenchmarkVerifyRangeNoProof1000(b *testing.B) { benchmarkVerifyRangeNoProof(b, 1000) }

func benchmarkVerifyRangeNoProof(b *testing.B, size int) {
	trie, vals := randomTrie(size)
	var entries []*kv
	for _, kv := range vals {
		entries = append(entries, kv)
	}
	slices.SortFunc(entries, (*kv).cmp)

	var keys [][]byte
	var values [][]byte
	for _, entry := range entries {
		keys = append(keys, entry.k)
		values = append(values, entry.v)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := VerifyRangeProof(trie.Hash(), keys[0], keys, values, nil)
		if err != nil {
			b.Fatalf("Expected no error, got %v", err)
		}
	}
}

func randomTrie(n int) (*Trie, map[string]*kv) {
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	vals := make(map[string]*kv)
	for i := byte(0); i < 100; i++ {
		value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		value2 := &kv{common.LeftPadBytes([]byte{i + 10}, 32), []byte{i}, false}
		trie.MustUpdate(value.k, value.v)
		trie.MustUpdate(value2.k, value2.v)
		vals[string(value.k)] = value
		vals[string(value2.k)] = value2
	}
	for i := 0; i < n; i++ {
		value := &kv{randBytes(32), randBytes(20), false}
		trie.MustUpdate(value.k, value.v)
		vals[string(value.k)] = value
	}
	return trie, vals
}

func nonRandomTrie(n int) (*Trie, map[string]*kv) {
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	vals := make(map[string]*kv)
	max := uint64(0xffffffffffffffff)
	for i := uint64(0); i < uint64(n); i++ {
		value := make([]byte, 32)
		key := make([]byte, 32)
		binary.LittleEndian.PutUint64(key, i)
		binary.LittleEndian.PutUint64(value, i-max)
		//value := &kv{common.LeftPadBytes([]byte{i}, 32), []byte{i}, false}
		elem := &kv{key, value, false}
		trie.MustUpdate(elem.k, elem.v)
		vals[string(elem.k)] = elem
	}
	return trie, vals
}

func TestRangeProofKeysWithSharedPrefix(t *testing.T) {
	keys := [][]byte{
		common.Hex2Bytes("aa10000000000000000000000000000000000000000000000000000000000000"),
		common.Hex2Bytes("aa20000000000000000000000000000000000000000000000000000000000000"),
	}
	vals := [][]byte{
		common.Hex2Bytes("02"),
		common.Hex2Bytes("03"),
	}
	trie := NewEmpty(newTestDatabase(rawdb.NewMemoryDatabase(), rawdb.HashScheme))
	for i, key := range keys {
		trie.MustUpdate(key, vals[i])
	}
	root := trie.Hash()
	proof := memorydb.New()
	start := common.Hex2Bytes("0000000000000000000000000000000000000000000000000000000000000000")
	if err := trie.Prove(start, proof); err != nil {
		t.Fatalf("failed to prove start: %v", err)
	}
	if err := trie.Prove(keys[len(keys)-1], proof); err != nil {
		t.Fatalf("failed to prove end: %v", err)
	}

	more, err := VerifyRangeProof(root, start, keys, vals, proof)
	if err != nil {
		t.Fatalf("failed to verify range proof: %v", err)
	}
	if more != false {
		t.Error("expected more to be false")
	}
}
