"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = {
    name: 'byzantium',
    comment: 'Hardfork with new precompiles, instructions and other protocol changes',
    url: 'https://eips.ethereum.org/EIPS/eip-609',
    status: 'Final',
    gasConfig: {},
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
    vm: {},
    pow: {
        minerReward: {
            v: '3000000000000000000',
            d: 'the amount a miner get rewarded for mining a block',
        },
        difficultyBombDelay: {
            v: 3000000,
            d: 'the amount of blocks to delay the difficulty bomb with',
        },
    },
};
//# sourceMappingURL=byzantium.js.map