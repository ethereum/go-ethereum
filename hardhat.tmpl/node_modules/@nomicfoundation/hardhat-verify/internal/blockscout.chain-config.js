"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.builtinChains = void 0;
// See https://github.com/ethereum/EIPs/blob/master/EIPS/eip-155.md#list-of-chain-ids
exports.builtinChains = [
    {
        network: "mainnet",
        chainId: 1,
        urls: {
            apiURL: "https://eth.blockscout.com/api",
            browserURL: "https://eth.blockscout.com/",
        },
    },
    {
        network: "sepolia",
        chainId: 11155111,
        urls: {
            apiURL: "https://eth-sepolia.blockscout.com/api",
            browserURL: "https://eth-sepolia.blockscout.com/",
        },
    },
];
//# sourceMappingURL=blockscout.chain-config.js.map