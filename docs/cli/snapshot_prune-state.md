# Prune state

The ```bor snapshot prune-state``` command will prune historical state data
with the help of the state snapshot. All trie nodes and contract codes that do not belong to the
specified	version state will be deleted from the database. After pruning, only two version states
are available: genesis and the specific one.

## Options

- ```bloomfilter.size```: Size of the bloom filter (default: 2048)

- ```datadir```: Path of the data directory to store information

- ```datadir.ancient```: Path of the ancient data directory to store information

- ```keystore```: Path of the data directory to store keys

### Cache Options

- ```cache```: Megabytes of memory allocated to internal caching (default: 1024)

- ```cache.trie```: Percentage of cache memory allowance to use for trie caching (default: 25)

- ```cache.trie.journal```: Path of the trie journal directory to store information (default: triecache)