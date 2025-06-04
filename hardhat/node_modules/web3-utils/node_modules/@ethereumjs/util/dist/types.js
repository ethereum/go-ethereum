"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.toType = exports.TypeOutput = void 0;
const bytes_1 = require("./bytes");
const internal_1 = require("./internal");
/**
 * Type output options
 */
var TypeOutput;
(function (TypeOutput) {
    TypeOutput[TypeOutput["Number"] = 0] = "Number";
    TypeOutput[TypeOutput["BigInt"] = 1] = "BigInt";
    TypeOutput[TypeOutput["Buffer"] = 2] = "Buffer";
    TypeOutput[TypeOutput["PrefixedHexString"] = 3] = "PrefixedHexString";
})(TypeOutput = exports.TypeOutput || (exports.TypeOutput = {}));
function toType(input, outputType) {
    if (input === null) {
        return null;
    }
    if (input === undefined) {
        return undefined;
    }
    if (typeof input === 'string' && !(0, internal_1.isHexString)(input)) {
        throw new Error(`A string must be provided with a 0x-prefix, given: ${input}`);
    }
    else if (typeof input === 'number' && !Number.isSafeInteger(input)) {
        throw new Error('The provided number is greater than MAX_SAFE_INTEGER (please use an alternative input type)');
    }
    const output = (0, bytes_1.toBuffer)(input);
    switch (outputType) {
        case TypeOutput.Buffer:
            return output;
        case TypeOutput.BigInt:
            return (0, bytes_1.bufferToBigInt)(output);
        case TypeOutput.Number: {
            const bigInt = (0, bytes_1.bufferToBigInt)(output);
            if (bigInt > BigInt(Number.MAX_SAFE_INTEGER)) {
                throw new Error('The provided number is greater than MAX_SAFE_INTEGER (please use an alternative output type)');
            }
            return Number(bigInt);
        }
        case TypeOutput.PrefixedHexString:
            return (0, bytes_1.bufferToHex)(output);
        default:
            throw new Error('unknown outputType');
    }
}
exports.toType = toType;
//# sourceMappingURL=types.js.map