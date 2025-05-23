"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.accountSchema = exports.storageProofSchema = exports.feeHistorySchema = exports.SignatureObjectSchema = exports.transactionReceiptSchema = exports.syncSchema = exports.logSchema = exports.blockHeaderSchema = exports.blockSchema = exports.withdrawalsSchema = exports.transactionInfoSchema = exports.transactionSchema = exports.customChainSchema = exports.hardforkSchema = exports.chainSchema = exports.accessListResultSchema = exports.accessListSchema = exports.accessListItemSchema = void 0;
/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
exports.accessListItemSchema = {
    type: 'object',
    properties: {
        address: {
            format: 'address',
        },
        storageKeys: {
            type: 'array',
            items: {
                format: 'bytes32',
            },
        },
    },
};
exports.accessListSchema = {
    type: 'array',
    items: Object.assign({}, exports.accessListItemSchema),
};
exports.accessListResultSchema = {
    type: 'object',
    properties: {
        accessList: Object.assign({}, exports.accessListSchema),
        gasUsed: {
            type: 'string',
        },
    },
};
exports.chainSchema = {
    type: 'string',
    enum: ['goerli', 'kovan', 'mainnet', 'rinkeby', 'ropsten', 'sepolia'],
};
exports.hardforkSchema = {
    type: 'string',
    enum: [
        'arrowGlacier',
        'berlin',
        'byzantium',
        'chainstart',
        'constantinople',
        'dao',
        'homestead',
        'istanbul',
        'london',
        'merge',
        'muirGlacier',
        'petersburg',
        'shanghai',
        'spuriousDragon',
        'tangerineWhistle',
    ],
};
exports.customChainSchema = {
    type: 'object',
    properties: {
        name: {
            format: 'string',
        },
        networkId: {
            format: 'uint',
        },
        chainId: {
            format: 'uint',
        },
    },
};
exports.transactionSchema = {
    type: 'object',
    properties: {
        from: {
            format: 'address',
        },
        to: {
            oneOf: [{ format: 'address' }, { type: 'null' }],
        },
        value: {
            format: 'uint',
        },
        gas: {
            format: 'uint',
        },
        gasPrice: {
            format: 'uint',
        },
        effectiveGasPrice: {
            format: 'uint',
        },
        type: {
            format: 'uint',
        },
        maxFeePerGas: {
            format: 'uint',
        },
        maxPriorityFeePerGas: {
            format: 'uint',
        },
        accessList: Object.assign({}, exports.accessListSchema),
        data: {
            format: 'bytes',
        },
        input: {
            format: 'bytes',
        },
        nonce: {
            format: 'uint',
        },
        chain: Object.assign({}, exports.chainSchema),
        hardfork: Object.assign({}, exports.hardforkSchema),
        chainId: {
            format: 'uint',
        },
        networkId: {
            format: 'uint',
        },
        common: {
            type: 'object',
            properties: {
                customChain: Object.assign({}, exports.customChainSchema),
                baseChain: Object.assign({}, exports.chainSchema),
                hardfork: Object.assign({}, exports.hardforkSchema),
            },
        },
        gasLimit: {
            format: 'uint',
        },
        v: {
            format: 'uint',
        },
        r: {
            format: 'bytes32',
        },
        s: {
            format: 'bytes32',
        },
    },
};
exports.transactionInfoSchema = {
    type: 'object',
    properties: Object.assign(Object.assign({}, exports.transactionSchema.properties), { blockHash: {
            format: 'bytes32',
        }, blockNumber: {
            format: 'uint',
        }, hash: {
            format: 'bytes32',
        }, transactionIndex: {
            format: 'uint',
        }, from: {
            format: 'address',
        }, to: {
            oneOf: [{ format: 'address' }, { type: 'null' }],
        }, value: {
            format: 'uint',
        }, gas: {
            format: 'uint',
        }, gasPrice: {
            format: 'uint',
        }, effectiveGasPrice: {
            format: 'uint',
        }, type: {
            format: 'uint',
        }, maxFeePerGas: {
            format: 'uint',
        }, maxPriorityFeePerGas: {
            format: 'uint',
        }, accessList: Object.assign({}, exports.accessListSchema), data: {
            format: 'bytes',
        }, input: {
            format: 'bytes',
        }, nonce: {
            format: 'uint',
        }, gasLimit: {
            format: 'uint',
        }, v: {
            format: 'uint',
        }, r: {
            format: 'bytes32',
        }, s: {
            format: 'bytes32',
        } }),
};
exports.withdrawalsSchema = {
    type: 'object',
    properties: {
        index: {
            format: 'uint',
        },
        validatorIndex: {
            format: 'uint',
        },
        address: {
            format: 'address',
        },
        amount: {
            format: 'uint',
        },
    },
};
exports.blockSchema = {
    type: 'object',
    properties: {
        baseFeePerGas: {
            format: 'uint',
        },
        blobGasUsed: {
            format: 'uint',
        },
        difficulty: {
            format: 'uint',
        },
        excessBlobGas: {
            format: 'uint',
        },
        extraData: {
            format: 'bytes',
        },
        gasLimit: {
            format: 'uint',
        },
        gasUsed: {
            format: 'uint',
        },
        hash: {
            format: 'bytes32',
        },
        logsBloom: {
            format: 'bytes256',
        },
        miner: {
            format: 'bytes',
        },
        mixHash: {
            format: 'bytes32',
        },
        nonce: {
            format: 'uint',
        },
        number: {
            format: 'uint',
        },
        parentBeaconBlockRoot: {
            format: 'bytes32',
        },
        parentHash: {
            format: 'bytes32',
        },
        receiptsRoot: {
            format: 'bytes32',
        },
        sha3Uncles: {
            format: 'bytes32',
        },
        size: {
            format: 'uint',
        },
        stateRoot: {
            format: 'bytes32',
        },
        timestamp: {
            format: 'uint',
        },
        totalDifficulty: {
            format: 'uint',
        },
        transactions: {
            oneOf: [
                {
                    type: 'array',
                    items: Object.assign({}, exports.transactionInfoSchema),
                },
                {
                    type: 'array',
                    items: {
                        format: 'bytes32',
                    },
                },
            ],
        },
        transactionsRoot: {
            format: 'bytes32',
        },
        uncles: {
            type: 'array',
            items: {
                format: 'bytes32',
            },
        },
        withdrawals: {
            type: 'array',
            items: Object.assign({}, exports.withdrawalsSchema),
        },
        withdrawalsRoot: {
            format: 'bytes32',
        },
    },
};
exports.blockHeaderSchema = {
    type: 'object',
    properties: {
        author: {
            format: 'bytes32',
        },
        excessDataGas: {
            format: 'uint',
        },
        baseFeePerGas: {
            format: 'uint',
        },
        blobGasUsed: {
            format: 'uint',
        },
        difficulty: {
            format: 'uint',
        },
        excessBlobGas: {
            format: 'uint',
        },
        extraData: {
            format: 'bytes',
        },
        gasLimit: {
            format: 'uint',
        },
        gasUsed: {
            format: 'uint',
        },
        hash: {
            format: 'bytes32',
        },
        logsBloom: {
            format: 'bytes256',
        },
        miner: {
            format: 'bytes',
        },
        mixHash: {
            format: 'bytes32',
        },
        nonce: {
            format: 'uint',
        },
        number: {
            format: 'uint',
        },
        parentBeaconBlockRoot: {
            format: 'bytes32',
        },
        parentHash: {
            format: 'bytes32',
        },
        receiptsRoot: {
            format: 'bytes32',
        },
        sha3Uncles: {
            format: 'bytes32',
        },
        size: {
            format: 'uint',
        },
        stateRoot: {
            format: 'bytes32',
        },
        timestamp: {
            format: 'uint',
        },
        totalDifficulty: {
            format: 'uint',
        },
        transactions: {
            type: 'array',
            items: {
                format: 'bytes32',
            },
        },
        transactionsRoot: {
            format: 'bytes32',
        },
        uncles: {
            type: 'array',
            items: {
                format: 'bytes32',
            },
        },
        withdrawals: {
            type: 'array',
            items: Object.assign({}, exports.withdrawalsSchema),
        },
        withdrawalsRoot: {
            format: 'bytes32',
        },
    },
};
exports.logSchema = {
    type: 'object',
    properties: {
        removed: {
            format: 'bool',
        },
        logIndex: {
            format: 'uint',
        },
        transactionIndex: {
            format: 'uint',
        },
        transactionHash: {
            format: 'bytes32',
        },
        blockHash: {
            format: 'bytes32',
        },
        blockNumber: {
            format: 'uint',
        },
        address: {
            format: 'address',
        },
        data: {
            format: 'bytes',
        },
        topics: {
            type: 'array',
            items: {
                format: 'bytes32',
            },
        },
    },
};
exports.syncSchema = {
    type: 'object',
    properties: {
        startingBlock: {
            format: 'string',
        },
        currentBlock: {
            format: 'string',
        },
        highestBlock: {
            format: 'string',
        },
        knownStates: {
            format: 'string',
        },
        pulledStates: {
            format: 'string',
        },
    },
};
exports.transactionReceiptSchema = {
    type: 'object',
    properties: {
        transactionHash: {
            format: 'bytes32',
        },
        transactionIndex: {
            format: 'uint',
        },
        blockHash: {
            format: 'bytes32',
        },
        blockNumber: {
            format: 'uint',
        },
        from: {
            format: 'address',
        },
        to: {
            format: 'address',
        },
        cumulativeGasUsed: {
            format: 'uint',
        },
        gasUsed: {
            format: 'uint',
        },
        effectiveGasPrice: {
            format: 'uint',
        },
        contractAddress: {
            format: 'address',
        },
        logs: {
            type: 'array',
            items: Object.assign({}, exports.logSchema),
        },
        logsBloom: {
            format: 'bytes',
        },
        root: {
            format: 'bytes',
        },
        status: {
            format: 'uint',
        },
        type: {
            format: 'uint',
        },
    },
};
exports.SignatureObjectSchema = {
    type: 'object',
    properties: {
        messageHash: {
            format: 'bytes',
        },
        r: {
            format: 'bytes32',
        },
        s: {
            format: 'bytes32',
        },
        v: {
            format: 'bytes',
        },
        message: {
            format: 'bytes',
        },
        signature: {
            format: 'bytes',
        },
    },
};
exports.feeHistorySchema = {
    type: 'object',
    properties: {
        oldestBlock: {
            format: 'uint',
        },
        baseFeePerGas: {
            type: 'array',
            items: {
                format: 'uint',
            },
        },
        reward: {
            type: 'array',
            items: {
                type: 'array',
                items: {
                    format: 'uint',
                },
            },
        },
        gasUsedRatio: {
            type: 'array',
            items: {
                type: 'number',
            },
        },
    },
};
exports.storageProofSchema = {
    type: 'object',
    properties: {
        key: {
            format: 'bytes32',
        },
        value: {
            format: 'uint',
        },
        proof: {
            type: 'array',
            items: {
                format: 'bytes32',
            },
        },
    },
};
exports.accountSchema = {
    type: 'object',
    properties: {
        balance: {
            format: 'uint',
        },
        codeHash: {
            format: 'bytes32',
        },
        nonce: {
            format: 'uint',
        },
        storageHash: {
            format: 'bytes32',
        },
        accountProof: {
            type: 'array',
            items: {
                format: 'bytes32',
            },
        },
        storageProof: {
            type: 'array',
            items: Object.assign({}, exports.storageProofSchema),
        },
    },
};
//# sourceMappingURL=schemas.js.map