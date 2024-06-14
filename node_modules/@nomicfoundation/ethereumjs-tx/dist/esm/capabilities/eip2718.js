import { RLP } from '@nomicfoundation/ethereumjs-rlp';
import { BIGINT_0, BIGINT_1, concatBytes } from '@nomicfoundation/ethereumjs-util';
import { keccak256 as bufferKeccak256 } from 'ethereum-cryptography/keccak.js';
import { txTypeBytes } from '../util.js';
import { errorMsg } from './legacy.js';
function keccak256(msg) {
    return new Uint8Array(bufferKeccak256(Buffer.from(msg)));
}
export function getHashedMessageToSign(tx) {
    const keccakFunction = tx.common.customCrypto.keccak256 ?? keccak256;
    return keccakFunction(tx.getMessageToSign());
}
export function serialize(tx, base) {
    return concatBytes(txTypeBytes(tx.type), RLP.encode(base ?? tx.raw()));
}
export function validateYParity(tx) {
    const { v } = tx;
    if (v !== undefined && v !== BIGINT_0 && v !== BIGINT_1) {
        const msg = errorMsg(tx, 'The y-parity of the transaction should either be 0 or 1');
        throw new Error(msg);
    }
}
//# sourceMappingURL=eip2718.js.map