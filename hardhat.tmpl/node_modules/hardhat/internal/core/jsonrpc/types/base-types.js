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
exports.rpcDataToBuffer = exports.bufferToRpcData = exports.rpcDataToBigInt = exports.rpcDataToNumber = exports.numberToRpcStorageSlot = exports.numberToRpcQuantity = exports.rpcQuantityToBigInt = exports.rpcQuantityToNumber = exports.rpcFloat = exports.rpcQuantityAsNumber = exports.rpcUnsignedInteger = exports.rpcAddress = exports.rpcStorageSlotHexString = exports.rpcStorageSlot = exports.rpcHash = exports.rpcParity = exports.rpcData = exports.rpcQuantity = void 0;
const util_1 = require("@ethereumjs/util");
const t = __importStar(require("io-ts"));
const BigIntUtils = __importStar(require("../../../util/bigint"));
const errors_1 = require("../../errors");
const errors_list_1 = require("../../errors-list");
const ADDRESS_LENGTH_BYTES = 20;
const HASH_LENGTH_BYTES = 32;
exports.rpcQuantity = new t.Type("QUANTITY", BigIntUtils.isBigInt, (u, c) => (isRpcQuantityString(u) ? t.success(BigInt(u)) : t.failure(u, c)), t.identity);
exports.rpcData = new t.Type("DATA", Buffer.isBuffer, (u, c) => isRpcDataString(u) ? t.success(Buffer.from((0, util_1.toBytes)(u))) : t.failure(u, c), t.identity);
exports.rpcParity = new t.Type("PARITY", Buffer.isBuffer, (u, c) => isRpcParityString(u) ? t.success(Buffer.from((0, util_1.toBytes)(u))) : t.failure(u, c), t.identity);
exports.rpcHash = new t.Type("HASH", (v) => Buffer.isBuffer(v) && v.length === HASH_LENGTH_BYTES, (u, c) => isRpcHashString(u) ? t.success(Buffer.from((0, util_1.toBytes)(u))) : t.failure(u, c), t.identity);
exports.rpcStorageSlot = new t.Type("Storage slot", BigIntUtils.isBigInt, validateStorageSlot, t.identity);
// This type is necessary because objects' keys need to be either strings or numbers to be properly handled by the 'io-ts' module.
// If they are not defined as strings or numbers, the type definition will result in an empty object without the required properties.
// For example, instead of displaying { ke1: value1 }, it will display {}
exports.rpcStorageSlotHexString = new t.Type("Storage slot hex string", (x) => typeof x === "string", (u, c) => validateRpcStorageSlotHexString(u) ? t.success(u) : t.failure(u, c), t.identity);
function validateStorageSlot(u, c) {
    if (typeof u !== "string") {
        return t.failure(u, c, `Storage slot argument must be a string, got '${u}'`);
    }
    if (u === "") {
        return t.failure(u, c, "Storage slot argument cannot be an empty string");
    }
    if (u.startsWith("0x")) {
        if (u.length > 66) {
            return t.failure(u, c, `Storage slot argument must have a length of at most 66 ("0x" + 32 bytes), but '${u}' has a length of ${u.length}`);
        }
    }
    else {
        if (u.length > 64) {
            return t.failure(u, c, `Storage slot argument must have a length of at most 64 (32 bytes), but '${u}' has a length of ${u.length}`);
        }
    }
    if (u.match(/^(0x)?([0-9a-fA-F]){0,64}$/) === null) {
        return t.failure(u, c, `Storage slot argument must be a valid hexadecimal, got '${u}'`);
    }
    return t.success(u === "0x" ? 0n : BigInt(u.startsWith("0x") ? u : `0x${u}`));
}
exports.rpcAddress = new t.Type("ADDRESS", (v) => Buffer.isBuffer(v) && v.length === ADDRESS_LENGTH_BYTES, (u, c) => isRpcAddressString(u)
    ? t.success(Buffer.from((0, util_1.toBytes)(u)))
    : t.failure(u, c), t.identity);
