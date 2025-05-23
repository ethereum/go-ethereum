/*
This file is part of web3.js.

web3.js is free software: you can redistribute it and/or modify
it under the terms of the GNU Lesser General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

web3.js is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Lesser General Public License for more details.

You should have received a copy of the GNU Lesser General Public License
along with web3.js.  If not, see <http://www.gnu.org/licenses/>.
*/
import { RLP } from '@ethereumjs/rlp';
import { keccak256 } from 'ethereum-cryptography/keccak.js';
import { bytesToHex } from 'web3-utils';
import { validateNoLeadingZeroes } from 'web3-validator';
import { bigIntToHex, bigIntToUnpaddedUint8Array, ecrecover, toUint8Array, uint8ArrayToBigInt, unpadUint8Array, } from '../common/utils.js';
import { MAX_INTEGER } from './constants.js';
import { BaseTransaction } from './baseTransaction.js';
import { Capability } from './types.js';
const TRANSACTION_TYPE = 0;
function meetsEIP155(_v, chainId) {
    const v = Number(_v);
    const chainIdDoubled = Number(chainId) * 2;
    return v === chainIdDoubled + 35 || v === chainIdDoubled + 36;
}
/**
 * An Ethereum non-typed (legacy) transaction
 */
// eslint-disable-next-line no-use-before-define
export class Transaction extends BaseTransaction {
    /**
     * Instantiate a transaction from a data dictionary.
     *
     * Format: { nonce, gasPrice, gasLimit, to, value, data, v, r, s }
     *
     * Notes:
     * - All parameters are optional and have some basic default values
     */
    static fromTxData(txData, opts = {}) {
        return new Transaction(txData, opts);
    }
    /**
     * Instantiate a transaction from the serialized tx.
     *
     * Format: `rlp([nonce, gasPrice, gasLimit, to, value, data, v, r, s])`
     */
    static fromSerializedTx(serialized, opts = {}) {
        const values = RLP.decode(serialized);
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized tx input. Must be array');
        }
        return this.fromValuesArray(values, opts);
    }
    /**
     * Create a transaction from a values array.
     *
     * Format: `[nonce, gasPrice, gasLimit, to, value, data, v, r, s]`
     */
    static fromValuesArray(values, opts = {}) {
        // If length is not 6, it has length 9. If v/r/s are empty Uint8Array, it is still an unsigned transaction
        // This happens if you get the RLP data from `raw()`
        if (values.length !== 6 && values.length !== 9) {
            throw new Error('Invalid transaction. Only expecting 6 values (for unsigned tx) or 9 values (for signed tx).');
        }
        const [nonce, gasPrice, gasLimit, to, value, data, v, r, s] = values;
        validateNoLeadingZeroes({ nonce, gasPrice, gasLimit, value, v, r, s });
        return new Transaction({
            nonce,
            gasPrice,
            gasLimit,
            to,
            value,
            data,
            v,
            r,
            s,
        }, opts);
    }
    /**
     * This constructor takes the values, validates them, assigns them and freezes the object.
     *
     * It is not recommended to use this constructor directly. Instead use
     * the static factory methods to assist in creating a Transaction object from
     * varying data types.
     */
    constructor(txData, opts = {}) {
        var _a;
        super(Object.assign(Object.assign({}, txData), { type: TRANSACTION_TYPE }), opts);
        this.common = this._validateTxV(this.v, opts.common);
        this.gasPrice = uint8ArrayToBigInt(toUint8Array(txData.gasPrice === '' ? '0x' : txData.gasPrice));
        if (this.gasPrice * this.gasLimit > MAX_INTEGER) {
            const msg = this._errorMsg('gas limit * gasPrice cannot exceed MAX_INTEGER (2^256-1)');
            throw new Error(msg);
        }
        this._validateCannotExceedMaxInteger({ gasPrice: this.gasPrice });
        BaseTransaction._validateNotArray(txData);
        if (this.common.gteHardfork('spuriousDragon')) {
            if (!this.isSigned()) {
                this.activeCapabilities.push(Capability.EIP155ReplayProtection);
            }
            else {
                // EIP155 spec:
                // If block.number >= 2,675,000 and v = CHAIN_ID * 2 + 35 or v = CHAIN_ID * 2 + 36
                // then when computing the hash of a transaction for purposes of signing or recovering
                // instead of hashing only the first six elements (i.e. nonce, gasprice, startgas, to, value, data)
                // hash nine elements, with v replaced by CHAIN_ID, r = 0 and s = 0.
                // v and chain ID meet EIP-155 conditions
                // eslint-disable-next-line no-lonely-if
                if (meetsEIP155(this.v, this.common.chainId())) {
                    this.activeCapabilities.push(Capability.EIP155ReplayProtection);
                }
            }
        }
        const freeze = (_a = opts === null || opts === void 0 ? void 0 : opts.freeze) !== null && _a !== void 0 ? _a : true;
        if (freeze) {
            Object.freeze(this);
        }
    }
    /**
     * Returns a Uint8Array Array of the raw Uint8Arrays of the legacy transaction, in order.
     *
     * Format: `[nonce, gasPrice, gasLimit, to, value, data, v, r, s]`
     *
     * For legacy txs this is also the correct format to add transactions
     * to a block with {@link Block.fromValuesArray} (use the `serialize()` method
     * for typed txs).
     *
     * For an unsigned tx this method returns the empty Uint8Array values
     * for the signature parameters `v`, `r` and `s`. For an EIP-155 compliant
     * representation have a look at {@link Transaction.getMessageToSign}.
     */
    raw() {
        return [
            bigIntToUnpaddedUint8Array(this.nonce),
            bigIntToUnpaddedUint8Array(this.gasPrice),
            bigIntToUnpaddedUint8Array(this.gasLimit),
            this.to !== undefined ? this.to.buf : Uint8Array.from([]),
            bigIntToUnpaddedUint8Array(this.value),
            this.data,
            this.v !== undefined ? bigIntToUnpaddedUint8Array(this.v) : Uint8Array.from([]),
            this.r !== undefined ? bigIntToUnpaddedUint8Array(this.r) : Uint8Array.from([]),
            this.s !== undefined ? bigIntToUnpaddedUint8Array(this.s) : Uint8Array.from([]),
        ];
    }
    /**
     * Returns the serialized encoding of the legacy transaction.
     *
     * Format: `rlp([nonce, gasPrice, gasLimit, to, value, data, v, r, s])`
     *
     * For an unsigned tx this method uses the empty Uint8Array values for the
     * signature parameters `v`, `r` and `s` for encoding. For an EIP-155 compliant
     * representation for external signing use {@link Transaction.getMessageToSign}.
     */
    serialize() {
        return RLP.encode(this.raw());
    }
    _getMessageToSign() {
        const values = [
            bigIntToUnpaddedUint8Array(this.nonce),
            bigIntToUnpaddedUint8Array(this.gasPrice),
            bigIntToUnpaddedUint8Array(this.gasLimit),
            this.to !== undefined ? this.to.buf : Uint8Array.from([]),
            bigIntToUnpaddedUint8Array(this.value),
            this.data,
        ];
        if (this.supports(Capability.EIP155ReplayProtection)) {
            values.push(toUint8Array(this.common.chainId()));
            values.push(unpadUint8Array(toUint8Array(0)));
            values.push(unpadUint8Array(toUint8Array(0)));
        }
        return values;
    }
    getMessageToSign(hashMessage = true) {
        const message = this._getMessageToSign();
        if (hashMessage) {
            return keccak256(RLP.encode(message));
        }
        return message;
    }
    /**
     * The amount of gas paid for the data in this tx
     */
    getDataFee() {
        if (this.cache.dataFee && this.cache.dataFee.hardfork === this.common.hardfork()) {
            return this.cache.dataFee.value;
        }
        if (Object.isFrozen(this)) {
            this.cache.dataFee = {
                value: super.getDataFee(),
                hardfork: this.common.hardfork(),
            };
        }
        return super.getDataFee();
    }
    /**
     * The up front amount that an account must have for this transaction to be valid
     */
    getUpfrontCost() {
        return this.gasLimit * this.gasPrice + this.value;
    }
    /**
     * Computes a sha3-256 hash of the serialized tx.
     *
     * This method can only be used for signed txs (it throws otherwise).
     * Use {@link Transaction.getMessageToSign} to get a tx hash for the purpose of signing.
     */
    hash() {
        if (!this.isSigned()) {
            const msg = this._errorMsg('Cannot call hash method if transaction is not signed');
            throw new Error(msg);
        }
        if (Object.isFrozen(this)) {
            if (!this.cache.hash) {
                this.cache.hash = keccak256(RLP.encode(this.raw()));
            }
            return this.cache.hash;
        }
        return keccak256(RLP.encode(this.raw()));
    }
    /**
     * Computes a sha3-256 hash which can be used to verify the signature
     */
    getMessageToVerifySignature() {
        if (!this.isSigned()) {
            const msg = this._errorMsg('This transaction is not signed');
            throw new Error(msg);
        }
        const message = this._getMessageToSign();
        return keccak256(RLP.encode(message));
    }
    /**
     * Returns the public key of the sender
     */
    getSenderPublicKey() {
        const msgHash = this.getMessageToVerifySignature();
        const { v, r, s } = this;
        this._validateHighS();
        try {
            return ecrecover(msgHash, v, bigIntToUnpaddedUint8Array(r), bigIntToUnpaddedUint8Array(s), this.supports(Capability.EIP155ReplayProtection)
                ? this.common.chainId()
                : undefined);
        }
        catch (e) {
            const msg = this._errorMsg('Invalid Signature');
            throw new Error(msg);
        }
    }
    /**
     * Process the v, r, s values from the `sign` method of the base transaction.
     */
    _processSignature(_v, r, s) {
        let v = _v;
        if (this.supports(Capability.EIP155ReplayProtection)) {
            v += this.common.chainId() * BigInt(2) + BigInt(8);
        }
        const opts = Object.assign(Object.assign({}, this.txOptions), { common: this.common });
        return Transaction.fromTxData({
            nonce: this.nonce,
            gasPrice: this.gasPrice,
            gasLimit: this.gasLimit,
            to: this.to,
            value: this.value,
            data: this.data,
            v,
            r: uint8ArrayToBigInt(r),
            s: uint8ArrayToBigInt(s),
        }, opts);
    }
    /**
     * Returns an object with the JSON representation of the transaction.
     */
    toJSON() {
        return {
            nonce: bigIntToHex(this.nonce),
            gasPrice: bigIntToHex(this.gasPrice),
            gasLimit: bigIntToHex(this.gasLimit),
            to: this.to !== undefined ? this.to.toString() : undefined,
            value: bigIntToHex(this.value),
            data: bytesToHex(this.data),
            v: this.v !== undefined ? bigIntToHex(this.v) : undefined,
            r: this.r !== undefined ? bigIntToHex(this.r) : undefined,
            s: this.s !== undefined ? bigIntToHex(this.s) : undefined,
        };
    }
    /**
     * Validates tx's `v` value
     */
    _validateTxV(_v, common) {
        let chainIdBigInt;
        const v = _v !== undefined ? Number(_v) : undefined;
        // Check for valid v values in the scope of a signed legacy tx
        if (v !== undefined) {
            // v is 1. not matching the EIP-155 chainId included case and...
            // v is 2. not matching the classic v=27 or v=28 case
            if (v < 37 && v !== 27 && v !== 28) {
                throw new Error(`Legacy txs need either v = 27/28 or v >= 37 (EIP-155 replay protection), got v = ${v}`);
            }
        }
        // No unsigned tx and EIP-155 activated and chain ID included
        if (v !== undefined &&
            v !== 0 &&
            (!common || common.gteHardfork('spuriousDragon')) &&
            v !== 27 &&
            v !== 28) {
            if (common) {
                if (!meetsEIP155(BigInt(v), common.chainId())) {
                    throw new Error(`Incompatible EIP155-based V ${v} and chain id ${common.chainId()}. See the Common parameter of the Transaction constructor to set the chain id.`);
                }
            }
            else {
                // Derive the original chain ID
                let numSub;
                if ((v - 35) % 2 === 0) {
                    numSub = 35;
                }
                else {
                    numSub = 36;
                }
                // Use derived chain ID to create a proper Common
                chainIdBigInt = BigInt(v - numSub) / BigInt(2);
            }
        }
        return this._getCommon(common, chainIdBigInt);
    }
    /**
     * Return a compact error string representation of the object
     */
    errorStr() {
        let errorStr = this._getSharedErrorPostfix();
        errorStr += ` gasPrice=${this.gasPrice}`;
        return errorStr;
    }
    /**
     * Internal helper function to create an annotated error message
     *
     * @param msg Base error message
     * @hidden
     */
    _errorMsg(msg) {
        return `${msg} (${this.errorStr()})`;
    }
}
//# sourceMappingURL=legacyTransaction.js.map