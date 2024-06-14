"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getBalances = exports.getAddresses = void 0;
const account_1 = require("./account");
function getAddresses(accounts) {
    return Promise.all(accounts.map((account) => (0, account_1.getAddressOf)(account)));
}
exports.getAddresses = getAddresses;
async function getBalances(accounts, blockNumber) {
    const { toBigInt } = await Promise.resolve().then(() => __importStar(require("ethers")));
    const hre = await Promise.resolve().then(() => __importStar(require("hardhat")));
    const provider = hre.ethers.provider;
    return Promise.all(accounts.map(async (account) => {
        const address = await (0, account_1.getAddressOf)(account);
        const result = await provider.send("eth_getBalance", [
            address,
            `0x${blockNumber?.toString(16) ?? 0}`,
        ]);
        return toBigInt(result);
    }));
}
exports.getBalances = getBalances;
//# sourceMappingURL=balance.js.map