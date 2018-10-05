.. _state_tests:

General State Tests
===================

The state tests aim is to test the basic workings of the state in isolation.

=================== ==============================================================
Location            `/GeneralStateTests <https://github.com/ethereum/tests/tree/develop/GeneralStateTests>`_
Supported Hardforks ``Byzantium`` | ``Constantinople`` | ``EIP150`` | ``EIP158`` | ``Frontier`` | ``Homestead``
Status              Actively supported
=================== ==============================================================

A state test is based around the notion of executing a single transaction, described 
by the ``transaction`` portion of the test. The overarching environment 
in which it is executed is described by the ``env`` portion of the test and 
includes attributes of the current and previous blocks. A set of pre-existing accounts 
are detailed in the ``pre`` portion and form the world state prior to execution. 
Similarly, a set of accounts are detailed in the ``post`` portion to specify the 
end world state. Since the data of the blockchain is not given, the opcode ``BLOCKHASH`` 
could not return the hashes of the corresponding blocks. Therefore we define the hash of 
block number ``n`` to be  ``SHA256("n")``.

The log entries (``logs``) as well as any output returned from the code (``output``) is also detailed.

Test Implementation
-------------------

It is generally expected that the test implementer will read ``env``, ``transaction`` 
and ``pre`` then check their results against ``logs``, ``out``, and ``post``.

.. note::
   The structure of state tests was reworked lately, see the associated discussion
   `here <https://github.com/ethereum/EIPs/issues/176>`_.

Test Structure
--------------

::

  {
    "testname" : {
      "env" : {
        "currentCoinbase" : "address",
        "currentDifficulty" : "0x020000", //minimum difficulty for mining on blockchain   
        "currentGasLimit" : "u64",  //not larger then maxGasLimit = 0x7fffffffffffffff
        "currentNumber" : "0x01",   //Irrelevant to hardfork parameters!
        "currentTimestamp" : "1000", //for blockchain version
        "previousHash" : "h256"
      },
      "post" : {
        "EIP150" : [
          {
            "hash" : "3e6dacc1575c6a8c76422255eca03529bbf4c0dda75dfc110b22d6dc4152396f",
            "indexes" : { "data" : 0, "gas" : 0,  "value" : 0 } 
          },
          {
            "hash" : "99a450d8ce5b987a71346d8a0a1203711f770745c7ef326912e46761f14cd764",
            "indexes" : { "data" : 0, "gas" : 0,  "value" : 1 }
          },
          ...        
        ],
        "EIP158" : [
          {
            "hash" : "3e6dacc1575c6a8c76422255eca03529bbf4c0dda75dfc110b22d6dc4152396f",
            "indexes" : { "data" : 0,   "gas" : 0,  "value" : 0 }
          },
          {
            "hash" : "99a450d8ce5b987a71346d8a0a1203711f770745c7ef326912e46761f14cd764",
            "indexes" : { "data" : 0,   "gas" : 0,  "value" : 1  }
          },
          ...         
        ],
        "Frontier" : [
          ...
        ],
        "Homestead" : [
          ...
        ]
      },
      "pre" : {
          //same as for StateTests
      },
      "transaction" : {
        "data" : [ "" ],
        "gasLimit" : [ "285000",   "100000",  "6000" ],
        "gasPrice" : "0x01",
        "nonce" : "0x00",
        "secretKey" : "45a915e4d060149eb4365960e6a7a45f334393093061116b197e3240065ff2d8",
        "to" : "095e7baea6a6c7c4c2dfeb977efac326af552d87",
          "value" : [   "10",   "0" ]
        }
    }
  }


The env Section
^^^^^^^^^^^^^^^

| ``currentCoinbase``	
|	The current block's coinbase address, to be returned by the `COINBASE` instruction.
| ``currentDifficulty``
|	The current block's difficulty, to be returned by the `DIFFICULTY` instruction.
| ``currentGasLimit``	
|	The current block's gas limit.
| ``currentNumber``
|	The current block's number. Also indicates network rules for the transaction. Since blocknumber = **1000000** Homestead rules are applied to transaction. (see https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.mediawiki)
| ``currentTimestamp``
|	The current block's timestamp.
| ``previousHash``
|	The previous block's hash.
|

The transaction Section
^^^^^^^^^^^^^^^^^^^^^^^

| ``data`` 
|	The input data passed to the execution, as used by the `CALLDATA`... instructions. Given as an array of byte values. See $DATA_ARRAY.
| ``gasLimit`` 
|	The total amount of gas available for the execution, as would be returned by the `GAS` instruction were it be executed first.
| ``gasPrice`` 
|	The price of gas for the transaction, as used by the `GASPRICE` instruction.
| ``nonce``
|	Scalar value equal to the number of transactions sent by the sender.
| ``address``
|	The address of the account under which the code is executing, to be returned by the `ADDRESS` instruction.
| ``secretKey``
|	The secret key as can be derived by the v,r,s values if the transaction.
| ``to``
|	The address of the transaction's recipient, to be returned by the `ORIGIN` instruction.
| ``value`` 
|	The value of the transaction (or the endowment of the create), to be returned by the `CALLVALUE`` instruction (if executed first, before any `CALL`).
| 

The post Section
^^^^^^^^^^^^^^^^

``Indexes`` section describes which values from given array to set for transaction
before it's execution on a pre state. Transaction now has data, value, and gasLimit as arrays.
post section now has array of implemented forks. For each fork it has another array
of execution results on that fork rules with post state root hash and transaction parameters.

The pre Section
^^^^^^^^^^^^^^^

The ``pre`` section have the format of a mapping between addresses and accounts. 
Each account has the format:

| ``balance``
|	The balance of the account.
| ``nonce``
|	The nonce of the account.
| ``code``
|	The body code of the account, given as an array of byte values. See $DATA_ARRAY.
| ``storage``
|	The account's storage, given as a mapping of keys to values. For key used notion of string as digital or hex number e.g: ``"1200"`` or ``"0x04B0"`` For values used $DATA_ARRAY.
|


