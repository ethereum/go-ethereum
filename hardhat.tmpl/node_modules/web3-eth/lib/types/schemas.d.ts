export declare const accessListItemSchema: {
    type: string;
    properties: {
        address: {
            format: string;
        };
        storageKeys: {
            type: string;
            items: {
                format: string;
            };
        };
    };
};
export declare const accessListSchema: {
    type: string;
    items: {
        type: string;
        properties: {
            address: {
                format: string;
            };
            storageKeys: {
                type: string;
                items: {
                    format: string;
                };
            };
        };
    };
};
export declare const accessListResultSchema: {
    type: string;
    properties: {
        accessList: {
            type: string;
            items: {
                type: string;
                properties: {
                    address: {
                        format: string;
                    };
                    storageKeys: {
                        type: string;
                        items: {
                            format: string;
                        };
                    };
                };
            };
        };
        gasUsed: {
            type: string;
        };
    };
};
export declare const chainSchema: {
    type: string;
    enum: string[];
};
export declare const hardforkSchema: {
    type: string;
    enum: string[];
};
export declare const customChainSchema: {
    type: string;
    properties: {
        name: {
            format: string;
        };
        networkId: {
            format: string;
        };
        chainId: {
            format: string;
        };
    };
};
export declare const transactionSchema: {
    type: string;
    properties: {
        from: {
            format: string;
        };
        to: {
            oneOf: ({
                format: string;
                type?: undefined;
            } | {
                type: string;
                format?: undefined;
            })[];
        };
        value: {
            format: string;
        };
        gas: {
            format: string;
        };
        gasPrice: {
            format: string;
        };
        effectiveGasPrice: {
            format: string;
        };
        type: {
            format: string;
        };
        maxFeePerGas: {
            format: string;
        };
        maxPriorityFeePerGas: {
            format: string;
        };
        accessList: {
            type: string;
            items: {
                type: string;
                properties: {
                    address: {
                        format: string;
                    };
                    storageKeys: {
                        type: string;
                        items: {
                            format: string;
                        };
                    };
                };
            };
        };
        data: {
            format: string;
        };
        input: {
            format: string;
        };
        nonce: {
            format: string;
        };
        chain: {
            type: string;
            enum: string[];
        };
        hardfork: {
            type: string;
            enum: string[];
        };
        chainId: {
            format: string;
        };
        networkId: {
            format: string;
        };
        common: {
            type: string;
            properties: {
                customChain: {
                    type: string;
                    properties: {
                        name: {
                            format: string;
                        };
                        networkId: {
                            format: string;
                        };
                        chainId: {
                            format: string;
                        };
                    };
                };
                baseChain: {
                    type: string;
                    enum: string[];
                };
                hardfork: {
                    type: string;
                    enum: string[];
                };
            };
        };
        gasLimit: {
            format: string;
        };
        v: {
            format: string;
        };
        r: {
            format: string;
        };
        s: {
            format: string;
        };
    };
};
export declare const transactionInfoSchema: {
    type: string;
    properties: {
        blockHash: {
            format: string;
        };
        blockNumber: {
            format: string;
        };
        hash: {
            format: string;
        };
        transactionIndex: {
            format: string;
        };
        from: {
            format: string;
        };
        to: {
            oneOf: ({
                format: string;
                type?: undefined;
            } | {
                type: string;
                format?: undefined;
            })[];
        };
        value: {
            format: string;
        };
        gas: {
            format: string;
        };
        gasPrice: {
            format: string;
        };
        effectiveGasPrice: {
            format: string;
        };
        type: {
            format: string;
        };
        maxFeePerGas: {
            format: string;
        };
        maxPriorityFeePerGas: {
            format: string;
        };
        accessList: {
            type: string;
            items: {
                type: string;
                properties: {
                    address: {
                        format: string;
                    };
                    storageKeys: {
                        type: string;
                        items: {
                            format: string;
                        };
                    };
                };
            };
        };
        data: {
            format: string;
        };
        input: {
            format: string;
        };
        nonce: {
            format: string;
        };
        gasLimit: {
            format: string;
        };
        v: {
            format: string;
        };
        r: {
            format: string;
        };
        s: {
            format: string;
        };
        chain: {
            type: string;
            enum: string[];
        };
        hardfork: {
            type: string;
            enum: string[];
        };
        chainId: {
            format: string;
        };
        networkId: {
            format: string;
        };
        common: {
            type: string;
            properties: {
                customChain: {
                    type: string;
                    properties: {
                        name: {
                            format: string;
                        };
                        networkId: {
                            format: string;
                        };
                        chainId: {
                            format: string;
                        };
                    };
                };
                baseChain: {
                    type: string;
                    enum: string[];
                };
                hardfork: {
                    type: string;
                    enum: string[];
                };
            };
        };
    };
};
export declare const withdrawalsSchema: {
    type: string;
    properties: {
        index: {
            format: string;
        };
        validatorIndex: {
            format: string;
        };
        address: {
            format: string;
        };
        amount: {
            format: string;
        };
    };
};
export declare const blockSchema: {
    type: string;
    properties: {
        baseFeePerGas: {
            format: string;
        };
        blobGasUsed: {
            format: string;
        };
        difficulty: {
            format: string;
        };
        excessBlobGas: {
            format: string;
        };
        extraData: {
            format: string;
        };
        gasLimit: {
            format: string;
        };
        gasUsed: {
            format: string;
        };
        hash: {
            format: string;
        };
        logsBloom: {
            format: string;
        };
        miner: {
            format: string;
        };
        mixHash: {
            format: string;
        };
        nonce: {
            format: string;
        };
        number: {
            format: string;
        };
        parentBeaconBlockRoot: {
            format: string;
        };
        parentHash: {
            format: string;
        };
        receiptsRoot: {
            format: string;
        };
        sha3Uncles: {
            format: string;
        };
        size: {
            format: string;
        };
        stateRoot: {
            format: string;
        };
        timestamp: {
            format: string;
        };
        totalDifficulty: {
            format: string;
        };
        transactions: {
            oneOf: ({
                type: string;
                items: {
                    type: string;
                    properties: {
                        blockHash: {
                            format: string;
                        };
                        blockNumber: {
                            format: string;
                        };
                        hash: {
                            format: string;
                        };
                        transactionIndex: {
                            format: string;
                        };
                        from: {
                            format: string;
                        };
                        to: {
                            oneOf: ({
                                format: string;
                                type?: undefined;
                            } | {
                                type: string;
                                format?: undefined;
                            })[];
                        };
                        value: {
                            format: string;
                        };
                        gas: {
                            format: string;
                        };
                        gasPrice: {
                            format: string;
                        };
                        effectiveGasPrice: {
                            format: string;
                        };
                        type: {
                            format: string;
                        };
                        maxFeePerGas: {
                            format: string;
                        };
                        maxPriorityFeePerGas: {
                            format: string;
                        };
                        accessList: {
                            type: string;
                            items: {
                                type: string;
                                properties: {
                                    address: {
                                        format: string;
                                    };
                                    storageKeys: {
                                        type: string;
                                        items: {
                                            format: string;
                                        };
                                    };
                                };
                            };
                        };
                        data: {
                            format: string;
                        };
                        input: {
                            format: string;
                        };
                        nonce: {
                            format: string;
                        };
                        gasLimit: {
                            format: string;
                        };
                        v: {
                            format: string;
                        };
                        r: {
                            format: string;
                        };
                        s: {
                            format: string;
                        };
                        chain: {
                            type: string;
                            enum: string[];
                        };
                        hardfork: {
                            type: string;
                            enum: string[];
                        };
                        chainId: {
                            format: string;
                        };
                        networkId: {
                            format: string;
                        };
                        common: {
                            type: string;
                            properties: {
                                customChain: {
                                    type: string;
                                    properties: {
                                        name: {
                                            format: string;
                                        };
                                        networkId: {
                                            format: string;
                                        };
                                        chainId: {
                                            format: string;
                                        };
                                    };
                                };
                                baseChain: {
                                    type: string;
                                    enum: string[];
                                };
                                hardfork: {
                                    type: string;
                                    enum: string[];
                                };
                            };
                        };
                    };
                    format?: undefined;
                };
            } | {
                type: string;
                items: {
                    format: string;
                };
            })[];
        };
        transactionsRoot: {
            format: string;
        };
        uncles: {
            type: string;
            items: {
                format: string;
            };
        };
        withdrawals: {
            type: string;
            items: {
                type: string;
                properties: {
                    index: {
                        format: string;
                    };
                    validatorIndex: {
                        format: string;
                    };
                    address: {
                        format: string;
                    };
                    amount: {
                        format: string;
                    };
                };
            };
        };
        withdrawalsRoot: {
            format: string;
        };
    };
};
export declare const blockHeaderSchema: {
    type: string;
    properties: {
        author: {
            format: string;
        };
        excessDataGas: {
            format: string;
        };
        baseFeePerGas: {
            format: string;
        };
        blobGasUsed: {
            format: string;
        };
        difficulty: {
            format: string;
        };
        excessBlobGas: {
            format: string;
        };
        extraData: {
            format: string;
        };
        gasLimit: {
            format: string;
        };
        gasUsed: {
            format: string;
        };
        hash: {
            format: string;
        };
        logsBloom: {
            format: string;
        };
        miner: {
            format: string;
        };
        mixHash: {
            format: string;
        };
        nonce: {
            format: string;
        };
        number: {
            format: string;
        };
        parentBeaconBlockRoot: {
            format: string;
        };
        parentHash: {
            format: string;
        };
        receiptsRoot: {
            format: string;
        };
        sha3Uncles: {
            format: string;
        };
        size: {
            format: string;
        };
        stateRoot: {
            format: string;
        };
        timestamp: {
            format: string;
        };
        totalDifficulty: {
            format: string;
        };
        transactions: {
            type: string;
            items: {
                format: string;
            };
        };
        transactionsRoot: {
            format: string;
        };
        uncles: {
            type: string;
            items: {
                format: string;
            };
        };
        withdrawals: {
            type: string;
            items: {
                type: string;
                properties: {
                    index: {
                        format: string;
                    };
                    validatorIndex: {
                        format: string;
                    };
                    address: {
                        format: string;
                    };
                    amount: {
                        format: string;
                    };
                };
            };
        };
        withdrawalsRoot: {
            format: string;
        };
    };
};
export declare const logSchema: {
    type: string;
    properties: {
        removed: {
            format: string;
        };
        logIndex: {
            format: string;
        };
        transactionIndex: {
            format: string;
        };
        transactionHash: {
            format: string;
        };
        blockHash: {
            format: string;
        };
        blockNumber: {
            format: string;
        };
        address: {
            format: string;
        };
        data: {
            format: string;
        };
        topics: {
            type: string;
            items: {
                format: string;
            };
        };
    };
};
export declare const syncSchema: {
    type: string;
    properties: {
        startingBlock: {
            format: string;
        };
        currentBlock: {
            format: string;
        };
        highestBlock: {
            format: string;
        };
        knownStates: {
            format: string;
        };
        pulledStates: {
            format: string;
        };
    };
};
export declare const transactionReceiptSchema: {
    type: string;
    properties: {
        transactionHash: {
            format: string;
        };
        transactionIndex: {
            format: string;
        };
        blockHash: {
            format: string;
        };
        blockNumber: {
            format: string;
        };
        from: {
            format: string;
        };
        to: {
            format: string;
        };
        cumulativeGasUsed: {
            format: string;
        };
        gasUsed: {
            format: string;
        };
        effectiveGasPrice: {
            format: string;
        };
        contractAddress: {
            format: string;
        };
        logs: {
            type: string;
            items: {
                type: string;
                properties: {
                    removed: {
                        format: string;
                    };
                    logIndex: {
                        format: string;
                    };
                    transactionIndex: {
                        format: string;
                    };
                    transactionHash: {
                        format: string;
                    };
                    blockHash: {
                        format: string;
                    };
                    blockNumber: {
                        format: string;
                    };
                    address: {
                        format: string;
                    };
                    data: {
                        format: string;
                    };
                    topics: {
                        type: string;
                        items: {
                            format: string;
                        };
                    };
                };
            };
        };
        logsBloom: {
            format: string;
        };
        root: {
            format: string;
        };
        status: {
            format: string;
        };
        type: {
            format: string;
        };
    };
};
export declare const SignatureObjectSchema: {
    type: string;
    properties: {
        messageHash: {
            format: string;
        };
        r: {
            format: string;
        };
        s: {
            format: string;
        };
        v: {
            format: string;
        };
        message: {
            format: string;
        };
        signature: {
            format: string;
        };
    };
};
export declare const feeHistorySchema: {
    type: string;
    properties: {
        oldestBlock: {
            format: string;
        };
        baseFeePerGas: {
            type: string;
            items: {
                format: string;
            };
        };
        reward: {
            type: string;
            items: {
                type: string;
                items: {
                    format: string;
                };
            };
        };
        gasUsedRatio: {
            type: string;
            items: {
                type: string;
            };
        };
    };
};
export declare const storageProofSchema: {
    type: string;
    properties: {
        key: {
            format: string;
        };
        value: {
            format: string;
        };
        proof: {
            type: string;
            items: {
                format: string;
            };
        };
    };
};
export declare const accountSchema: {
    type: string;
    properties: {
        balance: {
            format: string;
        };
        codeHash: {
            format: string;
        };
        nonce: {
            format: string;
        };
        storageHash: {
            format: string;
        };
        accountProof: {
            type: string;
            items: {
                format: string;
            };
        };
        storageProof: {
            type: string;
            items: {
                type: string;
                properties: {
                    key: {
                        format: string;
                    };
                    value: {
                        format: string;
                    };
                    proof: {
                        type: string;
                        items: {
                            format: string;
                        };
                    };
                };
            };
        };
    };
};
//# sourceMappingURL=schemas.d.ts.map