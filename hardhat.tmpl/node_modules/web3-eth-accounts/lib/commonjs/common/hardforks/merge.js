"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = {
    name: 'merge',
    comment: 'Hardfork to upgrade the consensus mechanism to Proof-of-Stake',
    url: 'https://github.com/ethereum/execution-specs/blob/master/network-upgrades/mainnet-upgrades/merge.md',
    status: 'Final',
    consensus: {
        type: 'pos',
        algorithm: 'casper',
        casper: {},
    },
    eips: [3675, 4399],
};
//# sourceMappingURL=merge.js.map