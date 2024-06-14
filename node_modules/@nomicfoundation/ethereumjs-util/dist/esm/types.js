import { bytesToBigInt, bytesToHex, toBytes } from './bytes.js';
import { isHexString } from './internal.js';
/**
 * Type output options
 */
export var TypeOutput;
(function (TypeOutput) {
    TypeOutput[TypeOutput["Number"] = 0] = "Number";
    TypeOutput[TypeOutput["BigInt"] = 1] = "BigInt";
    TypeOutput[TypeOutput["Uint8Array"] = 2] = "Uint8Array";
    TypeOutput[TypeOutput["PrefixedHexString"] = 3] = "PrefixedHexString";
})(TypeOutput || (TypeOutput = {}));
export function toType(input, outputType) {
    if (input === null) {
        return null;
    }
    if (input === undefined) {
        return undefined;
    }
    if (typeof input === 'string' && !isHexString(input)) {
        throw new Error(`A string must be provided with a 0x-prefix, given: ${input}`);
    }
    else if (typeof input === 'number' && !Number.isSafeInteger(input)) {
        throw new Error('The provided number is greater than MAX_SAFE_INTEGER (please use an alternative input type)');
    }
    const output = toBytes(input);
    switch (outputType) {
        case TypeOutput.Uint8Array:
            return output;
        case TypeOutput.BigInt:
            return bytesToBigInt(output);
        case TypeOutput.Number: {
            const bigInt = bytesToBigInt(output);
            if (bigInt > BigInt(Number.MAX_SAFE_INTEGER)) {
                throw new Error('The provided number is greater than MAX_SAFE_INTEGER (please use an alternative output type)');
            }
            return Number(bigInt);
        }
        case TypeOutput.PrefixedHexString:
            return bytesToHex(output);
        default:
            throw new Error('unknown outputType');
    }
}
//# sourceMappingURL=types.js.map