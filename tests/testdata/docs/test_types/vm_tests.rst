.. _vm_tests:

VM Tests
========

The VM tests aim is to test the basic workings of the VM in
isolation.

=================== ==============================================================
Location            `/VMTests <https://github.com/ethereum/tests/tree/develop/VMTests>`_
Supported Hardforks Currently only Homestead
Status              Actively supported
=================== ==============================================================

This is specifically not meant to cover transaction, creation or call 
processing, or management of the state trie. Indeed at least one implementation 
tests the VM without calling into any Trie code at all.

A VM test is based around the notion of executing a single piece of code as part of a transaction, 
described by the ``exec`` portion of the test. The overarching environment in which it is 
executed is described by the ``env`` portion of the test and includes attributes 
of the current and previous blocks. A set of pre-existing accounts are detailed 
in the ``pre`` portion and form the world state prior to execution. Similarly, a set 
of accounts are detailed in the ``post`` portion to specify the end world state.

The gas remaining (``gas``), the log entries (``logs``) as well as any output returned 
from the code (``output``) is also detailed.


Test Implementation
-------------------

It is generally expected that the test implementer will read ``env``, ``exec`` and ``pre`` 
then check their results against ``gas``, ``logs``, ``out``, ``post`` and ``callcreates``. 
If an exception is expected, then latter sections are absent in the test. Since the 
reverting of the state is not part of the VM tests.

Because the data of the blockchain is not given, the opcode BLOCKHASH could not 
return the hashes of the corresponding blocks. Therefore we define the hash of 
block number n to be SHA3-256("n").

Since these tests are meant only as a basic test of VM operation, the ``CALL`` and 
``CREATE`` instructions are not actually executed. To provide the possibility of 
testing to guarantee they were actually run at all, a separate portion ``callcreates`` 
details each ``CALL`` or ``CREATE`` operation in the order they would have been executed. 
Furthermore, gas required is simply that of the VM execution: the gas cost for 
transaction processing is excluded.

Test Structure
--------------

::

	{
	   "test name 1": {
		   "env": { ... },
		   "pre": { ... },
		   "exec": { ... },
		   "gas": { ... },
		   "logs": { ... },
		   "out": { ... },
		   "post": { ... },
		   "callcreates": { ... }
	   },
	   "test name 2": {
		   "env": { ... },
		   "pre": { ... },
		   "exec": { ... },
		   "gas": { ... },
		   "logs": { ... },
		   "out": { ... },
		   "post": { ... },
		   "callcreates": { ... }
	   },
	   ...
	}

The env Section
^^^^^^^^^^^^^^^

* ``currentCoinbase``: The current block's coinbase address, to be returned by the ``COINBASE`` instruction.
* ``currentDifficulty``: The current block's difficulty, to be returned by the ``DIFFICULTY`` instruction.
* ``currentGasLimit``: The current block's gas limit.
* ``currentNumber``: The current block's number.
* ``currentTimestamp``: The current block's timestamp.
* ``previousHash``: The previous block's hash.

The exec Section
^^^^^^^^^^^^^^^^

* ``address``: The address of the account under which the code is executing, to be returned by the ``ADDRESS`` instruction.
* ``origin``: The address of the execution's origin, to be returned by the ``ORIGIN`` instruction.
* ``caller``: The address of the execution's caller, to be returned by the ``CALLER`` instruction.
* ``value``: The value of the call (or the endowment of the create), to be returned by the ``CALLVALUE`` instruction.
* ``data``: The input data passed to the execution, as used by the ``CALLDATA``... instructions. Given as an array of byte values. See $DATA_ARRAY.
* ``code``: The actual code that should be executed on the VM (not the one stored in the state(address)) . See $DATA_ARRAY.
* ``gasPrice``: The price of gas for the transaction, as used by the ``GASPRICE`` instruction.
* ``gas``: The total amount of gas available for the execution, as would be returned by the ``GAS`` instruction were it be executed first.

The pre and post Section
^^^^^^^^^^^^^^^^^^^^^^^^

The ``pre`` and ``post`` sections each have the same format of a mapping between addresses and accounts. Each account has the format:

* ``balance``: The balance of the account.
* ``nonce``: The nonce of the account.
* ``code``: The body code of the account, given as an array of byte values. See $DATA_ARRAY.
* ``storage``: The account's storage, given as a mapping of keys to values. For key used notion of string as digital or hex number e.g: ``"1200"`` or ``"0x04B0"`` For values used $DATA_ARRAY.

The callcreates Section
^^^^^^^^^^^^^^^^^^^^^^^

The ``callcreates`` section details each ``CALL`` or ``CREATE`` instruction that has been executed. It is an array of maps with keys:

* ``data``: An array of bytes specifying the data with which the ``CALL`` or ``CREATE`` operation was made. In the case of ``CREATE``, this would be the (initialisation) code. See $DATA_ARRAY.
* ``destination``: The receipt address to which the ``CALL`` was made, or the null address (``"0000..."``) if the corresponding operation was ``CREATE``.
* ``gasLimit``: The amount of gas with which the operation was made.
* ``value``: The value or endowment with which the operation was made.

The logs Section
^^^^^^^^^^^^^^^^

The ``logs`` sections is a mapping between the blooms and their corresponding logentries.
Each logentry has the format:

* ``address``: The address of the logentry.
* ``data``: The data of the logentry.
* ``topics``: The topics of the logentry, given as an array of values.  

The gas and output Keys
^^^^^^^^^^^^^^^^^^^^^^^

Finally, there are two simple keys, ``gas`` and ``output``:

* ``gas``: The amount of gas remaining after execution.
* ``output``: The data, given as an array of bytes, returned from the execution (using the ``RETURN`` instruction). See $DATA_ARRAY.

 **$DATA_ARRAY** - type that intended to contain raw byte data   
  and for convenient of the users is populated with three   
  types of numbers, all of them should be converted and   
  concatenated to a byte array for VM execution.   

* The types are:    
  1. number - (unsigned 64bit)
  2. "longnumber" - (any long number)
  3. "0xhex_num"  - (hex format number)


   e.g: ``````[1, 2, 10000, "0xabc345dFF", "199999999999999999999999999999999999999"]``````			 