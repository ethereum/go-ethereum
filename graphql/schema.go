// Copyright 2018 The go-ethereum Authors
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

package graphql

const schema string = `
    # Bytes32 is a 32 byte binary string, represented as 0x-prefixed hexadecimal.
    scalar Bytes32
    # Address is a 20 byte Ethereum address, represented as 0x-prefixed hexadecimal.
    scalar Address
    # Bytes is an arbitrary length binary string, represented as 0x-prefixed hexadecimal.
    scalar Bytes
    # BigInt is a large integer. Input is accepted as either a JSON number or as a string.
    # Strings may be either decimal or 0x-prefixed hexadecimal. Output values are all
    # 0x-prefixed hexadecimal.
    scalar BigInt
    # Long is a 64 bit unsigned integer.
    scalar Long

    schema {
        query: Query
        mutation: Mutation
    }

    # Account is an Ethereum account at a particular block.
    type Account {
        # Address is the address owning the account.
        address: Address!
        # Balance is the balance of the account, in wei.
        balance: BigInt!
        # TransactionCount is the number of transactions sent from this account,
        # or in the case of a contract, the number of contracts created. Otherwise
        # known as the nonce.
        transactionCount: Long!
        # Code contains the smart contract code for this account, if the account
        # is a (non-self-destructed) contract.
        code: Bytes!
        # Storage provides access to the storage of a contract account, indexed
        # by its 32 byte slot identifier.
        storage(slot: Bytes32!): Bytes32!
    }

    # Log is an Ethereum event log.
    type Log {
        # Index is the index of this log in the block.
        index: Int!
        # Account is the account which generated this log - this will always
        # be a contract account.
        account(block: Long): Account!
        # Topics is a list of 0-4 indexed topics for the log.
        topics: [Bytes32!]!
        # Data is unindexed data for this log.
        data: Bytes!
        # Transaction is the transaction that generated this log entry.
        transaction: Transaction!
    }

    # Transaction is an Ethereum transaction.
    type Transaction {
        # Hash is the hash of this transaction.
        hash: Bytes32!
        # Nonce is the nonce of the account this transaction was generated with.
        nonce: Long!
        # Index is the index of this transaction in the parent block. This will
        # be null if the transaction has not yet beenn mined.
        index: Int
        # From is the account that sent this transaction - this will always be
        # an externally owned account.
        from(block: Long): Account!
        # To is the account the transaction was sent to. This is null for
        # contract-creating transactions.
        to(block: Long): Account
        # Value is the value, in wei, sent along with this transaction.
        value: BigInt!
        # GasPrice is the price offered to miners for gas, in wei per unit.
        gasPrice: BigInt!
        # Gas is the maximum amount of gas this transaction can consume.
        gas: Long!
        # InputData is the data supplied to the target of the transaction.
        inputData: Bytes!
        # Block is the block this transaction was mined in. This will be null if
        # the transaction has not yet been mined.
        block: Block

        # Status is the return status of the transaction. This will be 1 if the
        # transaction succeeded, or 0 if it failed (due to a revert, or due to
        # running out of gas). If the transaction has not yet been mined, this
        # field will be null.
        status: Long
        # GasUsed is the amount of gas that was used processing this transaction.
        # If the transaction has not yet been mined, this field will be null.
        gasUsed: Long
        # CumulativeGasUsed is the total gas used in the block up to and including
        # this transaction. If the transaction has not yet been mined, this field
        # will be null.
        cumulativeGasUsed: Long
        # CreatedContract is the account that was created by a contract creation
        # transaction. If the transaction was not a contract creation transaction,
        # or it has not yet been mined, this field will be null.
        createdContract(block: Long): Account
        # Logs is a list of log entries emitted by this transaction. If the
        # transaction has not yet been mined, this field will be null.
        logs: [Log!]
    }

    # BlockFilterCriteria encapsulates log filter criteria for a filter applied
    # to a single block.
    input BlockFilterCriteria {
        # Addresses is list of addresses that are of interest. If this list is
        # empty, results will not be filtered by address.
        addresses: [Address!]
        # Topics list restricts matches to particular event topics. Each event has a list
    	# of topics. Topics matches a prefix of that list. An empty element array matches any
    	# topic. Non-empty elements represent an alternative that matches any of the
    	# contained topics.
    	#
    	# Examples:
    	#  - [] or nil          matches any topic list
    	#  - [[A]]              matches topic A in first position
    	#  - [[], [B]]          matches any topic in first position, B in second position
    	#  - [[A], [B]]         matches topic A in first position, B in second position
    	#  - [[A, B]], [C, D]]  matches topic (A OR B) in first position, (C OR D) in second position
        topics: [[Bytes32!]!]
    }

    # Block is an Ethereum block.
    type Block {
        # Number is the number of this block, starting at 0 for the genesis block.
        number: Long!
        # Hash is the block hash of this block.
        hash: Bytes32!
        # Parent is the parent block of this block.
        parent: Block
        # Nonce is the block nonce, an 8 byte sequence determined by the miner.
        nonce: Bytes!
        # TransactionsRoot is the keccak256 hash of the root of the trie of transactions in this block.
        transactionsRoot: Bytes32!
        # TransactionCount is the number of transactions in this block. if
        # transactions are not available for this block, this field will be null.
        transactionCount: Int
        # StateRoot is the keccak256 hash of the state trie after this block was processed.
        stateRoot: Bytes32!
        # ReceiptsRoot is the keccak256 hash of the trie of transaction receipts in this block.
        receiptsRoot: Bytes32!
        # Miner is the account that mined this block.
        miner(block: Long): Account!
        # ExtraData is an arbitrary data field supplied by the miner.
        extraData: Bytes!
        # GasLimit is the maximum amount of gas that was available to transactions in this block.
        gasLimit: Long!
        # GasUsed is the amount of gas that was used executing transactions in this block.
        gasUsed: Long!
        # Timestamp is the unix timestamp at which this block was mined.
        timestamp: BigInt!
        # LogsBloom is a bloom filter that can be used to check if a block may
        # contain log entries matching a filter.
        logsBloom: Bytes!
        # MixHash is the hash that was used as an input to the PoW process.
        mixHash: Bytes32!
        # Difficulty is a measure of the difficulty of mining this block.
        difficulty: BigInt!
        # TotalDifficulty is the sum of all difficulty values up to and including
        # this block.
        totalDifficulty: BigInt!
        # OmmerCount is the number of ommers (AKA uncles) associated with this
        # block. If ommers are unavailable, this field will be null.
        ommerCount: Int
        # Ommers is a list of ommer (AKA uncle) blocks associated with this block.
        # If ommers are unavailable, this field will be null. Depending on your
        # node, the transactions, transactionAt, transactionCount, ommers,
        # ommerCount and ommerAt fields may not be available on any ommer blocks.
        ommers: [Block]
        # OmmerAt returns the ommer (AKA uncle) at the specified index. If ommers
        # are unavailable, or the index is out of bounds, this field will be null.
        ommerAt(index: Int!): Block
        # OmmerHash is the keccak256 hash of all the ommers (AKA uncles)
        # associated with this block.
        ommerHash: Bytes32!
        # Transactions is a list of transactions associated with this block. If
        # transactions are unavailable for this block, this field will be null.
        transactions: [Transaction!]
        # TransactionAt returns the transaction at the specified index. If
        # transactions are unavailable for this block, or if the index is out of
        # bounds, this field will be null.
        transactionAt(index: Int!): Transaction
        # Logs returns a filtered set of logs from this block.
        logs(filter: BlockFilterCriteria!): [Log!]!
    }

    # CallData represents the data associated with a local contract call.
    # All fields are optional.
    input CallData {
        # From is the address making the call.
        from: Address
        # To is the address the call is sent to.
        to: Address
        # Gas is the amount of gas sent with the call.
        gas: Long
        # GasPrice is the price, in wei, offered for each unit of gas.
        gasPrice: BigInt
        # Value is the value, in wei, sent along with the call.
        value: BigInt
        # Data is the data sent to the callee.
        data: Bytes
    }

    # CallResult is the result of a local call operationn.
    type CallResult {
        # Data is the return data of the called contract.
        data: Bytes!
        # GasUsed is the amount of gas used by the call, after any refunds.
        gasUsed: Long!
        # Status is the result of the call - 1 for success or 0 for failure.
        status: Long!
    }

    # FilterCriteria encapsulates log filter criteria for searching log entries.
    input FilterCriteria {
        # FromBlock is the block at which to start searching, inclusive. Defaults
        # to the latest block if not supplied.
        fromBlock: Long
        # ToBlock is the block at which to stop searching, inclusive. Defaults
        # to the latest block if not supplied.
        toBlock: Long
        # Addresses is a list of addresses that are of interest. If this list is
        # empty, results will not be filtered by address.
        addresses: [Address!]
        # Topics list restricts matches to particular event topics. Each event has a list
    	# of topics. Topics matches a prefix of that list. An empty element array matches any
    	# topic. Non-empty elements represent an alternative that matches any of the
    	# contained topics.
    	#
    	# Examples:
    	#  - [] or nil          matches any topic list
    	#  - [[A]]              matches topic A in first position
    	#  - [[], [B]]          matches any topic in first position, B in second position
    	#  - [[A], [B]]         matches topic A in first position, B in second position
    	#  - [[A, B]], [C, D]]  matches topic (A OR B) in first position, (C OR D) in second position
        topics: [[Bytes32!]!]
    }

    # SyncState contains the current synchronisation state of the client.
    type SyncState{
        # StartingBlock is the block number at which synchronisation started.
        startingBlock: Long!
        # CurrentBlock is the point at which synchronisation has presently reached.
        currentBlock: Long!
        # HighestBlock is the latest known block number.
        highestBlock: Long!
        # PulledStates is the number of state entries fetched so far, or null
        # if this is not known or not relevant.
        pulledStates: Long
        # KnownStates is the number of states the node knows of so far, or null
        # if this is not known or not relevant.
        knownStates: Long
    }

    type Query {
        # Account fetches an Ethereum account at the specified block number.
        # If blockNumber is not provided, it defaults to the most recent block.
        account(address: Address!, blockNumber: Long): Account!
        # Block fetches an Ethereum block by number or by hash. If neither is
        # supplied, the most recent known block is returned.
        block(number: Long, hash: Bytes32): Block
        # Blocks returns all the blocks between two numbers, inclusive. If
        # to is not supplied, it defaults to the most recent known block.
        blocks(from: Long!, to: Long): [Block!]!
        # Transaction returns a transaction specified by its hash.
        transaction(hash: Bytes32!): Transaction
        # Call executes a local call operation. If blockNumber is not specified,
        # it defaults to the most recent known block.
        call(data: CallData!, blockNumber: Long): CallResult
        # EstimateGas estimates the amount of gas that will be required for
        # successful execution of a transaction. If blockNumber is not specified,
        # it defaults ot the most recent known block.
        estimateGas(data: CallData!, blockNumber: Long): Long!
        # Logs returns log entries matching the provided filter.
        logs(filter: FilterCriteria!): [Log!]!
        # GasPrice returns the node's estimate of a gas price sufficient to
        # ensure a transaction is mined in a timely fashion.
        gasPrice: BigInt!
        # ProtocolVersion returns the current wire protocol version number.
        protocolVersion: Int!
        # Syncing returns information on the current synchronisation state.
        syncing: SyncState
    }

    type Mutation {
        # SendRawTransaction sends an RLP-encoded transaction to the network.
        sendRawTransaction(data: Bytes!): Bytes32!
    }
`
