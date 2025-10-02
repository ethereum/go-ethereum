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

// Package temporary provides thread-safe, temporary registration of all libevm
// hooks and payloads.
package temporary

import (
	"sync"

	"github.com/ava-labs/libevm/core/state"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/params"
)

var mu sync.Mutex

// WithRegisteredExtras takes a global lock and temporarily registers [params],
// [state], [types], and [vm] extras before calling the provided function. It
// can be thought of as an atomic call to all functions equivalent to
// [params.WithTempRegisteredExtras].
//
// This is the *only* safe way to override libevm functionality. Direct calls to
// the package-specific temporary registration functions are not advised.
//
// WithRegisteredExtras MUST NOT be used on a live chain. It is solely intended
// for off-chain consumers that require access to extras.
func WithRegisteredExtras[
	C params.ChainConfigHooks, R params.RulesHooks,
	H, B, SA any,
	HPtr types.HeaderHooksPointer[H],
	BPtr types.BlockBodyHooksPointer[B, BPtr],
](
	paramsExtras params.Extras[C, R],
	sdbHooks state.StateDBHooks,
	vmHooks vm.Hooks,
	fn func(params.ExtraPayloads[C, R], types.ExtraPayloads[HPtr, BPtr, SA]),
) {
	mu.Lock()
	defer mu.Unlock()

	params.WithTempRegisteredExtras(paramsExtras, func(paramsPayloads params.ExtraPayloads[C, R]) {
		types.WithTempRegisteredExtras(func(typesPayloads types.ExtraPayloads[HPtr, BPtr, SA]) {
			state.WithTempRegisteredExtras(sdbHooks, func() {
				vm.WithTempRegisteredHooks(vmHooks, func() {
					fn(paramsPayloads, typesPayloads)
				})
			})
		})
	})
}