exports.rpcUnsignedInteger = new t.Type("Unsigned integer", isInteger, (u, c) => (isInteger(u) && u >= 0 ? t.success(u) : t.failure(u, c)), t.identity);
exports.rpcQuantityAsNumber = new t.Type("Integer", BigIntUtils.isBigInt, (u, c) => (isInteger(u) ? t.success(BigInt(u)) : t.failure(u, c)), t.identity);
exports.rpcFloat = new t.Type("Float number", isNumber, (u, c) => (typeof u === "number" ? t.success(u) : t.failure(u, c)), t.identity);
// Conversion functions
/**
 * Transforms a QUANTITY into a number. It should only be used if you are 100% sure that the value
 * fits in a number.
 */
function rpcQuantityToNumber(quantity) {
    return Number(rpcQuantityToBigInt(quantity));
}
exports.rpcQuantityToNumber = rpcQuantityToNumber;
function rpcQuantityToBigInt(quantity) {
    // We validate it in case a value gets here through a cast or any
    if (!isRpcQuantityString(quantity)) {
        throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.INVALID_RPC_QUANTITY_VALUE, {
            value: quantity,
        });
    }
    return BigInt(quantity);
}
exports.rpcQuantityToBigInt = rpcQuantityToBigInt;
function numberToRpcQuantity(n) {
    (0, errors_1.assertHardhatInvariant)(typeof n === "number" || typeof n === "bigint", "Expected number");
    return `0x${n.toString(16)}`;
}
exports.numberToRpcQuantity = numberToRpcQuantity;
function numberToRpcStorageSlot(n) {
    (0, errors_1.assertHardhatInvariant)(typeof n === "number" || typeof n === "bigint", "Expected number");
    return `0x${BigIntUtils.toEvmWord(n)}`;
}
exports.numberToRpcStorageSlot = numberToRpcStorageSlot;
/**
 * Transforms a DATA into a number. It should only be used if you are 100% sure that the data
 * represents a value fits in a number.
 */
function rpcDataToNumber(data) {
    return Number(rpcDataToBigInt(data));
}
exports.rpcDataToNumber = rpcDataToNumber;
function rpcDataToBigInt(data) {
    return data === "0x" ? 0n : BigInt(data);
}
exports.rpcDataToBigInt = rpcDataToBigInt;
function bufferToRpcData(buffer, padToBytes = 0) {
    let s = (0, util_1.bytesToHex)(buffer);
    if (padToBytes > 0 && s.length < padToBytes * 2 + 2) {
        s = `0x${"0".repeat(padToBytes * 2 + 2 - s.length)}${s.slice(2)}`;
    }
    return s;
}
exports.bufferToRpcData = bufferToRpcData;
function rpcDataToBuffer(data) {
    // We validate it in case a value gets here through a cast or any
    if (!isRpcDataString(data)) {
        throw new errors_1.HardhatError(errors_list_1.ERRORS.NETWORK.INVALID_RPC_DATA_VALUE, {
            value: data,
        });
    }
    return Buffer.from((0, util_1.toBytes)(data));
}
exports.rpcDataToBuffer = rpcDataToBuffer;
// Type guards
function validateRpcStorageSlotHexString(u) {
    return typeof u === "string" && /^0x([0-9a-fA-F]){64}$/.test(u);
}
function isRpcQuantityString(u) {
    return (typeof u === "string" &&
        u.match(/^0x(?:0|(?:[1-9a-fA-F][0-9a-fA-F]*))$/) !== null);
}
function isRpcDataString(u) {
    return typeof u === "string" && u.match(/^0x(?:[0-9a-fA-F]{2})*$/) !== null;
}
function isRpcParityString(u) {
    return typeof u === "string" && u.match(/^0x[0-9a-fA-F]{1,2}$/) !== null;
}
function isRpcHashString(u) {
    return typeof u === "string" && u.length === 66 && isRpcDataString(u);
}
function isRpcAddressString(u) {
    return typeof u === "string" && (0, util_1.isValidAddress)(u);
}
function isInteger(num) {
    return Number.isInteger(num);
}
function isNumber(num) {
    return typeof num === "number";
}
//# sourceMappingURL=base-types.js.map