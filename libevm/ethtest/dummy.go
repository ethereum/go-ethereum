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

package ethtest

import (
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/consensus"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/types"
)

// DummyChainContext returns a dummy that returns [DummyEngine] when its
// Engine() method is called, and panics when its GetHeader() method is called.
func DummyChainContext() core.ChainContext {
	return chainContext{}
}

// DummyEngine returns a dummy that panics when its Author() method is called.
func DummyEngine() consensus.Engine {
	return engine{}
}

type (
	chainContext struct{}
	engine       struct{ consensus.Engine }
)

func (chainContext) Engine() consensus.Engine                    { return engine{} }
func (chainContext) GetHeader(common.Hash, uint64) *types.Header { panic("unimplemented") }
func (engine) Author(h *types.Header) (common.Address, error)    { panic("unimplemented") }
