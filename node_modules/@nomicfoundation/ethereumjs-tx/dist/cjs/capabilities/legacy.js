"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.getSenderPublicKey = exports.validateHighS = exports.hash = exports.getDataFee = exports.isSigned = exports.errorMsg = void 0;
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const keccak_js_1 = require("ethereum-cryptography/keccak.js");
const baseTransaction_js_1 = require("../baseTransaction.js");
const types_js_1 = require("../types.js");
function keccak256(msg) {
    return new Uint8Array((0, keccak_js_1.keccak256)(Buffer.from(msg)));
}
function errorMsg(tx, msg) {
    return `${msg} (${tx.errorStr()})`;
}
exports.errorMsg = errorMsg;
function isSigned(tx) {
    const { v, r, s } = tx;
    if (v === undefined || r === undefined || s === undefined) {
        return false;
    }
    else {
        return true;
    }
}
exports.isSigned = isSigned;
/**
 * The amount of gas paid for the data in this tx
 */
function getDataFee(tx, extraCost) {
    if (tx.cache.dataFee && tx.cache.dataFee.hardfork === tx.common.hardfork()) {
        return tx.cache.dataFee.value;
    }
    const cost = baseTransaction_js_1.BaseTransaction.prototype.getDataFee.bind(tx)() + (extraCost ?? 0n);
    if (Object.isFrozen(tx)) {
        tx.cache.dataFee = {
            value: cost,
            hardfork: tx.common.hardfork(),
        };
    }
    return cost;
}
exports.getDataFee = getDataFee;
function hash(tx) {
    if (!tx.isSigned()) {
        const msg = errorMsg(tx, 'Cannot call hash method if transaction is not signed');
        throw new Error(msg);
    }
    const keccakFunction = tx.common.customCrypto.keccak256 ?? keccak256;
    if (Object.isFrozen(tx)) {
        if (!tx.cache.hash) {
            tx.cache.hash = keccakFunction(tx.serialize());
        }
        return tx.cache.hash;
    }
    return keccakFunction(tx.serialize());
}
exports.hash = hash;
/**
 * EIP-2: All transaction signatures whose s-value is greater than secp256k1n/2are considered invalid.
 * Reasoning: https://ethereum.stackexchange.com/a/55728
 */
function validateHighS(tx) {
    const { s } = tx;
    if (tx.common.gteHardfork('homestead') && s !== undefined && s > ethereumjs_util_1.SECP256K1_ORDER_DIV_2) {
        const msg = errorMsg(tx, 'Invalid Signature: s-values greater than secp256k1n/2 are considered invalid');
        throw new Error(msg);
    }
}
exports.validateHighS = validateHighS;
function getSenderPublicKey(tx) {
    if (tx.cache.senderPubKey !== undefined) {
        return tx.cache.senderPubKey;
    }
    const msgHash = tx.getMessageToVerifySignature();
    const { v, r, s } = tx;
    validateHighS(tx);
    try {
        const ecrecoverFunction = tx.common.customCrypto.ecrecover ?? ethereumjs_util_1.ecrecover;
        const sender = ecrecoverFunction(msgHash, v, (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(r), (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(s), tx.supports(types_js_1.Capability.EIP155ReplayProtection) ? tx.common.chainId() : undefined);
        if (Object.isFrozen(tx)) {
            tx.cache.senderPubKey = sender;
        }
        return sender;
    }
    catch (e) {
        const msg = errorMsg(tx, 'Invalid Signature');
        throw new Error(msg);
    }
}
exports.getSenderPublicKey = getSenderPublicKey;
//# sourceMappingURL=legacy.js.map