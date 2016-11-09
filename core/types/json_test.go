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

package types

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var unmarshalHeaderTests = map[string]struct {
	input     string
	wantHash  common.Hash
	wantError error
}{
	"block 0x1e2200": {
		input:    `{"difficulty":"0x311ca98cebfe","extraData":"0x7777772e62772e636f6d","gasLimit":"0x47db3d","gasUsed":"0x43760c","hash":"0x3724bc6b9dcd4a2b3a26e0ed9b821e7380b5b3d7dec7166c7983cead62a37e48","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0xbcdfc35b86bedf72f0cda046a3c16829a2ef41d1","mixHash":"0x1ccfddb506dac5afc09b6f92eb09a043ffc8e08f7592250af57b9c64c20f9b25","nonce":"0x670bd98c79585197","number":"0x1e2200","parentHash":"0xd3e13296d064e7344f20c57c57b67a022f6bf7741fa42428c2db77e91abdf1f8","receiptsRoot":"0xeeab1776c1fafbe853a8ee0c1bafe2e775a1b6fdb6ff3e9f9410ddd4514889ff","sha3Uncles":"0x5fbfa4ec8b089678c53b6798cc0d9260ea40a529e06d5300aae35596262e0eb3","size":"0x57f","stateRoot":"0x62ad2007e4a3f31ea98e5d2fd150d894887bafde36eeac7331a60ae12053ec76","timestamp":"0x579b82f2","totalDifficulty":"0x24fe813c101d00f97","transactions":["0xb293408e85735bfc78b35aa89de8b48e49641e3d82e3d52ea2d44ec42a4e88cf","0x124acc383ff2da6faa0357829084dae64945221af6f6f09da1d11688b779f939","0xee090208b6051c442ccdf9ec19f66389e604d342a6d71144c7227ce995bef46f"],"transactionsRoot":"0xce0042dd9af0c1923dd7f58ca6faa156d39d4ef39fdb65c5bcd1d4b4720096db","uncles":["0x6818a31d1f204cf640c952082940b68b8db6d1b39ee71f7efe0e3629ed5d7eb3"]}`,
		wantHash: common.HexToHash("0x3724bc6b9dcd4a2b3a26e0ed9b821e7380b5b3d7dec7166c7983cead62a37e48"),
	},
	"bad nonce": {
		input:     `{"difficulty":"0x311ca98cebfe","extraData":"0x7777772e62772e636f6d","gasLimit":"0x47db3d","gasUsed":"0x43760c","hash":"0x3724bc6b9dcd4a2b3a26e0ed9b821e7380b5b3d7dec7166c7983cead62a37e48","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0xbcdfc35b86bedf72f0cda046a3c16829a2ef41d1","mixHash":"0x1ccfddb506dac5afc09b6f92eb09a043ffc8e08f7592250af57b9c64c20f9b25","nonce":"0x670bd98c7958","number":"0x1e2200","parentHash":"0xd3e13296d064e7344f20c57c57b67a022f6bf7741fa42428c2db77e91abdf1f8","receiptsRoot":"0xeeab1776c1fafbe853a8ee0c1bafe2e775a1b6fdb6ff3e9f9410ddd4514889ff","sha3Uncles":"0x5fbfa4ec8b089678c53b6798cc0d9260ea40a529e06d5300aae35596262e0eb3","size":"0x57f","stateRoot":"0x62ad2007e4a3f31ea98e5d2fd150d894887bafde36eeac7331a60ae12053ec76","timestamp":"0x579b82f2","totalDifficulty":"0x24fe813c101d00f97","transactions":["0xb293408e85735bfc78b35aa89de8b48e49641e3d82e3d52ea2d44ec42a4e88cf","0x124acc383ff2da6faa0357829084dae64945221af6f6f09da1d11688b779f939","0xee090208b6051c442ccdf9ec19f66389e604d342a6d71144c7227ce995bef46f"],"transactionsRoot":"0xce0042dd9af0c1923dd7f58ca6faa156d39d4ef39fdb65c5bcd1d4b4720096db","uncles":["0x6818a31d1f204cf640c952082940b68b8db6d1b39ee71f7efe0e3629ed5d7eb3"]}`,
		wantError: errBadNonceSize,
	},
	"missing mixHash": {
		input:     `{"difficulty":"0x311ca98cebfe","extraData":"0x7777772e62772e636f6d","gasLimit":"0x47db3d","gasUsed":"0x43760c","hash":"0x3724bc6b9dcd4a2b3a26e0ed9b821e7380b5b3d7dec7166c7983cead62a37e48","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0xbcdfc35b86bedf72f0cda046a3c16829a2ef41d1","nonce":"0x670bd98c79585197","number":"0x1e2200","parentHash":"0xd3e13296d064e7344f20c57c57b67a022f6bf7741fa42428c2db77e91abdf1f8","receiptsRoot":"0xeeab1776c1fafbe853a8ee0c1bafe2e775a1b6fdb6ff3e9f9410ddd4514889ff","sha3Uncles":"0x5fbfa4ec8b089678c53b6798cc0d9260ea40a529e06d5300aae35596262e0eb3","size":"0x57f","stateRoot":"0x62ad2007e4a3f31ea98e5d2fd150d894887bafde36eeac7331a60ae12053ec76","timestamp":"0x579b82f2","totalDifficulty":"0x24fe813c101d00f97","transactions":["0xb293408e85735bfc78b35aa89de8b48e49641e3d82e3d52ea2d44ec42a4e88cf","0x124acc383ff2da6faa0357829084dae64945221af6f6f09da1d11688b779f939","0xee090208b6051c442ccdf9ec19f66389e604d342a6d71144c7227ce995bef46f"],"transactionsRoot":"0xce0042dd9af0c1923dd7f58ca6faa156d39d4ef39fdb65c5bcd1d4b4720096db","uncles":["0x6818a31d1f204cf640c952082940b68b8db6d1b39ee71f7efe0e3629ed5d7eb3"]}`,
		wantError: errMissingHeaderMixDigest,
	},
	"missing fields": {
		input:     `{"gasLimit":"0x47db3d","gasUsed":"0x43760c","hash":"0x3724bc6b9dcd4a2b3a26e0ed9b821e7380b5b3d7dec7166c7983cead62a37e48","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0xbcdfc35b86bedf72f0cda046a3c16829a2ef41d1","mixHash":"0x1ccfddb506dac5afc09b6f92eb09a043ffc8e08f7592250af57b9c64c20f9b25","nonce":"0x670bd98c79585197","number":"0x1e2200","parentHash":"0xd3e13296d064e7344f20c57c57b67a022f6bf7741fa42428c2db77e91abdf1f8","receiptsRoot":"0xeeab1776c1fafbe853a8ee0c1bafe2e775a1b6fdb6ff3e9f9410ddd4514889ff","sha3Uncles":"0x5fbfa4ec8b089678c53b6798cc0d9260ea40a529e06d5300aae35596262e0eb3","size":"0x57f","stateRoot":"0x62ad2007e4a3f31ea98e5d2fd150d894887bafde36eeac7331a60ae12053ec76","timestamp":"0x579b82f2","totalDifficulty":"0x24fe813c101d00f97","transactions":["0xb293408e85735bfc78b35aa89de8b48e49641e3d82e3d52ea2d44ec42a4e88cf","0x124acc383ff2da6faa0357829084dae64945221af6f6f09da1d11688b779f939","0xee090208b6051c442ccdf9ec19f66389e604d342a6d71144c7227ce995bef46f"],"transactionsRoot":"0xce0042dd9af0c1923dd7f58ca6faa156d39d4ef39fdb65c5bcd1d4b4720096db","uncles":["0x6818a31d1f204cf640c952082940b68b8db6d1b39ee71f7efe0e3629ed5d7eb3"]}`,
		wantError: errMissingHeaderFields,
	},
}

