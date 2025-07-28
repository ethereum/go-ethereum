// Copyright 2025 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common/hexutil"
)

// InclusionList is a list of transactions that should be included in a payload.
type InclusionList [][]byte

func (inclusionList InclusionList) MarshalJSON() ([]byte, error) {
	inclusionListHex := make([]hexutil.Bytes, len(inclusionList))
	for i, tx := range inclusionList {
		inclusionListHex[i] = tx
	}
	return json.Marshal(inclusionListHex)
}

func (inclusionList *InclusionList) UnmarshalJSON(input []byte) error {
	var inclusionListHex []hexutil.Bytes
	if err := json.Unmarshal(input, &inclusionListHex); err != nil {
		return err
	}

	*inclusionList = make([][]byte, len(inclusionListHex))
	for i, tx := range inclusionListHex {
		(*inclusionList)[i] = tx
	}

	return nil
}
