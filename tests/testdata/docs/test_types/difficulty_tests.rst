.. _difficulty_tests:

Difficulty Tests
================

These tests are designed to just check the difficulty formula of a block.

=================== ==============================================================
Location            `\BasicTests <https://github.com/ethereum/tests/tree/develop/BasicTests>`_  (difficulty*.json)
Supported Hardforks ``Test Networks`` | ``Frontier`` | ``Homestead``
Status              Outdated
=================== ==============================================================

difficulty = DIFFICULTY(currentBlockNumber, currentTimestamp, parentTimestamp, parentDifficulty)

described at `EIP2 <https://github.com/ethereum/EIPs/blob/master/EIPS/eip-2.mediawiki>`_ point 4 with homestead changes.

So basically this .json tests are just to check how this function is calculated on different function parameters (parentDifficulty, currentNumber) in its extremum points. 

There are several test files:

``difficulty.json``
	Normal Frontier/Homestead chain difficulty tests defined manually
``difficultyFrontier.json``
	Same as above, but auto-generated tests
``difficultyMorden.json``
	Tests for testnetwork difficulty. (it has different homestead transition block)
``difficultyOlimpic.json``
	Olympic network. (no homestead)
``difficultyHomestead.json``
	Tests for homestead difficulty (regardless of the block number)
``difficultyCustomHomestead.json``
	Tests for homestead difficulty (regardless of the block number)

Test Structure
--------------
::

	{
		"difficultyTest" : {
			"parentTimestamp" : "42",
			"parentDifficulty" : "1000000",
			"currentTimestamp" : "43",
			"currentBlockNumber" : "42",
			"currentDifficulty" : "1000488"
		}
	}

Sections
^^^^^^^^

* ``parentTimestamp`` - indicates the timestamp of a previous block
* ``parentDifficulty`` - indicates the difficulty of a previous block
* ``currentTimestamp`` - indicates the timestamp of a current block
* ``currentBlockNumber`` - indicates the number of a current block (previous block number = currentBlockNumber - 1)
* ``currentDifficulty`` - indicates the difficulty of a current block
