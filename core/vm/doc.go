// Copyright 2014 The go-ethereum Authors
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

/*
Package vm implements the Ethereum Virtual Machine.

The vm package implements two EVMs, a byte code VM and a JIT VM. The BC
(Byte Code) VM loops over a set of bytes and executes them according to the set
of rules defined in the Ethereum yellow paper. When the BC VM is invoked it
invokes the JIT VM in a seperate goroutine and compiles the byte code in JIT
instructions.

The JIT VM, when invoked, loops around a set of pre-defined instructions until
it either runs of gas, causes an internal error, returns or stops. At a later
stage the JIT VM will see some additional features that will cause sets of
instructions to be compiled down to segments. Segments are sets of instructions
that can be run in one go saving precious time during execution.
*/
package vm
