export default {
    name: 'petersburg',
    comment: 'Aka constantinopleFix, removes EIP-1283, activate together with or after constantinople',
    url: 'https://eips.ethereum.org/EIPS/eip-1716',
    status: 'Final',
    gasConfig: {},
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
    vm: {},
    pow: {},
};
//# sourceMappingURL=petersburg.js.map