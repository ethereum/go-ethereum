.. _transaction_tests:

Transaction Tests
=================

Describes a complete transaction and its `RLP <https://github.com/ethereum/wiki/wiki/RLP>`_ representation using the .json file.

=================== ==============================================================
Location            `/TransactionTests <https://github.com/ethereum/tests/tree/develop/TransactionTests>`_
Supported Hardforks ``Constantinople`` | ``EIP158`` | ``Frontier`` | ``Homestead``
Status              Actively supported
=================== ==============================================================

Test Implementation
-------------------

The client should read the rlp and check whether the transaction is valid, has the correct sender and corresponds to the transaction parameters.
If it is an invalid transaction, the transaction and the sender object will be missing.

Test Structure
--------------
::

	{
	   "transactionTest1": {
		  "rlp" : "bytearray",
		  "sender" : "address",
		  "blocknumber" : "1000000"
		  "transaction" : {
				"nonce" : "int",
				"gasPrice" : "int",
				"gasLimit" : "int",
				"to" : "address",
				"value" : "int",
				"v" : "byte",
				"r" : "256 bit unsigned int",
				"s" : "256 bit unsigned int",
				"data" : "byte array"
		  }
	   },

	   "invalidTransactionTest": {
		  "rlp" : "bytearray",
	   },
	   ...
	}

Sections
^^^^^^^^

* ``rlp`` - RLP encoded data of this transaction
* ``transaction`` - transaction described by fields
* ``nonce`` - A scalar value equal to the number of transactions sent by the sender.
* ``gasPrice`` - A scalar value equal to the number of wei to be paid per unit of gas.
* ``gasLimit`` - A scalar value equal to the maximum amount of gas that should be used in executing this transaction.
* ``to`` - The 160-bit address of the message call's recipient or empty for a contract creation transaction.
* ``value`` - A scalar value equal to the number of wei to be transferred to the message call's recipient or, in the case of contract creation, as an endowment to the newly created account.
* ``v, r, s`` - Values corresponding to the signature of the transaction and used to determine the sender of the transaction.
* ``sender`` - the address of the sender, derived from the v,r,s values.
* ``blocknumber`` - indicates network rules for the transaction. Since blocknumber = **1000000** Homestead rules are applied to transaction. (see https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.mediawiki)
