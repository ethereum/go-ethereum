"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.extractPublicKey = exports.recoverPersonalSignature = exports.personalSign = void 0;
const ethereumjs_util_1 = require("ethereumjs-util");
const utils_1 = require("./utils");
/**
 * Create an Ethereum-specific signature for a message.
 *
 * This function is equivalent to the `eth_sign` Ethereum JSON-RPC method as specified in EIP-1417,
 * as well as the MetaMask's `personal_sign` method.
 *
 * @param options - The personal sign options.
 * @param options.privateKey - The key to sign with.
 * @param options.data - The hex data to sign.
 * @returns The '0x'-prefixed hex encoded signature.
 */
function personalSign({ privateKey, data, }) {
    if (utils_1.isNullish(data)) {
        throw new Error('Missing data parameter');
    }
    else if (utils_1.isNullish(privateKey)) {
        throw new Error('Missing privateKey parameter');
    }
    const message = utils_1.legacyToBuffer(data);
    const msgHash = ethereumjs_util_1.hashPersonalMessage(message);
    const sig = ethereumjs_util_1.ecsign(msgHash, privateKey);
    const serialized = utils_1.concatSig(ethereumjs_util_1.toBuffer(sig.v), sig.r, sig.s);
    return serialized;
}
exports.personalSign = personalSign;
/**
 * Recover the address of the account used to create the given Ethereum signature. The message
 * must have been signed using the `personalSign` function, or an equivalent function.
 *
 * @param options - The signature recovery options.
 * @param options.data - The hex data that was signed.
 * @param options.signature - The '0x'-prefixed hex encoded message signature.
 * @returns The '0x'-prefixed hex encoded address of the message signer.
 */
function recoverPersonalSignature({ data, signature, }) {
    if (utils_1.isNullish(data)) {
        throw new Error('Missing data parameter');
    }
    else if (utils_1.isNullish(signature)) {
        throw new Error('Missing signature parameter');
    }
    const publicKey = getPublicKeyFor(data, signature);
    const sender = ethereumjs_util_1.publicToAddress(publicKey);
    const senderHex = ethereumjs_util_1.bufferToHex(sender);
    return senderHex;
}
exports.recoverPersonalSignature = recoverPersonalSignature;
/**
 * Recover the public key of the account used to create the given Ethereum signature. The message
 * must have been signed using the `personalSign` function, or an equivalent function.
 *
 * @param options - The public key recovery options.
 * @param options.data - The hex data that was signed.
 * @param options.signature - The '0x'-prefixed hex encoded message signature.
 * @returns The '0x'-prefixed hex encoded public key of the message signer.
 */
function extractPublicKey({ data, signature, }) {
    if (utils_1.isNullish(data)) {
        throw new Error('Missing data parameter');
    }
    else if (utils_1.isNullish(signature)) {
        throw new Error('Missing signature parameter');
    }
    const publicKey = getPublicKeyFor(data, signature);
    return `0x${publicKey.toString('hex')}`;
}
exports.extractPublicKey = extractPublicKey;
/**
 * Get the public key for the given signature and message.
 *
 * @param message - The message that was signed.
 * @param signature - The '0x'-prefixed hex encoded message signature.
 * @returns The public key of the signer.
 */
function getPublicKeyFor(message, signature) {
    const messageHash = ethereumjs_util_1.hashPersonalMessage(utils_1.legacyToBuffer(message));
    return utils_1.recoverPublicKey(messageHash, signature);
}
//# sourceMappingURL=personal-sign.js.map