"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toHex = exports.fromBigIntLike = exports.toEvmWord = exports.cmp = exports.divUp = exports.isBigInt = exports.max = exports.min = void 0;
const errors_1 = require("../core/errors");
function min(x, y) {
    return x < y ? x : y;
}
exports.min = min;
function max(x, y) {
    return x > y ? x : y;
}
exports.max = max;
function isBigInt(x) {
    return typeof x === "bigint";
}
exports.isBigInt = isBigInt;
function divUp(x, y) {
    let result = x / y;
    if (x % y !== 0n) {
        result = result + 1n;
    }
    return result;
}
exports.divUp = divUp;
function cmp(a, b) {
    return a < b ? -1 : a > b ? 1 : 0;
}
exports.cmp = cmp;
/**
 * Converts the number to a hexadecimal string with a length of 32
 * bytes. This hex string is NOT 0x-prefixed.
 */
function toEvmWord(x) {
    return x.toString(16).padStart(64, "0");
}
exports.toEvmWord = toEvmWord;
function bufferToBigInt(x) {
    const hex = `0x${Buffer.from(x).toString("hex")}`;
    return hex === "0x" ? 0n : BigInt(hex);
}
function fromBigIntLike(x) {
    if (x === undefined || typeof x === "bigint") {
        return x;
    }
    if (typeof x === "number" || typeof x === "string") {
        return BigInt(x);
    }
    if (x instanceof Uint8Array) {
        return bufferToBigInt(x);
    }
    const exhaustiveCheck = x;
    return exhaustiveCheck;
}
exports.fromBigIntLike = fromBigIntLike;
function toHex(x) {
    (0, errors_1.assertHardhatInvariant)(x >= 0, `toHex can only be used with non-negative numbers, but received ${x}`);
    return `0x${x.toString(16)}`;
}
exports.toHex = toHex;
//# sourceMappingURL=bigint.js.map