func TestUnmarshalHeader(t *testing.T) {
	for name, test := range unmarshalHeaderTests {
		var head *Header
		err := json.Unmarshal([]byte(test.input), &head)
		if !checkError(t, name, err, test.wantError) {
			continue
		}
		if head.Hash() != test.wantHash {
			t.Errorf("test %q: got hash %x, want %x", name, head.Hash(), test.wantHash)
			continue
		}
	}
}

func TestMarshalHeader(t *testing.T) {
	for name, test := range unmarshalHeaderTests {
		if test.wantError != nil {
			continue
		}
		var original *Header
		json.Unmarshal([]byte(test.input), &original)

		blob, err := json.Marshal(original)
		if err != nil {
			t.Errorf("test %q: failed to marshal header: %v", name, err)
			continue
		}
		var proced *Header
		if err := json.Unmarshal(blob, &proced); err != nil {
			t.Errorf("Test %q: failed to unmarshal marhsalled header: %v", name, err)
			continue
		}
		if !reflect.DeepEqual(original, proced) {
			t.Errorf("test %q: header mismatch: have %+v, want %+v", name, proced, original)
			continue
		}
	}
}

var unmarshalTransactionTests = map[string]struct {
	input     string
	wantHash  common.Hash
	wantFrom  common.Address
	wantError error
}{
	"value transfer": {
		input:    `{"blockHash":"0x0188a05dcc825bd1a05dab91bea0c03622542683446e56302eabb46097d4ae11","blockNumber":"0x1e478d","from":"0xf36c3f6c4a2ce8d353fb92d5cd10d19ce69ae689","gas":"0x15f90","gasPrice":"0x4a817c800","hash":"0xd91c08f1e27c5ce7e1f57d78d7c56a9ee446be07b9635d84d0475660ea8905e9","input":"0x","nonce":"0x58d","to":"0x88f252f674ac755feff877abf957d4aa05adce86","transactionIndex":"0x1","value":"0x19f0ec3ed71ec00","v":"0x1c","r":"0x53829f206c99b866672f987909d556cd1c2eb60e990a3425f65083977c14187b","s":"0x5cc52383e41c923ec7d63749c1f13a7236b540527ee5b9a78b3fb869a66f60e"}`,
		wantHash: common.HexToHash("0xd91c08f1e27c5ce7e1f57d78d7c56a9ee446be07b9635d84d0475660ea8905e9"),
		wantFrom: common.HexToAddress("0xf36c3f6c4a2ce8d353fb92d5cd10d19ce69ae689"),
	},
	"bad signature fields": {
		input:     `{"blockHash":"0x0188a05dcc825bd1a05dab91bea0c03622542683446e56302eabb46097d4ae11","blockNumber":"0x1e478d","from":"0xf36c3f6c4a2ce8d353fb92d5cd10d19ce69ae689","gas":"0x15f90","gasPrice":"0x4a817c800","hash":"0xd91c08f1e27c5ce7e1f57d78d7c56a9ee446be07b9635d84d0475660ea8905e9","input":"0x","nonce":"0x58d","to":"0x88f252f674ac755feff877abf957d4aa05adce86","transactionIndex":"0x1","value":"0x19f0ec3ed71ec00","v":"0x58","r":"0x53829f206c99b866672f987909d556cd1c2eb60e990a3425f65083977c14187b","s":"0x5cc52383e41c923ec7d63749c1f13a7236b540527ee5b9a78b3fb869a66f60e"}`,
		wantError: ErrInvalidSig,
	},
	"missing signature v": {
		input:     `{"blockHash":"0x0188a05dcc825bd1a05dab91bea0c03622542683446e56302eabb46097d4ae11","blockNumber":"0x1e478d","from":"0xf36c3f6c4a2ce8d353fb92d5cd10d19ce69ae689","gas":"0x15f90","gasPrice":"0x4a817c800","hash":"0xd91c08f1e27c5ce7e1f57d78d7c56a9ee446be07b9635d84d0475660ea8905e9","input":"0x","nonce":"0x58d","to":"0x88f252f674ac755feff877abf957d4aa05adce86","transactionIndex":"0x1","value":"0x19f0ec3ed71ec00","r":"0x53829f206c99b866672f987909d556cd1c2eb60e990a3425f65083977c14187b","s":"0x5cc52383e41c923ec7d63749c1f13a7236b540527ee5b9a78b3fb869a66f60e"}`,
		wantError: errMissingTxSignatureFields,
	},
	"missing signature fields": {
		input:     `{"blockHash":"0x0188a05dcc825bd1a05dab91bea0c03622542683446e56302eabb46097d4ae11","blockNumber":"0x1e478d","from":"0xf36c3f6c4a2ce8d353fb92d5cd10d19ce69ae689","gas":"0x15f90","gasPrice":"0x4a817c800","hash":"0xd91c08f1e27c5ce7e1f57d78d7c56a9ee446be07b9635d84d0475660ea8905e9","input":"0x","nonce":"0x58d","to":"0x88f252f674ac755feff877abf957d4aa05adce86","transactionIndex":"0x1","value":"0x19f0ec3ed71ec00"}`,
		wantError: errMissingTxSignatureFields,
	},
	"missing fields": {
		input:     `{"blockHash":"0x0188a05dcc825bd1a05dab91bea0c03622542683446e56302eabb46097d4ae11","blockNumber":"0x1e478d","from":"0xf36c3f6c4a2ce8d353fb92d5cd10d19ce69ae689","hash":"0xd91c08f1e27c5ce7e1f57d78d7c56a9ee446be07b9635d84d0475660ea8905e9","input":"0x","nonce":"0x58d","to":"0x88f252f674ac755feff877abf957d4aa05adce86","transactionIndex":"0x1","value":"0x19f0ec3ed71ec00","v":"0x1c","r":"0x53829f206c99b866672f987909d556cd1c2eb60e990a3425f65083977c14187b","s":"0x5cc52383e41c923ec7d63749c1f13a7236b540527ee5b9a78b3fb869a66f60e"}`,
		wantError: errMissingTxFields,
	},
}

