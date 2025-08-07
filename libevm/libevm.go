// Copyright 2024 the libevm authors.
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

package libevm

import (
	"github.com/holiman/uint256"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/libevm/stateconf"
)

// PrecompiledContract is an exact copy of vm.PrecompiledContract, mirrored here
// for instances where importing that package would result in a circular
// dependency.
type PrecompiledContract interface {
	RequiredGas(input []byte) uint64
	Run(input []byte) ([]byte, error)
}

// StateReader is a subset of vm.StateDB, exposing only methods that read from
// but do not modify state. See method comments in vm.StateDB, which aren't
// copied here as they risk becoming outdated.
type StateReader interface {
	GetBalance(common.Address) *uint256.Int
	GetNonce(common.Address) uint64

	GetCodeHash(common.Address) common.Hash
	GetCode(common.Address) []byte
	GetCodeSize(common.Address) int

	GetRefund() uint64

	GetCommittedState(common.Address, common.Hash, ...stateconf.StateDBStateOption) common.Hash
	GetState(common.Address, common.Hash, ...stateconf.StateDBStateOption) common.Hash

	GetTransientState(addr common.Address, key common.Hash) common.Hash

	HasSelfDestructed(common.Address) bool

	Exist(common.Address) bool
	Empty(common.Address) bool

	AddressInAccessList(addr common.Address) bool
	SlotInAccessList(addr common.Address, slot common.Hash) (addressOk bool, slotOk bool)
}

// AddressContext carries addresses available to contexts such as calls and
// contract creation.
//
// With respect to contract creation, the EVMSemantic.Self address MAY be the
// predicted address of the contract about to be deployed, which might not exist
// yet.
type AddressContext struct {
	Origin common.Address // equivalent to vm.ORIGIN op code
	// EVMSemantic addresses are those defined by the rules of the EVM, based on
	// the type of call made to a contract; i.e. the addresses pushed to the
	// stack by the vm.CALLER and vm.SELF op codes, respectively.
	EVMSemantic CallerAndSelf
	// Raw addresses are those that would be available to a contract under a
	// standard CALL; i.e. not interpreted according EVM rules. They are the
	// "intuitive" addresses such that the `Caller` is the account that called
	// `Self` even if it did so via DELEGATECALL or CALLCODE (in which cases
	// `Raw` and `EVMSemantic` would differ).
	//
	// Raw MUST NOT be nil when returned to a precompile implementation but MAY
	// be nil in other situations (e.g. hooks), which MUST document behaviour on
	// a case-by-case basis.
	Raw *CallerAndSelf
}

// CallerAndSelf carries said addresses for use in an [AddressContext], where
// the definitions of `Caller` and `Self` are defined based on context.
type CallerAndSelf struct {
	Caller common.Address
	Self   common.Address
}
