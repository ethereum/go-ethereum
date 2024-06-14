"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.validateYParity = exports.serialize = exports.getHashedMessageToSign = void 0;
const ethereumjs_rlp_1 = require("@nomicfoundation/ethereumjs-rlp");
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const keccak_js_1 = require("ethereum-cryptography/keccak.js");
const util_js_1 = require("../util.js");
const legacy_js_1 = require("./legacy.js");
function keccak256(msg) {
    return new Uint8Array((0, keccak_js_1.keccak256)(Buffer.from(msg)));
}
function getHashedMessageToSign(tx) {
    const keccakFunction = tx.common.customCrypto.keccak256 ?? keccak256;
    return keccakFunction(tx.getMessageToSign());
}
exports.getHashedMessageToSign = getHashedMessageToSign;
function serialize(tx, base) {
    return (0, ethereumjs_util_1.concatBytes)((0, util_js_1.txTypeBytes)(tx.type), ethereumjs_rlp_1.RLP.encode(base ?? tx.raw()));
}
exports.serialize = serialize;
function validateYParity(tx) {
    const { v } = tx;
    if (v !== undefined && v !== ethereumjs_util_1.BIGINT_0 && v !== ethereumjs_util_1.BIGINT_1) {
        const msg = (0, legacy_js_1.errorMsg)(tx, 'The y-parity of the transaction should either be 0 or 1');
        throw new Error(msg);
    }
}
exports.validateYParity = validateYParity;
//# sourceMappingURL=eip2718.js.map