func TestUnmarshalTransaction(t *testing.T) {
	for name, test := range unmarshalTransactionTests {
		var tx *Transaction
		err := json.Unmarshal([]byte(test.input), &tx)
		if !checkError(t, name, err, test.wantError) {
			continue
		}
		if tx.Hash() != test.wantHash {
			t.Errorf("test %q: got hash %x, want %x", name, tx.Hash(), test.wantHash)
			continue
		}
		from, err := tx.From()
		if err != nil {
			t.Errorf("test %q: From error %v", name, err)
		}
		if from != test.wantFrom {
			t.Errorf("test %q: sender mismatch: got %x, want %x", name, from, test.wantFrom)
		}
	}
}

func TestMarshalTransaction(t *testing.T) {
	for name, test := range unmarshalTransactionTests {
		if test.wantError != nil {
			continue
		}
		var original *Transaction
		json.Unmarshal([]byte(test.input), &original)

		blob, err := json.Marshal(original)
		if err != nil {
			t.Errorf("test %q: failed to marshal transaction: %v", name, err)
			continue
		}
		var proced *Transaction
		if err := json.Unmarshal(blob, &proced); err != nil {
			t.Errorf("Test %q: failed to unmarshal marhsalled transaction: %v", name, err)
			continue
		}
		proced.Hash() // hack private fields to pass deep equal
		if !reflect.DeepEqual(original, proced) {
			t.Errorf("test %q: transaction mismatch: have %+v, want %+v", name, proced, original)
			continue
		}
	}
}

