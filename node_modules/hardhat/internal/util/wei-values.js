"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.weiToHumanReadableString = void 0;
/**
 * This function turns a wei value in a human readable string. It shows values
 * in ETH, gwei or wei, depending on how large it is.
 *
 * It never show more than 99999 wei or gwei, moving to the larger denominator
 * when necessary.
 *
 * It never shows more than 4 decimal digits. Adapting denominator and
 * truncating as necessary.
 */
function weiToHumanReadableString(wei) {
    if (typeof wei === "number") {
        wei = BigInt(wei);
    }
    if (wei === 0n) {
        return "0 ETH";
    }
    if (wei < 100000n) {
        return `${wei.toString()} wei`;
    }
    if (wei < 10n ** 14n) {
        return `${toDecimalString(wei, 9, 4)} gwei`;
    }
    return `${toDecimalString(wei, 18, 4)} ETH`;
}
exports.weiToHumanReadableString = weiToHumanReadableString;
function toDecimalString(value, digitsToInteger, decimalDigits = 4) {
    const oneUnit = 10n ** BigInt(digitsToInteger);
    const oneDecimal = 10n ** BigInt(digitsToInteger - decimalDigits);
    const integer = value / oneUnit;
    const decimals = (value % oneUnit) / oneDecimal;
    if (decimals === 0n) {
        return integer.toString(10);
    }
    const decimalsString = removeRightZeros(decimals.toString(10).padStart(decimalDigits, "0"));
    return `${integer.toString(10)}.${decimalsString}`;
}
function removeRightZeros(str) {
    let zeros = 0;
    for (let i = str.length - 1; i >= 0; i--) {
        if (str.charAt(i) !== "0") {
            break;
        }
        zeros += 1;
    }
    return str.substr(0, str.length - zeros);
}
//# sourceMappingURL=wei-values.js.map