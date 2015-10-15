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

package common

import "testing"

/* More test vectors:
https://github.com/ethereum/web3.js/blob/master/test/iban.fromAddress.js
https://github.com/ethereum/web3.js/blob/master/test/iban.toAddress.js
https://github.com/ethereum/web3.js/blob/master/test/iban.isValid.js
https://github.com/ethereum/libethereum/blob/develop/test/libethcore/icap.cpp
*/

type icapTest struct {
	name string
	addr string
	icap string
}

var icapOKTests = []icapTest{
	{"Direct1", "0x52dc504a422f0e2a9e7632a34a50f1a82f8224c7", "XE499OG1EH8ZZI0KXC6N83EKGT1BM97P2O7"},
	{"Direct2", "0x11c5496aee77c1ba1f0854206a26dda82a81d6d8", "XE1222Q908LN1QBBU6XUQSO1OHWJIOS46OO"},
	{"DirectZeroPrefix", "0x00c5496aee77c1ba1f0854206a26dda82a81d6d8", "XE7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS"},
	{"DirectDoubleZeroPrefix", "0x0000a5327eab78357cbf2ae8f3d49fd9d90c7d22", "XE0600DQK33XDTYUCRI0KYM5ELAKXDWWF6"},
}

var icapInvalidTests = []icapTest{
	{"DirectInvalidCheckSum", "", "XE7438O073KYGTWWZN0F2WZ0R8PX5ZPPZS"},
	{"DirectInvalidCountryCode", "", "XD7338O073KYGTWWZN0F2WZ0R8PX5ZPPZS"},
	{"DirectInvalidLength36", "", "XE499OG1EH8ZZI0KXC6N83EKGT1BM97P2O77"},
	{"DirectInvalidLength33", "", "XE499OG1EH8ZZI0KXC6N83EKGT1BM97P2"},

	{"IndirectInvalidCheckSum", "", "XE35ETHXREGGOPHERSSS"},
	{"IndirectInvalidAssetIdentifier", "", "XE34ETHXREGGOPHERSSS"},
	{"IndirectInvalidLength19", "", "XE34ETHXREGGOPHERSS"},
	{"IndirectInvalidLength21", "", "XE34ETHXREGGOPHERSSSS"},
}

func TestICAPOK(t *testing.T) {
	for _, test := range icapOKTests {
		decodeEncodeTest(HexToAddress(test.addr), test.icap, t)
	}
}

func TestICAPInvalid(t *testing.T) {
	for _, test := range icapInvalidTests {
		failedDecodingTest(test.icap, t)
	}
}

func decodeEncodeTest(addr0 Address, icap0 string, t *testing.T) {
	icap1, err := AddressToICAP(addr0)
	if err != nil {
		t.Errorf("ICAP encoding failed: %s", err)
	}
	if icap1 != icap0 {
		t.Errorf("ICAP mismatch: have: %s want: %s", icap1, icap0)
	}

	addr1, err := ICAPToAddress(icap0)
	if err != nil {
		t.Errorf("ICAP decoding failed: %s", err)
	}
	if addr1 != addr0 {
		t.Errorf("Address mismatch: have: %x want: %x", addr1, addr0)
	}
}

func failedDecodingTest(icap string, t *testing.T) {
	addr, err := ICAPToAddress(icap)
	if err == nil {
		t.Errorf("Expected ICAP decoding to fail.")
	}
	if addr != (Address{}) {
		t.Errorf("Expected empty Address on failed ICAP decoding.")
	}
}