var unmarshalReceiptTests = map[string]struct {
	input     string
	wantError error
}{
	"ok": {
		input: `{"blockHash":"0xad20a0f78d19d7857067a9c06e6411efeab7673e183e4a545f53b724bb7fabf0","blockNumber":"0x1e773b","contractAddress":null,"cumulativeGasUsed":"0x10cea","from":"0xdf21fa922215b1a56f5a6d6294e6e36c85a0acfb","gasUsed":"0xbae2","logs":[{"address":"0xbb9bc244d798123fde783fcc1c72d3bb8c189413","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000df21fa922215b1a56f5a6d6294e6e36c85a0acfb","0x00000000000000000000000032be343b94f860124dc4fee278fdcbd38c102d88"],"data":"0x0000000000000000000000000000000000000000000000027cfefc4f3f392700","blockNumber":"0x1e773b","transactionIndex":"0x1","transactionHash":"0x0b4cc7844537023b709953390e3881ec5b233703a8e8824dc03e13729a1bd95a","blockHash":"0xad20a0f78d19d7857067a9c06e6411efeab7673e183e4a545f53b724bb7fabf0","logIndex":"0x0"}],"logsBloom":"0x00000000000000020000000000020000000000000000000000000000000000000000000000000000000000000000000000040000000000000100000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000010000000000000000000000000000000000000000000000010000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000002002000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000","root":"0x6e8a06b2dac39ac5c9d4db5fb2a2a94ef7a6e5ec1c554079112112caf162998a","to":"0xbb9bc244d798123fde783fcc1c72d3bb8c189413","transactionHash":"0x0b4cc7844537023b709953390e3881ec5b233703a8e8824dc03e13729a1bd95a","transactionIndex":"0x1"}`,
	},
	"missing post state": {
		input:     `{"blockHash":"0xad20a0f78d19d7857067a9c06e6411efeab7673e183e4a545f53b724bb7fabf0","blockNumber":"0x1e773b","contractAddress":null,"cumulativeGasUsed":"0x10cea","from":"0xdf21fa922215b1a56f5a6d6294e6e36c85a0acfb","gasUsed":"0xbae2","logs":[{"address":"0xbb9bc244d798123fde783fcc1c72d3bb8c189413","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000df21fa922215b1a56f5a6d6294e6e36c85a0acfb","0x00000000000000000000000032be343b94f860124dc4fee278fdcbd38c102d88"],"data":"0x0000000000000000000000000000000000000000000000027cfefc4f3f392700","blockNumber":"0x1e773b","transactionIndex":"0x1","transactionHash":"0x0b4cc7844537023b709953390e3881ec5b233703a8e8824dc03e13729a1bd95a","blockHash":"0xad20a0f78d19d7857067a9c06e6411efeab7673e183e4a545f53b724bb7fabf0","logIndex":"0x0"}],"logsBloom":"0x00000000000000020000000000020000000000000000000000000000000000000000000000000000000000000000000000040000000000000100000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000010000000000000000000000000000000000000000000000010000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000002002000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000","to":"0xbb9bc244d798123fde783fcc1c72d3bb8c189413","transactionHash":"0x0b4cc7844537023b709953390e3881ec5b233703a8e8824dc03e13729a1bd95a","transactionIndex":"0x1"}`,
		wantError: errMissingReceiptPostState,
	},
	"missing fields": {
		input:     `{"blockHash":"0xad20a0f78d19d7857067a9c06e6411efeab7673e183e4a545f53b724bb7fabf0","blockNumber":"0x1e773b","contractAddress":null,"cumulativeGasUsed":"0x10cea","from":"0xdf21fa922215b1a56f5a6d6294e6e36c85a0acfb","gasUsed":"0xbae2","logs":[{"address":"0xbb9bc244d798123fde783fcc1c72d3bb8c189413","topics":["0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef","0x000000000000000000000000df21fa922215b1a56f5a6d6294e6e36c85a0acfb","0x00000000000000000000000032be343b94f860124dc4fee278fdcbd38c102d88"],"data":"0x0000000000000000000000000000000000000000000000027cfefc4f3f392700","blockNumber":"0x1e773b","transactionIndex":"0x1","transactionHash":"0x0b4cc7844537023b709953390e3881ec5b233703a8e8824dc03e13729a1bd95a","blockHash":"0xad20a0f78d19d7857067a9c06e6411efeab7673e183e4a545f53b724bb7fabf0","logIndex":"0x0"}],"logsBloom":"0x00000000000000020000000000020000000000000000000000000000000000000000000000000000000000000000000000040000000000000100000000000000000000000000000000000008000000000000000000000000000000000000000000000000000000000000000000000000000000000000000800000010000000000000000000000000000000000000000000000010000000000000000000001000000000000000000000000000000000000000000000000000000000000000000000002002000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000","root":"0x6e8a06b2dac39ac5c9d4db5fb2a2a94ef7a6e5ec1c554079112112caf162998a","to":"0xbb9bc244d798123fde783fcc1c72d3bb8c189413"}`,
		wantError: errMissingReceiptFields,
	},
}

