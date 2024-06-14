"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.hardforks = void 0;
var Status;
(function (Status) {
    Status["Draft"] = "draft";
    Status["Review"] = "review";
    Status["Final"] = "final";
})(Status || (Status = {}));
exports.hardforks = {
    chainstart: {
        name: 'chainstart',
        comment: 'Start of the Ethereum main chain',
        url: '',
        status: Status.Final,
        gasConfig: {
            minGasLimit: {
                v: 5000,
                d: 'Minimum the gas limit may ever be',
            },
            gasLimitBoundDivisor: {
                v: 1024,
                d: 'The bound divisor of the gas limit, used in update calculations',
            },
            maxRefundQuotient: {
                v: 2,
                d: 'Maximum refund quotient; max tx refund is min(tx.gasUsed/maxRefundQuotient, tx.gasRefund)',
            },
        },
        gasPrices: {
            base: {
                v: 2,
                d: 'Gas base cost, used e.g. for ChainID opcode (Istanbul)',
            },
            exp: {
                v: 10,
                d: 'Base fee of the EXP opcode',
            },
            expByte: {
                v: 10,
                d: 'Times ceil(log256(exponent)) for the EXP instruction',
            },
            keccak256: {
                v: 30,
                d: 'Base fee of the SHA3 opcode',
            },
            keccak256Word: {
                v: 6,
                d: "Once per word of the SHA3 operation's data",
            },
            sload: {
                v: 50,
                d: 'Base fee of the SLOAD opcode',
            },
            sstoreSet: {
                v: 20000,
                d: 'Once per SSTORE operation if the zeroness changes from zero',
            },
            sstoreReset: {
                v: 5000,
                d: 'Once per SSTORE operation if the zeroness does not change from zero',
            },
            sstoreRefund: {
                v: 15000,
                d: 'Once per SSTORE operation if the zeroness changes to zero',
            },
            jumpdest: {
                v: 1,
                d: 'Base fee of the JUMPDEST opcode',
            },
            log: {
                v: 375,
                d: 'Base fee of the LOG opcode',
            },
            logData: {
                v: 8,
                d: "Per byte in a LOG* operation's data",
            },
            logTopic: {
                v: 375,
                d: 'Multiplied by the * of the LOG*, per LOG transaction. e.g. LOG0 incurs 0 * c_txLogTopicGas, LOG4 incurs 4 * c_txLogTopicGas',
            },
            create: {
                v: 32000,
                d: 'Base fee of the CREATE opcode',
            },
            call: {
                v: 40,
                d: 'Base fee of the CALL opcode',
            },
            callStipend: {
                v: 2300,
                d: 'Free gas given at beginning of call',
            },
            callValueTransfer: {
                v: 9000,
                d: 'Paid for CALL when the value transfor is non-zero',
            },
            callNewAccount: {
                v: 25000,
                d: "Paid for CALL when the destination address didn't exist prior",
            },
            selfdestructRefund: {
                v: 24000,
                d: 'Refunded following a selfdestruct operation',
            },
            memory: {
                v: 3,
                d: 'Times the address of the (highest referenced byte in memory + 1). NOTE: referencing happens on read, write and in instructions such as RETURN and CALL',
            },
            quadCoeffDiv: {
                v: 512,
                d: 'Divisor for the quadratic particle of the memory cost equation',
            },
            createData: {
                v: 200,
                d: '',
            },
            tx: {
                v: 21000,
                d: 'Per transaction. NOTE: Not payable on data of calls between transactions',
            },
            txCreation: {
                v: 32000,
                d: 'The cost of creating a contract via tx',
            },
            txDataZero: {
                v: 4,
                d: 'Per byte of data attached to a transaction that equals zero. NOTE: Not payable on data of calls between transactions',
            },
            txDataNonZero: {
                v: 68,
                d: 'Per byte of data attached to a transaction that is not equal to zero. NOTE: Not payable on data of calls between transactions',
            },
            copy: {
                v: 3,
                d: 'Multiplied by the number of 32-byte words that are copied (round up) for any *COPY operation and added',
            },
            ecRecover: {
                v: 3000,
                d: '',
            },
            sha256: {
                v: 60,
                d: '',
            },
            sha256Word: {
                v: 12,
                d: '',
            },
            ripemd160: {
                v: 600,
                d: '',
            },
            ripemd160Word: {
                v: 120,
                d: '',
            },
            identity: {
                v: 15,
                d: '',
            },
            identityWord: {
                v: 3,
                d: '',
            },
            stop: {
                v: 0,
                d: 'Base fee of the STOP opcode',
            },
            add: {
                v: 3,
                d: 'Base fee of the ADD opcode',
            },
            mul: {
                v: 5,
                d: 'Base fee of the MUL opcode',
            },
            sub: {
                v: 3,
                d: 'Base fee of the SUB opcode',
            },
            div: {
                v: 5,
                d: 'Base fee of the DIV opcode',
            },
            sdiv: {
                v: 5,
                d: 'Base fee of the SDIV opcode',
            },
            mod: {
                v: 5,
                d: 'Base fee of the MOD opcode',
            },
            smod: {
                v: 5,
                d: 'Base fee of the SMOD opcode',
            },
            addmod: {
                v: 8,
                d: 'Base fee of the ADDMOD opcode',
            },
            mulmod: {
                v: 8,
                d: 'Base fee of the MULMOD opcode',
            },
            signextend: {
                v: 5,
                d: 'Base fee of the SIGNEXTEND opcode',
            },
            lt: {
                v: 3,
                d: 'Base fee of the LT opcode',
            },
            gt: {
                v: 3,
                d: 'Base fee of the GT opcode',
            },
            slt: {
                v: 3,
                d: 'Base fee of the SLT opcode',
            },
            sgt: {
                v: 3,
                d: 'Base fee of the SGT opcode',
            },
            eq: {
                v: 3,
                d: 'Base fee of the EQ opcode',
            },
            iszero: {
                v: 3,
                d: 'Base fee of the ISZERO opcode',
            },
            and: {
                v: 3,
                d: 'Base fee of the AND opcode',
            },
            or: {
                v: 3,
                d: 'Base fee of the OR opcode',
            },
            xor: {
                v: 3,
                d: 'Base fee of the XOR opcode',
            },
            not: {
                v: 3,
                d: 'Base fee of the NOT opcode',
            },
            byte: {
                v: 3,
                d: 'Base fee of the BYTE opcode',
            },
            address: {
                v: 2,
                d: 'Base fee of the ADDRESS opcode',
            },
            balance: {
                v: 20,
                d: 'Base fee of the BALANCE opcode',
            },
            origin: {
                v: 2,
                d: 'Base fee of the ORIGIN opcode',
            },
            caller: {
                v: 2,
                d: 'Base fee of the CALLER opcode',
            },
            callvalue: {
                v: 2,
                d: 'Base fee of the CALLVALUE opcode',
            },
            calldataload: {
                v: 3,
                d: 'Base fee of the CALLDATALOAD opcode',
            },
            calldatasize: {
                v: 2,
                d: 'Base fee of the CALLDATASIZE opcode',
            },
            calldatacopy: {
                v: 3,
                d: 'Base fee of the CALLDATACOPY opcode',
            },
            codesize: {
                v: 2,
                d: 'Base fee of the CODESIZE opcode',
            },
            codecopy: {
                v: 3,
                d: 'Base fee of the CODECOPY opcode',
            },
            gasprice: {
                v: 2,
                d: 'Base fee of the GASPRICE opcode',
            },
            extcodesize: {
                v: 20,
                d: 'Base fee of the EXTCODESIZE opcode',
            },
            extcodecopy: {
                v: 20,
                d: 'Base fee of the EXTCODECOPY opcode',
            },
            blockhash: {
                v: 20,
                d: 'Base fee of the BLOCKHASH opcode',
            },
            coinbase: {
                v: 2,
                d: 'Base fee of the COINBASE opcode',
            },
            timestamp: {
                v: 2,
                d: 'Base fee of the TIMESTAMP opcode',
            },
            number: {
                v: 2,
                d: 'Base fee of the NUMBER opcode',
            },
            difficulty: {
                v: 2,
                d: 'Base fee of the DIFFICULTY opcode',
            },
            gaslimit: {
                v: 2,
                d: 'Base fee of the GASLIMIT opcode',
            },
            pop: {
                v: 2,
                d: 'Base fee of the POP opcode',
            },
            mload: {
                v: 3,
                d: 'Base fee of the MLOAD opcode',
            },
            mstore: {
                v: 3,
                d: 'Base fee of the MSTORE opcode',
            },
            mstore8: {
                v: 3,
                d: 'Base fee of the MSTORE8 opcode',
            },
            sstore: {
                v: 0,
                d: 'Base fee of the SSTORE opcode',
            },
            jump: {
                v: 8,
                d: 'Base fee of the JUMP opcode',
            },
            jumpi: {
                v: 10,
                d: 'Base fee of the JUMPI opcode',
            },
            pc: {
                v: 2,
                d: 'Base fee of the PC opcode',
            },
            msize: {
                v: 2,
                d: 'Base fee of the MSIZE opcode',
            },
            gas: {
                v: 2,
                d: 'Base fee of the GAS opcode',
            },
            push: {
                v: 3,
                d: 'Base fee of the PUSH opcode',
            },
            dup: {
                v: 3,
                d: 'Base fee of the DUP opcode',
            },
            swap: {
                v: 3,
                d: 'Base fee of the SWAP opcode',
            },
            callcode: {
                v: 40,
                d: 'Base fee of the CALLCODE opcode',
            },
            return: {
                v: 0,
                d: 'Base fee of the RETURN opcode',
            },
            invalid: {
                v: 0,
                d: 'Base fee of the INVALID opcode',
            },
            selfdestruct: {
                v: 0,
                d: 'Base fee of the SELFDESTRUCT opcode',
            },
        },
        vm: {
            stackLimit: {
                v: 1024,
                d: 'Maximum size of VM stack allowed',
            },
            callCreateDepth: {
                v: 1024,
                d: 'Maximum depth of call/create stack',
            },
            maxExtraDataSize: {
                v: 32,
                d: 'Maximum size extra data may be after Genesis',
            },
        },
        pow: {
            minimumDifficulty: {
                v: 131072,
                d: 'The minimum that the difficulty may ever be',
            },
            difficultyBoundDivisor: {
                v: 2048,
                d: 'The bound divisor of the difficulty, used in the update calculations',
            },
            durationLimit: {
                v: 13,
                d: 'The decision boundary on the blocktime duration used to determine whether difficulty should go up or not',
            },
            epochDuration: {
                v: 30000,
                d: 'Duration between proof-of-work epochs',
            },
            timebombPeriod: {
                v: 100000,
                d: 'Exponential difficulty timebomb period',
            },
            minerReward: {
                v: BigInt('5000000000000000000'),
                d: 'the amount a miner get rewarded for mining a block',
            },
            difficultyBombDelay: {
                v: 0,
                d: 'the amount of blocks to delay the difficulty bomb with',
            },
        },
    },
    homestead: {
        name: 'homestead',
        comment: 'Homestead hardfork with protocol and network changes',
        url: 'https://eips.ethereum.org/EIPS/eip-606',
        status: Status.Final,
        gasPrices: {
            delegatecall: {
                v: 40,
                d: 'Base fee of the DELEGATECALL opcode',
            },
        },
    },
    dao: {
        name: 'dao',
        comment: 'DAO rescue hardfork',
        url: 'https://eips.ethereum.org/EIPS/eip-779',
        status: Status.Final,
    },
    tangerineWhistle: {
        name: 'tangerineWhistle',
        comment: 'Hardfork with gas cost changes for IO-heavy operations',
        url: 'https://eips.ethereum.org/EIPS/eip-608',
        status: Status.Final,
        gasPrices: {
            sload: {
                v: 200,
                d: 'Once per SLOAD operation',
            },
            call: {
                v: 700,
                d: 'Once per CALL operation & message call transaction',
            },
            extcodesize: {
                v: 700,
                d: 'Base fee of the EXTCODESIZE opcode',
            },
            extcodecopy: {
                v: 700,
                d: 'Base fee of the EXTCODECOPY opcode',
            },
            balance: {
                v: 400,
                d: 'Base fee of the BALANCE opcode',
            },
            delegatecall: {
                v: 700,
                d: 'Base fee of the DELEGATECALL opcode',
            },
            callcode: {
                v: 700,
                d: 'Base fee of the CALLCODE opcode',
            },
            selfdestruct: {
                v: 5000,
                d: 'Base fee of the SELFDESTRUCT opcode',
            },
        },
    },
    spuriousDragon: {
        name: 'spuriousDragon',
        comment: 'HF with EIPs for simple replay attack protection, EXP cost increase, state trie clearing, contract code size limit',
        url: 'https://eips.ethereum.org/EIPS/eip-607',
        status: Status.Final,
        gasPrices: {
            expByte: {
                v: 50,
                d: 'Times ceil(log256(exponent)) for the EXP instruction',
            },
        },
        vm: {
            maxCodeSize: {
                v: 24576,
                d: 'Maximum length of contract code',
            },
        },
    },
    byzantium: {
        name: 'byzantium',
        comment: 'Hardfork with new precompiles, instructions and other protocol changes',
        url: 'https://eips.ethereum.org/EIPS/eip-609',
        status: Status.Final,
        gasPrices: {
            modexpGquaddivisor: {
                v: 20,
                d: 'Gquaddivisor from modexp precompile for gas calculation',
            },
            ecAdd: {
                v: 500,
                d: 'Gas costs for curve addition precompile',
            },
            ecMul: {
                v: 40000,
                d: 'Gas costs for curve multiplication precompile',
            },
            ecPairing: {
                v: 100000,
                d: 'Base gas costs for curve pairing precompile',
            },
            ecPairingWord: {
                v: 80000,
                d: 'Gas costs regarding curve pairing precompile input length',
            },
            revert: {
                v: 0,
                d: 'Base fee of the REVERT opcode',
            },
            staticcall: {
                v: 700,
                d: 'Base fee of the STATICCALL opcode',
            },
            returndatasize: {
                v: 2,
                d: 'Base fee of the RETURNDATASIZE opcode',
            },
            returndatacopy: {
                v: 3,
                d: 'Base fee of the RETURNDATACOPY opcode',
            },
        },
        pow: {
            minerReward: {
                v: BigInt('3000000000000000000'),
                d: 'the amount a miner get rewarded for mining a block',
            },
            difficultyBombDelay: {
                v: 3000000,
                d: 'the amount of blocks to delay the difficulty bomb with',
            },
        },
    },
    constantinople: {
        name: 'constantinople',
        comment: 'Postponed hardfork including EIP-1283 (SSTORE gas metering changes)',
        url: 'https://eips.ethereum.org/EIPS/eip-1013',
        status: Status.Final,
        gasPrices: {
            netSstoreNoopGas: {
                v: 200,
                d: "Once per SSTORE operation if the value doesn't change",
            },
            netSstoreInitGas: {
                v: 20000,
                d: 'Once per SSTORE operation from clean zero',
            },
            netSstoreCleanGas: {
                v: 5000,
                d: 'Once per SSTORE operation from clean non-zero',
            },
            netSstoreDirtyGas: {
                v: 200,
                d: 'Once per SSTORE operation from dirty',
            },
            netSstoreClearRefund: {
                v: 15000,
                d: 'Once per SSTORE operation for clearing an originally existing storage slot',
            },
            netSstoreResetRefund: {
                v: 4800,
                d: 'Once per SSTORE operation for resetting to the original non-zero value',
            },
            netSstoreResetClearRefund: {
                v: 19800,
                d: 'Once per SSTORE operation for resetting to the original zero value',
            },
            shl: {
                v: 3,
                d: 'Base fee of the SHL opcode',
            },
            shr: {
                v: 3,
                d: 'Base fee of the SHR opcode',
            },
            sar: {
                v: 3,
                d: 'Base fee of the SAR opcode',
            },
            extcodehash: {
                v: 400,
                d: 'Base fee of the EXTCODEHASH opcode',
            },
            create2: {
                v: 32000,
                d: 'Base fee of the CREATE2 opcode',
            },
        },
        pow: {
            minerReward: {
                v: BigInt('2000000000000000000'),
                d: 'The amount a miner gets rewarded for mining a block',
            },
            difficultyBombDelay: {
                v: 5000000,
                d: 'the amount of blocks to delay the difficulty bomb with',
            },
        },
    },
    petersburg: {
        name: 'petersburg',
        comment: 'Aka constantinopleFix, removes EIP-1283, activate together with or after constantinople',
        url: 'https://eips.ethereum.org/EIPS/eip-1716',
        status: Status.Final,
        gasPrices: {
            netSstoreNoopGas: {
                v: null,
                d: 'Removed along EIP-1283',
            },
            netSstoreInitGas: {
                v: null,
                d: 'Removed along EIP-1283',
            },
            netSstoreCleanGas: {
                v: null,
                d: 'Removed along EIP-1283',
            },
            netSstoreDirtyGas: {
                v: null,
                d: 'Removed along EIP-1283',
            },
            netSstoreClearRefund: {
                v: null,
                d: 'Removed along EIP-1283',
            },
            netSstoreResetRefund: {
                v: null,
                d: 'Removed along EIP-1283',
            },
            netSstoreResetClearRefund: {
                v: null,
                d: 'Removed along EIP-1283',
            },
        },
    },
    istanbul: {
        name: 'istanbul',
        comment: 'HF targeted for December 2019 following the Constantinople/Petersburg HF',
        url: 'https://eips.ethereum.org/EIPS/eip-1679',
        status: Status.Final,
        gasConfig: {},
        gasPrices: {
            blake2Round: {
                v: 1,
                d: 'Gas cost per round for the Blake2 F precompile',
            },
            ecAdd: {
                v: 150,
                d: 'Gas costs for curve addition precompile',
            },
            ecMul: {
                v: 6000,
                d: 'Gas costs for curve multiplication precompile',
            },
            ecPairing: {
                v: 45000,
                d: 'Base gas costs for curve pairing precompile',
            },
            ecPairingWord: {
                v: 34000,
                d: 'Gas costs regarding curve pairing precompile input length',
            },
            txDataNonZero: {
                v: 16,
                d: 'Per byte of data attached to a transaction that is not equal to zero. NOTE: Not payable on data of calls between transactions',
            },
            sstoreSentryGasEIP2200: {
                v: 2300,
                d: 'Minimum gas required to be present for an SSTORE call, not consumed',
            },
            sstoreNoopGasEIP2200: {
                v: 800,
                d: "Once per SSTORE operation if the value doesn't change",
            },
            sstoreDirtyGasEIP2200: {
                v: 800,
                d: 'Once per SSTORE operation if a dirty value is changed',
            },
            sstoreInitGasEIP2200: {
                v: 20000,
                d: 'Once per SSTORE operation from clean zero to non-zero',
            },
            sstoreInitRefundEIP2200: {
                v: 19200,
                d: 'Once per SSTORE operation for resetting to the original zero value',
            },
            sstoreCleanGasEIP2200: {
                v: 5000,
                d: 'Once per SSTORE operation from clean non-zero to something else',
            },
            sstoreCleanRefundEIP2200: {
                v: 4200,
                d: 'Once per SSTORE operation for resetting to the original non-zero value',
            },
            sstoreClearRefundEIP2200: {
                v: 15000,
                d: 'Once per SSTORE operation for clearing an originally existing storage slot',
            },
            balance: {
                v: 700,
                d: 'Base fee of the BALANCE opcode',
            },
            extcodehash: {
                v: 700,
                d: 'Base fee of the EXTCODEHASH opcode',
            },
            chainid: {
                v: 2,
                d: 'Base fee of the CHAINID opcode',
            },
            selfbalance: {
                v: 5,
                d: 'Base fee of the SELFBALANCE opcode',
            },
            sload: {
                v: 800,
                d: 'Base fee of the SLOAD opcode',
            },
        },
    },
    muirGlacier: {
        name: 'muirGlacier',
        comment: 'HF to delay the difficulty bomb',
        url: 'https://eips.ethereum.org/EIPS/eip-2384',
        status: Status.Final,
        pow: {
            difficultyBombDelay: {
                v: 9000000,
                d: 'the amount of blocks to delay the difficulty bomb with',
            },
        },
    },
    berlin: {
        name: 'berlin',
        comment: 'HF targeted for July 2020 following the Muir Glacier HF',
        url: 'https://eips.ethereum.org/EIPS/eip-2070',
        status: Status.Final,
        eips: [2565, 2929, 2718, 2930],
    },
    london: {
        name: 'london',
        comment: 'HF targeted for July 2021 following the Berlin fork',
        url: 'https://github.com/ethereum/eth1.0-specs/blob/master/network-upgrades/mainnet-upgrades/london.md',
        status: Status.Final,
        eips: [1559, 3198, 3529, 3541],
    },
    arrowGlacier: {
        name: 'arrowGlacier',
        comment: 'HF to delay the difficulty bomb',
        url: 'https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/arrow-glacier.md',
        status: Status.Final,
        eips: [4345],
    },
    grayGlacier: {
        name: 'grayGlacier',
        comment: 'Delaying the difficulty bomb to Mid September 2022',
        url: 'https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/gray-glacier.md',
        status: Status.Final,
        eips: [5133],
    },
    paris: {
        name: 'paris',
        comment: 'Hardfork to upgrade the consensus mechanism to Proof-of-Stake',
        url: 'https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/merge.md',
        status: Status.Final,
        consensus: {
            type: 'pos',
            algorithm: 'casper',
            casper: {},
        },
        eips: [3675, 4399],
    },
    mergeForkIdTransition: {
        name: 'mergeForkIdTransition',
        comment: 'Pre-merge hardfork to fork off non-upgraded clients',
        url: 'https://eips.ethereum.org/EIPS/eip-3675',
        status: Status.Final,
        eips: [],
    },
    shanghai: {
        name: 'shanghai',
        comment: 'Next feature hardfork after the merge hardfork having withdrawals, warm coinbase, push0, limit/meter initcode',
        url: 'https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/shanghai.md',
        status: Status.Final,
        eips: [3651, 3855, 3860, 4895],
    },
    cancun: {
        name: 'cancun',
        comment: 'Next feature hardfork after shanghai, includes proto-danksharding EIP 4844 blobs (still WIP hence not for production use), transient storage opcodes, parent beacon block root availability in EVM, selfdestruct only in same transaction, and blob base fee opcode',
        url: 'https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/cancun.md',
        status: Status.Final,
        eips: [1153, 4844, 4788, 5656, 6780, 7516],
    },
    prague: {
        name: 'prague',
        comment: 'Next feature hardfork after cancun replaing merkle based state with a verkle based one (incomplete/experimental)',
        url: 'https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/prague.md',
        status: Status.Draft,
        eips: [6800],
    },
};
//# sourceMappingURL=hardforks.js.map