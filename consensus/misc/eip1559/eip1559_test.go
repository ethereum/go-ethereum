// Copyright 2021 The go-ethereum Authors
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

package eip1559

import (
	"math/big"
	"testing"

	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/params"
)

func testConfigEip1559() *params.ChainConfig {
	config := *params.TestChainConfig
	config.Eip1559Block = big.NewInt(1)
	return &config
}

func TestVerifyEip1559HeaderParentBaseFee(t *testing.T) {
	config := testConfigEip1559()

	for _, tc := range []struct {
		name      string
		parent    *types.Header
		headerNum int64
		wantOk    bool
	}{
		{
			name: "eip1559 parent missing basefee",
			parent: &types.Header{
				Number: big.NewInt(1),
			},
			headerNum: 2,
			wantOk:    false,
		},
		{
			name: "eip1559 parent with basefee",
			parent: &types.Header{
				Number:  big.NewInt(1),
				BaseFee: new(big.Int).Set(common.BaseFee),
			},
			headerNum: 2,
			wantOk:    true,
		},
		{
			name: "pre-eip1559 parent ignores basefee",
			parent: &types.Header{
				Number: big.NewInt(0),
			},
			headerNum: 1,
			wantOk:    true,
		},
	} {
		header := &types.Header{
			Number:  big.NewInt(tc.headerNum),
			BaseFee: new(big.Int).Set(common.BaseFee),
		}
		err := VerifyEip1559Header(config, tc.parent, header)
		if tc.wantOk && err != nil {
			t.Fatalf("%s: expected no error, got %v", tc.name, err)
		}
		if !tc.wantOk && err == nil {
			t.Fatalf("%s: expected error, got nil", tc.name)
		}
	}
}
