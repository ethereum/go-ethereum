// Copyright 2026 The go-ethereum Authors
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

//go:build zkvm

package overlay

// A zkVM guest executes a block statelessly from a witness, which carries no
// transition state, so the gob decoder (whose package initialisation alone
// runs millions of instructions) is left out: finding a transition state in
// the database is a bug, not an input.
func decodeTransitionState(data []byte) (*TransitionState, error) {
	panic("overlay: a stored transition state cannot be decoded in a zkVM guest")
}
