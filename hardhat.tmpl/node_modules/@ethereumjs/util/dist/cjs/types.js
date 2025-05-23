"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toType = exports.TypeOutput = exports.isNestedUint8Array = void 0;
const bytes_js_1 = require("./bytes.js");
const internal_js_1 = require("./internal.js");
function isNestedUint8Array(value) {
    if (!Array.isArray(value)) {
        return false;
    }
    for (const item of value) {
        if (Array.isArray(item)) {
            if (!isNestedUint8Array(item)) {
                return false;
            }
        }
        else if (!(item instanceof Uint8Array)) {
            return false;
        }
    }
    return true;
}
exports.isNestedUint8Array = isNestedUint8Array;
/**
 * Type output options
 */
var TypeOutput;
(function (TypeOutput) {
    TypeOutput[TypeOutput["Number"] = 0] = "Number";
    TypeOutput[TypeOutput["BigInt"] = 1] = "BigInt";
    TypeOutput[TypeOutput["Uint8Array"] = 2] = "Uint8Array";
    TypeOutput[TypeOutput["PrefixedHexString"] = 3] = "PrefixedHexString";
})(TypeOutput = exports.TypeOutput || (exports.TypeOutput = {}));
function toType(input, outputType) {
    if (input === null) {
        return null;
    }
    if (input === undefined) {
        return undefined;
    }
    if (typeof input === 'string' && !(0, internal_js_1.isHexString)(input)) {
        throw new Error(`A string must be provided with a 0x-prefix, given: ${input}`);
    }
    else if (typeof input === 'number' && !Number.isSafeInteger(input)) {
        throw new Error('The provided number is greater than MAX_SAFE_INTEGER (please use an alternative input type)');
    }
    const output = (0, bytes_js_1.toBytes)(input);
    switch (outputType) {
        case TypeOutput.Uint8Array:
            return output;
        case TypeOutput.BigInt:
            return (0, bytes_js_1.bytesToBigInt)(output);
        case TypeOutput.Number: {
            const bigInt = (0, bytes_js_1.bytesToBigInt)(output);
            if (bigInt > BigInt(Number.MAX_SAFE_INTEGER)) {
                throw new Error('The provided number is greater than MAX_SAFE_INTEGER (please use an alternative output type)');
            }
            return Number(bigInt);
        }
        case TypeOutput.PrefixedHexString:
            return (0, bytes_js_1.bytesToHex)(output);
        default:
            throw new Error('unknown outputType');
    }
}
exports.toType = toType;
//# sourceMappingURL=types.js.map