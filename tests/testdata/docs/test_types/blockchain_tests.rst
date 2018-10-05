.. _blockchain_tests:

Blockchain Tests
================

The blockchain tests aim is to test the basic verification of a blockchain.

=================== ==============================================================
Location            `/BlockchainTests <https://github.com/ethereum/tests/tree/develop/BlockchainTests>`_
Supported Hardforks ``Byzantium`` | ``Constantinople`` | ``EIP150`` | ``EIP158`` | ``Frontier`` | ``Homestead``
Status              Actively supported
=================== ==============================================================

A blockchain test is based around the notion of executing a list of single blocks,
described by the ``blocks`` portion of the test. The first block is the modified
genesis block as described by the ``genesisBlockHeader`` portion of the test. 
A set of pre-existing accounts are detailed in the ``pre`` portion and form the 
world state of the genesis block.

Of special notice is the 
`/BlockchainTests/GeneralStateTests <https://github.com/ethereum/tests/tree/develop/BlockchainTests/GeneralStateTests>`_
folder within the blockchain tests folder structure, which contains a copy of the
:ref:`state_tests` but executes them within the logic of the blockchain tests.


Test Implementation
-------------------

It is generally expected that the test implementer will read ``genesisBlockHeader`` 
and ``pre`` and build the corresponding blockchain in the client. Then the new blocks, 
described by its RLP found in the ``rlp`` object of the ``blocks`` (RLP of a complete block, 
not the block header only), is read. If the client concludes that the block is valid, 
it should execute the block and verify the parameters given in ``blockHeader`` 
(block header of the new block), ``transactions`` (transaction list) and ``uncleHeaders`` 
(list of uncle headers). If the client concludes that the block is invalid, it should verify 
that no ``blockHeader``, ``transactions`` or ``uncleHeaders`` object is present in the test. 
The client is expected to iterate through the list of blocks and ignore invalid blocks.

Test Structure
--------------

::

  {
     "TESTNAME_Byzantium": {
       "blocks" : [
         {
           "blockHeader": { ... },
           "rlp": { ... },
           "transactions": { ... },
           "uncleHeaders": { ... }
         },
         {
           "blockHeader": { ... },
           "rlp": { ... },
           "transactions": { ... },
           "uncleHeaders": { ... }
         },
         { ... }
       ],
       "genesisBlockHeader": { ... },
       "genesisRLP": " ... ",
       "lastblockhash": " ... ",
       "network": "Byzantium",
       "postState": { ... },
       "pre": { ... }       
     },
     "TESTNAME_EIP150": {
       "blocks" : [
         {
           "blockHeader": { ... },
           "rlp": { ... },
           "transactions": { ... },
           "uncleHeaders": { ... }
         },
         {
           "blockHeader": { ... },
           "rlp": { ... },
           "transactions": { ... },
           "uncleHeaders": { ... }
         },
         { ... }
       ],
       "genesisBlockHeader": { ... },
       "genesisRLP": " ... ",
       "lastblockhash": " ... ",
       "network": "Byzantium",
       "postState": { ... },
       "pre": { ... }       
     },
     ...
  }


The Blocks Section
^^^^^^^^^^^^^^^^^^

The ``blocks`` section is a list of block objects, which have the following format:

* ``rlp`` section contains the complete rlp of the new block as described in the 
  yellow paper in section 4.3.3.

* ``blockHeader`` section  describes the block header of the new block in the same 
  format as described in `genesisBlockHeader`.

* ``transactions`` section is a list of transactions which have the same format as 
  in :ref:`transaction_tests`.

* ``uncleHeaders`` section is a list of block headers which have the same format as 
  descibed in `genesisBlockHeader`.


The genesisBlockHeader Section
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

``coinbase``:
  The 160-bit address to which all fees collected from the successful mining of this block be
  transferred, as returned by the **COINBASE** instruction.
``difficulty``: 
  A scalar value corresponding to the difficulty level of this block. This can be 
  calculated from the previous block’s difficulty level and the timestamp, as returned 
  by the **DIFFICULTY** instruction.
``gasLimit``: 
  A scalar value equal to the current limit of gas expenditure per block, as returned 
  by the **GASLIMIT** instruction.
``number``:
  A scalar value equal to the number of ancestor blocks. The genesis block has a number of zero.
``timestamp``: 
  A scalar value equal to the reasonable output of Unix’s time() at this block’s inception,
  as returned by the **TIMESTAMP** instruction.
``parentHash``: 
  The Keccak 256-bit hash of the parent block’s header, in its entirety
``bloom``:
  The Bloom filter composed from indexable information (logger address and log topics)
  contained in each log entry from the receipt of each transaction in the transactions list.
``extraData``:
  An arbitrary byte array containing data relevant to this block. This must be 1024 bytes or fewer.
``gasUsed``:
  A scalar value equal to the total gas used in transactions in this block.
``nonce``:
  A 256-bit hash which proves that a sufficient amount of computation has been 
  carried out on this block.
``receiptTrie``: 
  The Keccak 256-bit hash of the root node of the trie structure populated with 
  the receipts of each transaction in the transactions list portion of the block.
``stateRoot``: 
  The Keccak 256-bit hash of the root node of the state trie, after all transactions 
  are executed and finalisations applied.
``transactionsTrie``: 
  The Keccak 256-bit hash of the root node of the trie structure populated with 
  each transaction in the transactions list portion of the block.
``uncleHash``: 
  The Keccak 256-bit hash of the uncles list portion of this block


Pre and postState Sections
^^^^^^^^^^^^^^^^^^^^^^^^^^

* ``pre`` section: as described in :ref:`state_tests`.

* ``postState`` section: as described in :ref:`state_tests` (section - post).


Optional BlockHeader Information
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

``"blocknumber" = "int"`` is section which defines what is the order of this block. 
It is used to define a situation when you have 3 blocks already imported but then it comes new version of the block 2 and 3 and thus you might have new best blockchain with blocks 1 2' 3' instead previous. If `blocknumber` is undefined then it is assumed that blocks are imported one by one. When running test, this field could be used for information purpose only.

``"chainname" = "string"`` This is used for defining forks in the same test. You could mine blocks to chain "A": 1, 2, 3 then to chain "B": 1, 2, 3, 4 (chainB becomes primary). Then again to chain "A": 4, 5, 6  (chainA becomes primary) and so on. chainname could also be defined in uncle header section. If defined in uncle header it tells on which chain's block uncle header would be populated from. When running test, this field could be used for information purpose only.

``"chainnetwork" = "string"`` Defines on which network rules this block was mined. (see the difference https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.mediawiki). When running test, this field could be used for information purpose only.