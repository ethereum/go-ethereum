// Copyright 2024 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common/hexutil"
)

func TestRequest_JSONCodec(t *testing.T) {
	tests := []struct {
		name   string
		fields *Request
	}{
		{
			name: "Deposit",
			fields: NewRequest(&Deposit{
				PublicKey:             [48]byte(hexutil.MustDecode("0xab89324e578c5b0162f260340bf9505080dab873c5269d112f95a90b6743442b918c36c4bfeadeec840266d801966a31")),
				WithdrawalCredentials: common.HexToHash("0x01000000000000000000000005562e2e3725f641a140a098ec720f65595ae58d"),
				Amount:                32000000000,
				Signature:             [96]byte(hexutil.MustDecode("0xb1921ba4029eaa2e7d783ae4f0f3417b4a0b3ef5691eb4d3a717355479ec89cb28dfac62b94815fc4939492bd00460430dafc49006a0196574741141b0acca886009d85da019039977a7cdb6c995a5691bd87914f95783c36df9f0470e6b302e")),
				Index:                 0x77a4190000000000,
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw, err := json.Marshal(tt.fields)
			if err != nil {
				t.Errorf("Request.MarshalJSON() error = %v", err)
				return
			}

			req := new(Request)
			if err := json.Unmarshal(raw, req); err != nil {
				t.Errorf("Request.UnmarshalJSON() error = %v", err)
				return
			}

			if !reflect.DeepEqual(req, tt.fields) {
				t.Errorf("Request not deep equal: want: %v have: %v", tt.fields, req)
			}
		})
	}
}
