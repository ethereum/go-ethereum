"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = {
    name: 'constantinople',
    comment: 'Postponed hardfork including EIP-1283 (SSTORE gas metering changes)',
    url: 'https://eips.ethereum.org/EIPS/eip-1013',
    status: 'Final',
    gasConfig: {},
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
    vm: {},
    pow: {
        minerReward: {
            v: '2000000000000000000',
            d: 'The amount a miner gets rewarded for mining a block',
        },
        difficultyBombDelay: {
            v: 5000000,
            d: 'the amount of blocks to delay the difficulty bomb with',
        },
    },
};
//# sourceMappingURL=constantinople.js.map