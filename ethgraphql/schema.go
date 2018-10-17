package ethgraphql

const schema string = `
    scalar Bytes32
    scalar Address
    scalar Bytes
    scalar BigInt
    scalar Long

    schema {
        query: Query
        mutation: Mutation
    }

    type Account {
        address: Address!
        balance: BigInt!
        transactionCount: Long!
        code: Bytes!
        storage(slot: Bytes32!): Bytes32!
    }

    type Log {
        index: Int!
        account(block: Long): Account!
        topics: [Bytes32!]!
        data: Bytes!
        transaction: Transaction!
    }

    type Transaction {
        hash: Bytes32!
        nonce: Long!
        index: Int
        from(block: Long): Account!
        to(block: Long): Account
        value: BigInt!
        gasPrice: BigInt!
        gas: Long!
        inputData: Bytes!
        block: Block

        status: Long
        gasUsed: Long
        cumulativeGasUsed: Long
        createdContract(block: Long): Account
        logs: [Log!]
    }

    input BlockFilterCriteria {
        addresses: [Address!]
        topics: [[Bytes32!]!]
    }

    type Block {
        number: Long!
        hash: Bytes32!
        parent: Block
        nonce: Bytes!
        transactionsRoot: Bytes32!
        transactionCount: Int
        stateRoot: Bytes32!
        receiptsRoot: Bytes32!
        miner(block: Long): Account!
        extraData: Bytes!
        gasLimit: Long!
        gasUsed: Long!
        timestamp: BigInt!
        logsBloom: Bytes!
        mixHash: Bytes32!
        difficulty: BigInt!
        totalDifficulty: BigInt!
        ommerCount: Int
        ommers: [Block]
        ommerAt(index: Int!): Block
        ommerHash: Bytes32!
        transactions: [Transaction!]
        transactionAt(index: Int!): Transaction
        logs(filter: BlockFilterCriteria!): [Log!]!
    }

    input CallData {
        from: Address
        to: Address
        gas: Long
        gasPrice: BigInt
        value: BigInt
        data: Bytes
    }

    type CallResult {
        data: Bytes!
        gasUsed: Long!
        status: Long!
    }

    input FilterCriteria {
        fromBlock: Long
        toBlock: Long
        addresses: [Address!]
        topics: [[Bytes32!]!]
    }

    type SyncState{
        startingBlock: Long!
        currentBlock: Long!
        highestBlock: Long!
        pulledStates: Long
        knownStates: Long
    }

    type Query {
        account(address: Address!, blockNumber: Long): Account!
        block(number: Long, hash: Bytes32): Block
        blocks(from: Long!, to: Long): [Block!]!
        transaction(hash: Bytes32!): Transaction
        call(data: CallData!, blockNumber: Long): CallResult
        estimateGas(data: CallData!, blockNumber: Long): Long!
        logs(filter: FilterCriteria!): [Log!]!
        gasPrice: BigInt!
        protocolVersion: Int!
        syncing: SyncState
    }

    type Mutation {
        sendRawTransaction(data: Bytes!): Bytes32!
    }
`
