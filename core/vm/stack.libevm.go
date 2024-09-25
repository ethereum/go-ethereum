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

package vm

import "github.com/holiman/uint256"

// A MutableStack embeds a Stack to expose unexported mutation methods.
type MutableStack struct {
	*Stack
}

// Push pushes a value to the stack.
func (s MutableStack) Push(d *uint256.Int) { s.Stack.push(d) }

// Pop pops a value from the stack.
func (s MutableStack) Pop() uint256.Int { return s.Stack.pop() }
