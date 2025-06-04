/**
 * Mnemonist Library Endpoint
 * ===========================
 *
 * Exporting every data structure through a unified endpoint. Consumers
 * of this library should prefer the modular access though.
 */
var Heap = require('./heap.js'),
    FibonacciHeap = require('./fibonacci-heap.js'),
    SuffixArray = require('./suffix-array.js');

module.exports = {
  BiMap: require('./bi-map.js'),
  BitSet: require('./bit-set.js'),
  BitVector: require('./bit-vector.js'),
  BloomFilter: require('./bloom-filter.js'),
  BKTree: require('./bk-tree.js'),
  CircularBuffer: require('./circular-buffer.js'),
  DefaultMap: require('./default-map.js'),
  DefaultWeakMap: require('./default-weak-map.js'),
  FixedDeque: require('./fixed-deque.js'),
  StaticDisjointSet: require('./static-disjoint-set.js'),
  FibonacciHeap: FibonacciHeap,
  MinFibonacciHeap: FibonacciHeap.MinFibonacciHeap,
  MaxFibonacciHeap: FibonacciHeap.MaxFibonacciHeap,
  FixedReverseHeap: require('./fixed-reverse-heap.js'),
  FuzzyMap: require('./fuzzy-map.js'),
  FuzzyMultiMap: require('./fuzzy-multi-map.js'),
  HashedArrayTree: require('./hashed-array-tree.js'),
  Heap: Heap,
  MinHeap: Heap.MinHeap,
  MaxHeap: Heap.MaxHeap,
  StaticIntervalTree: require('./static-interval-tree.js'),
  InvertedIndex: require('./inverted-index.js'),
  KDTree: require('./kd-tree.js'),
  LinkedList: require('./linked-list.js'),
  LRUCache: require('./lru-cache.js'),
  LRUMap: require('./lru-map.js'),
  MultiMap: require('./multi-map.js'),
  MultiSet: require('./multi-set.js'),
  PassjoinIndex: require('./passjoin-index.js'),
  Queue: require('./queue.js'),
  FixedStack: require('./fixed-stack.js'),
  Stack: require('./stack.js'),
  SuffixArray: SuffixArray,
  GeneralizedSuffixArray: SuffixArray.GeneralizedSuffixArray,
  Set: require('./set.js'),
  SparseQueueSet: require('./sparse-queue-set.js'),
  SparseMap: require('./sparse-map.js'),
  SparseSet: require('./sparse-set.js'),
  SymSpell: require('./symspell.js'),
  Trie: require('./trie.js'),
  TrieMap: require('./trie-map.js'),
  Vector: require('./vector.js'),
  VPTree: require('./vp-tree.js')
};
