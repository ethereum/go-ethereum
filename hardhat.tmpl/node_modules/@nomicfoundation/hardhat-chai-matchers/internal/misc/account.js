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
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.getAddressOf = void 0;
const assert_1 = __importDefault(require("assert"));
const errors_1 = require("../errors");
async function getAddressOf(account) {
    const { isAddressable } = await Promise.resolve().then(() => __importStar(require("ethers")));
    if (typeof account === "string") {
        (0, assert_1.default)(/^0x[0-9a-fA-F]{40}$/.test(account), `Invalid address ${account}`);
        return account;
    }
    if (isAddressable(account)) {
        return account.getAddress();
    }
    throw new errors_1.HardhatChaiMatchersAssertionError(`Expected string or addressable, got ${account}`);
}
exports.getAddressOf = getAddressOf;
//# sourceMappingURL=account.js.map