export default {
    name: 'EIP-4844',
    number: 4844,
    comment: 'Shard Blob Transactions',
    url: 'https://eips.ethereum.org/EIPS/eip-4844',
    status: 'Draft',
    minimumHardfork: 'merge',
    requiredEIPs: [1559, 2718, 2930, 4895],
    gasConfig: {
        dataGasPerBlob: {
            v: 131072,
            d: 'The base fee for data gas per blob',
        },
        targetDataGasPerBlock: {
            v: 262144,
            d: 'The target data gas consumed per block',
        },
        maxDataGasPerBlock: {
            v: 524288,
            d: 'The max data gas allowable per block',
        },
        dataGasPriceUpdateFraction: {
            v: 2225652,
            d: 'The denominator used in the exponential when calculating a data gas price',
        },
    },
    gasPrices: {
        simpleGasPerBlob: {
            v: 12000,
            d: 'The basic gas fee for each blob',
        },
        minDataGasPrice: {
            v: 1,
            d: 'The minimum fee per data gas',
        },
        kzgPointEvaluationGasPrecompilePrice: {
            v: 50000,
            d: 'The fee associated with the point evaluation precompile',
        },
        datahash: {
            v: 3,
            d: 'Base fee of the DATAHASH opcode',
        },
    },
    sharding: {
        blobCommitmentVersionKzg: {
            v: 1,
            d: 'The number indicated a versioned hash is a KZG commitment',
        },
        fieldElementsPerBlob: {
            v: 4096,
            d: 'The number of field elements allowed per blob',
        },
    },
    vm: {},
    pow: {},
};
//# sourceMappingURL=4844.js.map