func TestUnmarshalReceipt(t *testing.T) {
	for name, test := range unmarshalReceiptTests {
		var r *Receipt
		err := json.Unmarshal([]byte(test.input), &r)
		checkError(t, name, err, test.wantError)
	}
}

func TestMarshalReceipt(t *testing.T) {
	for name, test := range unmarshalReceiptTests {
		if test.wantError != nil {
			continue
		}
		var original *Receipt
		json.Unmarshal([]byte(test.input), &original)

		blob, err := json.Marshal(original)
		if err != nil {
			t.Errorf("test %q: failed to marshal receipt: %v", name, err)
			continue
		}
		var proced *Receipt
		if err := json.Unmarshal(blob, &proced); err != nil {
			t.Errorf("Test %q: failed to unmarshal marhsalled receipt: %v", name, err)
			continue
		}
		if !reflect.DeepEqual(original, proced) {
			t.Errorf("test %q: receipt mismatch: have %+v, want %+v", name, proced, original)
			continue
		}
	}
}

func checkError(t *testing.T, testname string, got, want error) bool {
	if got == nil {
		if want != nil {
			t.Errorf("test %q: got no error, want %q", testname, want)
			return false
		}
		return true
	}
	if want == nil {
		t.Errorf("test %q: unexpected error %q", testname, got)
	} else if got.Error() != want.Error() {
		t.Errorf("test %q: got error %q, want %q", testname, got, want)
	}
	return false
}
