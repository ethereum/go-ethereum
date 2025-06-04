"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.formatNumberType = exports.isBigNumber = exports.normalizeToBigInt = void 0;
const errors_1 = require("../internal/core/errors");
const errors_list_1 = require("../internal/core/errors-list");
function normalizeToBigInt(source) {
    switch (typeof source) {
        case "object":
            if (isBigNumber(source)) {
                return BigInt(source.toString());
            }
            else {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.INVALID_BIG_NUMBER, {
                    message: `Value ${JSON.stringify(source)} is of type "object" but is not an instanceof one of the known big number object types.`,
                });
            }
        case "number":
            if (!Number.isInteger(source)) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.INVALID_BIG_NUMBER, {
                    message: `${source} is not an integer`,
                });
            }
            if (!Number.isSafeInteger(source)) {
                throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.INVALID_BIG_NUMBER, {
                    message: `Integer ${source} is unsafe. Consider using ${source}n instead. For more details, see https://developer.mozilla.org/en-US/docs/Web/JavaScript/Reference/Global_Objects/Number/isSafeInteger`,
                });
            }
        // `break;` intentionally omitted. fallthrough desired.
        case "string":
        case "bigint":
            return BigInt(source);
        default:
            const _exhaustiveCheck = source;
            throw new errors_1.HardhatError(errors_list_1.ERRORS.GENERAL.INVALID_BIG_NUMBER, {
                message: `Unsupported type ${typeof source}`,
            });
    }
}
exports.normalizeToBigInt = normalizeToBigInt;
function isBigNumber(source) {
    return (typeof source === "bigint" ||
        isEthersBigNumber(source) ||
        isBN(source) ||
        isBigNumberJsBigNumber(source));
}
exports.isBigNumber = isBigNumber;
function isBN(n) {
    try {
        // eslint-disable-next-line import/no-extraneous-dependencies
        const BN = require("bn.js");
        return BN.isBN(n);
    }
    catch (e) {
        return false;
    }
}
function isEthersBigNumber(n) {
    try {
        const BigNumber = 
        // eslint-disable-next-line import/no-extraneous-dependencies
        require("ethers").ethers.BigNumber;
        return BigNumber.isBigNumber(n);
    }
    catch (e) {
        return false;
    }
}
function isBigNumberJsBigNumber(n) {
    try {
        // eslint-disable-next-line import/no-extraneous-dependencies
        const BigNumber = require("bignumber.js").BigNumber;
        return BigNumber.isBigNumber(n);
    }
    catch (e) {
        return false;
    }
}
function formatNumberType(n) {
    if (typeof n === "object") {
        if (isBN(n)) {
            return "BN";
        }
        else if (isEthersBigNumber(n)) {
            return "ethers.BigNumber";
        }
        else if (isBigNumberJsBigNumber(n)) {
            return "bignumber.js";
        }
    }
    return typeof n;
}
exports.formatNumberType = formatNumberType;
//# sourceMappingURL=bigInt.js.map