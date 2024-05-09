# Inspect ancient DB for block pruning

The ```bor snapshot inspect-ancient-db``` command will inspect few fields in the ancient datastore using the given datadir location.


This command prints the following information which is useful for block-pruning rounds:

1. Offset / Start block number (from kvDB).
2. Amount of items in the ancientdb.
3. Last block number written in ancientdb.


## Options

- ```datadir```: Path of the data directory to store information

- ```datadir.ancient```: Path of the old ancient data directory

- ```keystore```: Path of the data directory to store keys