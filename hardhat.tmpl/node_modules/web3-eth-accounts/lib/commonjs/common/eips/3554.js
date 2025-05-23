"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = {
    name: 'EIP-3554',
    comment: 'Reduction in refunds',
    url: 'Difficulty Bomb Delay to December 1st 2021',
    status: 'Final',
    minimumHardfork: 'muirGlacier',
    requiredEIPs: [],
    gasConfig: {},
    gasPrices: {},
    vm: {},
    pow: {
        difficultyBombDelay: {
            v: 9500000,
            d: 'the amount of blocks to delay the difficulty bomb with',
        },
    },
};
//# sourceMappingURL=3554.js.map