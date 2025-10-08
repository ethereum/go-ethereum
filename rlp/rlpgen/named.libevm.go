// Copyright 2025 the libevm authors.
//
// The libevm additions to go-ethereum are free software: you can redistribute
// them and/or modify them under the terms of the GNU Lesser General Public License
// as published by the Free Software Foundation, either version 3 of the License,
// or (at your option) any later version.
//
// The libevm additions are distributed in the hope that they will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the GNU Lesser
// General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see
// <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"go/types"
)

// makeNamedBasicOp is a convenience wrapper for basicOp.
// It returns a basicOp with the named type as the main type instead of the underlying basic type.
func (bctx *buildContext) makeNamedBasicOp(named *types.Named) (op, error) {
	underlying := named.Underlying()
	basic, ok := underlying.(*types.Basic)
	if !ok {
		return nil, fmt.Errorf("expected basic type, got %T", underlying)
	}

	// We use basic op because it actually supports necessary conversions (through writeNeedsConversion and decodeNeedsConversion)
	// for named types.
	// The only problem with that is it does not support the named type as the main type.
	// So we use the named type as the main type instead of the underlying basic type.
	baseOp, err := bctx.makeBasicOp(basic)
	if err != nil {
		return nil, err
	}

	op, ok := baseOp.(basicOp)
	if !ok {
		return nil, fmt.Errorf("expected basicOp, got %T", baseOp)
	}
	op.typ = named

	return op, nil
}

// hasBasicUnderlying checks whether `named` has an underlying basic type.
func hasBasicUnderlying(named *types.Named) bool {
	_, ok := named.Underlying().(*types.Basic)
	return ok
}
