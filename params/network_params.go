// Copyright 2017 The go-ethereum Authors
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

package params

// These are network parameters that need to be constant between clients, but
// aren't necessarily consensus related.

const (
	// BloomBitsBlocks is the number of blocks a single bloom bit section vector
	// contains on the server side.
	BloomBitsBlocks uint64 = 4096

	// BloomBitsBlocksClient is the number of blocks a single bloom bit section vector
	// contains on the light client side
	BloomBitsBlocksClient uint64 = 32768

	// BloomConfirms is the number of confirmation blocks before a bloom section is
	// considered probably final and its rotated bits are calculated.
	BloomConfirms = 256

	// CHTFrequency is the block frequency for creating CHTs
	CHTFrequency = 32768

	// BloomTrieFrequency is the block frequency for creating BloomTrie on both
	// server/client sides.
	BloomTrieFrequency = 32768

	// HelperTrieConfirmations is the number of confirmations before a client is expected
	// to have the given HelperTrie available.
	HelperTrieConfirmations = 2048

	// HelperTrieProcessConfirmations is the number of confirmations before a HelperTrie
	// is generated
	HelperTrieProcessConfirmations = 256
)
