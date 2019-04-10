// Copyright 2019 The go-ethereum Authors
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

package trie

import "testing"

// This test case reproduces a **very** rare failure scenario which literally
// took months to catch. Don't even dream about removing it.
//
// The scenario is a multi-stage failure:
//   1. Create a storage trie with a depth of 3:
//      P(arent) -1-> N(node) -1-> C(hild)1
//                            -1-> C(hild)2
//
//   2. Flush everything to disk (i.e. cache is full, all 4 nodes are old):
//      [P] ---> [N] ---> [C1]
//                   ---> [C2]
//
//   3. Create (or change) a sibling of N, resulting in a new parent:
//      P2 -0-> [N] ---> [C1]
//                  ---> [C2]
//         -1-> S
//      In the failing code, this caused P2 to have a live reference to N, but
//      since N is on disk, no ref-counters are traked.
//
//   4. Delete the child C1, resulting in a changed N and changed parent:
//      P3 -1-> N2 ---> [C2]
//         -1-> S
//      Although P2, N and C1 are dereferenced, P2 is still in the cache as a recent
//      block still references it, and N and C2 are still on disk due to the same.
//
//   5. Recreate the child C1, which recreates N and P2 too.
//      P2 -0-> N  -1-> C1
//                 ---> [C2]
//             [N] ---> [C1]
//                 ---> [C2]
//         -1-> S
//      P2 was already in the cache, so N->C1 is ref counted, but P2->N is not!
//      lso, N and C1 are now both in the dirty cache as well as on disk.
//
//   6. Flush P2 to disk (i.e. cache is full), N remain is memory because it was just created:
//      [P2] ---> N  -1-> C1
//                   ---> [C2]
//               [N] ---> [C1]
//                   ---> [C2]
//           ---> [S]
//
//   7. Dereference P2, pruning it from disk along with S, N, C1 and C2:
//      ---> N  -1-> C1
//
//   8.
func TestPrunerFlushedResurrection(t *testing.T) {
}
