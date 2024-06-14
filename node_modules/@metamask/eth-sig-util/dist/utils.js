"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.normalize = exports.recoverPublicKey = exports.concatSig = exports.legacyToBuffer = exports.isNullish = exports.padWithZeroes = void 0;
const ethereumjs_util_1 = require("ethereumjs-util");
const ethjs_util_1 = require("ethjs-util");
/**
 * Pads the front of the given hex string with zeroes until it reaches the
 * target length. If the input string is already longer than or equal to the
 * target length, it is returned unmodified.
 *
 * If the input string is "0x"-prefixed or not a hex string, an error will be
 * thrown.
 *
 * @param hexString - The hexadecimal string to pad with zeroes.
 * @param targetLength - The target length of the hexadecimal string.
 * @returns The input string front-padded with zeroes, or the original string
 * if it was already greater than or equal to to the target length.
 */
function padWithZeroes(hexString, targetLength) {
    if (hexString !== '' && !/^[a-f0-9]+$/iu.test(hexString)) {
        throw new Error(`Expected an unprefixed hex string. Received: ${hexString}`);
    }
    if (targetLength < 0) {
        throw new Error(`Expected a non-negative integer target length. Received: ${targetLength}`);
    }
    return String.prototype.padStart.call(hexString, targetLength, '0');
}
exports.padWithZeroes = padWithZeroes;
/**
 * Returns `true` if the given value is nullish.
 *
 * @param value - The value being checked.
 * @returns Whether the value is nullish.
 */
function isNullish(value) {
    return value === null || value === undefined;
}
exports.isNullish = isNullish;
/**
 * Convert a value to a Buffer. This function should be equivalent to the `toBuffer` function in
 * `ethereumjs-util@5.2.1`.
 *
 * @param value - The value to convert to a Buffer.
 * @returns The given value as a Buffer.
 */
function legacyToBuffer(value) {
    return typeof value === 'string' && !ethjs_util_1.isHexString(value)
        ? Buffer.from(value)
        : ethereumjs_util_1.toBuffer(value);
}
exports.legacyToBuffer = legacyToBuffer;
/**
 * Concatenate an extended ECDSA signature into a single '0x'-prefixed hex string.
 *
 * @param v - The 'v' portion of the signature.
 * @param r - The 'r' portion of the signature.
 * @param s - The 's' portion of the signature.
 * @returns The concatenated ECDSA signature as a '0x'-prefixed string.
 */
function concatSig(v, r, s) {
    const rSig = ethereumjs_util_1.fromSigned(r);
    const sSig = ethereumjs_util_1.fromSigned(s);
    const vSig = ethereumjs_util_1.bufferToInt(v);
    const rStr = padWithZeroes(ethereumjs_util_1.toUnsigned(rSig).toString('hex'), 64);
    const sStr = padWithZeroes(ethereumjs_util_1.toUnsigned(sSig).toString('hex'), 64);
    const vStr = ethjs_util_1.stripHexPrefix(ethjs_util_1.intToHex(vSig));
    return ethereumjs_util_1.addHexPrefix(rStr.concat(sStr, vStr));
}
exports.concatSig = concatSig;
/**
 * Recover the public key from the given signature and message hash.
 *
 * @param messageHash - The hash of the signed message.
 * @param signature - The signature.
 * @returns The public key of the signer.
 */
function recoverPublicKey(messageHash, signature) {
    const sigParams = ethereumjs_util_1.fromRpcSig(signature);
    return ethereumjs_util_1.ecrecover(messageHash, sigParams.v, sigParams.r, sigParams.s);
}
exports.recoverPublicKey = recoverPublicKey;
/**
 * Normalize the input to a lower-cased '0x'-prefixed hex string.
 *
 * @param input - The value to normalize.
 * @returns The normalized value.
 */
function normalize(input) {
    if (!input) {
        return undefined;
    }
    if (typeof input === 'number') {
        const buffer = ethereumjs_util_1.toBuffer(input);
        input = ethereumjs_util_1.bufferToHex(buffer);
    }
    if (typeof input !== 'string') {
        let msg = 'eth-sig-util.normalize() requires hex string or integer input.';
        msg += ` received ${typeof input}: ${input}`;
        throw new Error(msg);
    }
    return ethereumjs_util_1.addHexPrefix(input.toLowerCase());
}
exports.normalize = normalize;
//# sourceMappingURL=utils.js.map