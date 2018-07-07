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

/*
Package pot (proximity order tree) implements a container similar to a binary tree.
The elements are generic Val interface types.

Each fork in the trie is itself a value. Values of the subtree contained under
a node all share the same order when compared to other elements in the tree.

Example of proximity order is the length of the common prefix over bitvectors.
(which is equivalent to the reverse rank of order of magnitude of the MSB first X
OR distance over finite set of integers).

Methods take a comparison operator (pof, proximity order function) to compare two
value types. The default pof assumes Val to be or project to a byte slice using
the reverse rank on the MSB first XOR logarithmic disctance.

If the address space if limited, equality is defined as the maximum proximity order.

The container offers applicative (funcional) style methods on PO trees:
* adding/removing en element
* swap (value based add/remove)
* merging two PO trees (union)

as well as iterator accessors that respect proximity order

When synchronicity of membership if not 100% requirement (e.g. used as a database
of network connections), applicative structures have the advantage that nodes
are immutable therefore manipulation does not need locking allowing for
concurrent retrievals.
For the use case where the entire container is supposed to allow changes by
concurrent routines,

Pot
* retrieval, insertion and deletion by key involves log(n) pointer lookups
* for any item retrieval  (defined as common prefix on the binary key)
* provide synchronous iterators respecting proximity ordering  wrt any item
* provide asynchronous iterator (for parallel execution of operations) over n items
* allows cheap iteration over ranges
* asymmetric concurrent merge (union)

Note:
* as is, union only makes sense for set representations since which of two values
with equal keys survives is random
* intersection is not implemented
* simple get accessor is not implemented (but derivable from EachNeighbour)

Pinned value on the node implies no need to copy keys of the item type.

Note that
* the same set of values allows for a large number of alternative
POT representations.
* values on the top are accessed faster than lower ones and the steps needed to
retrieve items has a logarithmic distribution.

As a consequence one can organise the tree so that items that need faster access
are torwards the top. In particular for any subset where popularity has a power
distriution that is independent of proximity order (content addressed storage of
chunks), it is in principle possible to create a pot where the steps needed to
access an item is inversely proportional to its popularity.
Such organisation is not implemented as yet.

TODO:
* overwrite-style merge
* intersection
* access frequency based optimisations

*/
package pot
