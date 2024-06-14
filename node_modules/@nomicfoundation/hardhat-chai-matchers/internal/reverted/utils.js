"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.resultToArray = exports.decodeReturnData = exports.getReturnDataFromError = void 0;
const chai_1 = require("chai");
const errors_1 = require("../errors");
const panic_1 = require("./panic");
// method id of 'Error(string)'
const ERROR_STRING_PREFIX = "0x08c379a0";
// method id of 'Panic(uint256)'
const PANIC_CODE_PREFIX = "0x4e487b71";
/**
 * Try to obtain the return data of a transaction from the given value.
 *
 * If the value is an error but it doesn't have data, we assume it's not related
 * to a reverted transaction and we re-throw it.
 */
function getReturnDataFromError(error) {
    if (!(error instanceof Error)) {
        throw new chai_1.AssertionError("Expected an Error object");
    }
    // cast to any again so we don't have to cast it every time we access
    // some property that doesn't exist on Error
    error = error;
    const errorData = error.data ?? error.error?.data;
    if (errorData === undefined) {
        throw error;
    }
    const returnData = typeof errorData === "string" ? errorData : errorData.data;
    if (returnData === undefined || typeof returnData !== "string") {
        throw error;
    }
    return returnData;
}
exports.getReturnDataFromError = getReturnDataFromError;
function decodeReturnData(returnData) {
    const { AbiCoder } = require("ethers");
    const abi = new AbiCoder();
    if (returnData === "0x") {
        return { kind: "Empty" };
    }
    else if (returnData.startsWith(ERROR_STRING_PREFIX)) {
        const encodedReason = returnData.slice(ERROR_STRING_PREFIX.length);
        let reason;
        try {
            reason = abi.decode(["string"], `0x${encodedReason}`)[0];
        }
        catch (e) {
            throw new errors_1.HardhatChaiMatchersDecodingError(encodedReason, "string", e);
        }
        return {
            kind: "Error",
            reason,
        };
    }
    else if (returnData.startsWith(PANIC_CODE_PREFIX)) {
        const encodedReason = returnData.slice(PANIC_CODE_PREFIX.length);
        let code;
        try {
            code = abi.decode(["uint256"], `0x${encodedReason}`)[0];
        }
        catch (e) {
            throw new errors_1.HardhatChaiMatchersDecodingError(encodedReason, "uint256", e);
        }
        const description = (0, panic_1.panicErrorCodeToReason)(code) ?? "unknown panic code";
        return {
            kind: "Panic",
            code,
            description,
        };
    }
    return {
        kind: "Custom",
        id: returnData.slice(0, 10),
        data: `0x${returnData.slice(10)}`,
    };
}
exports.decodeReturnData = decodeReturnData;
/**
 * Takes an ethers result object and converts it into a (potentially nested) array.
 *
 * For example, given this error:
 *
 *   struct Point(uint x, uint y)
 *   error MyError(string, Point)
 *
 *   revert MyError("foo", Point(1, 2))
 *
 * The resulting array will be: ["foo", [1n, 2n]]
 */
function resultToArray(result) {
    return result
        .toArray()
        .map((x) => typeof x === "object" && x !== null && "toArray" in x
        ? resultToArray(x)
        : x);
}
exports.resultToArray = resultToArray;
//# sourceMappingURL=utils.js.map