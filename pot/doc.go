/*
POT: proximity order tree implements a container similar to a binary tree.
Value types implement the PoVal interface which provides the PO (proximity order)
comparison operator.
Each fork in the trie is itself a value. Values of the subtree contained under
a node all share the same order when compared to other elements in the tree.

Example of proximity order is the length of the common prefix over bitvectors.
(which is equivalent to the order of magnitude of the XOR distance over integers).

The package provides two implementations of PoVal,

* BoolAddress: an arbitrary length boolean vector
* HashAddress: a bitvector based address derived from 256 bit hash (common.Hash)

If the address space if limited, equality is defined as the maximum proximity order.

Arbitrary value types can extend these base types or define their own PO method.

The container offers applicative (funcional) style methods on PO trees:
* adding/removing en element
* swap (value based add/remove)
* merging two PO trees (union)

as well as iterator accessors that respect proximity order

When synchronicity of membership if not 100% requirement (e.g. used as a database
of network connections), applicative structures have the advantage that nodes
are immutable therefore manipulation does not need locking allowing for parallel retrievals.
For the use case where the entire container is supposed to allow changes by parallel routines, instance methods with write locks and access methods with readlock are provided. The latter only locks while reading the root node.

Pot
* retrieval, insertion and deletion by key involves log(n) pointer lookups
* for any item retrieval  (defined as common prefix on the binary key)
* provide syncronous iterators respecting proximity ordering  wrt any item
* provide asyncronous iterator (for parallel execution of operations) over n items
* allows cheap iteration over ranges
* asymmetric parallellised merge (union)

Note:
* as is, union only makes sense for set representations since which of two values with equal keys survives is random
* intersection is not implemented
* simple get accessor is not implemented (but derivable from EachNeighbour)

Pinned value on the node implies no need to copy keys of the item type.

Note that
* the same set of values allows for a large number of alternative
POT representations.
* values on the top are accessed faster than lower ones and the steps needed to retrieve items has a logarithmic distribution.

As a consequence on can organise the tree so that items that need faster access are torwards the top.
In particular for any subset where popularity has a power distriution that is independent of
proximity order (content addressed storage of chunks), it is in principle possible to create a pot where the steps needed to access an item is inversely proportional to its popularity.
Such organisation is not implemented as yet.

TODO:
* swap
* get
* overwrite-style merge
* intersection
* access frequency based optimisations

*/
package pot
