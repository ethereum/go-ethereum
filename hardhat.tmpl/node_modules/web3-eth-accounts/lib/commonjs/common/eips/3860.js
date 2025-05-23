"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.default = {
    name: 'EIP-3860',
    number: 3860,
    comment: 'Limit and meter initcode',
    url: 'https://eips.ethereum.org/EIPS/eip-3860',
    status: 'Review',
    minimumHardfork: 'spuriousDragon',
    requiredEIPs: [],
    gasConfig: {},
    gasPrices: {
        initCodeWordCost: {
            v: 2,
            d: 'Gas to pay for each word (32 bytes) of initcode when creating a contract',
        },
    },
    vm: {
        maxInitCodeSize: {
            v: 49152,
            d: 'Maximum length of initialization code when creating a contract',
        },
    },
    pow: {},
};
//# sourceMappingURL=3860.js.map