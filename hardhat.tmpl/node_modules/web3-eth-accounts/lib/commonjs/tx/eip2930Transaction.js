"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.AccessListEIP2930Transaction = void 0;
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
const keccak_js_1 = require("ethereum-cryptography/keccak.js");
const web3_validator_1 = require("web3-validator");
const rlp_1 = require("@ethereumjs/rlp");
const web3_utils_1 = require("web3-utils");
const constants_js_1 = require("./constants.js");
const utils_js_1 = require("./utils.js");
const utils_js_2 = require("../common/utils.js");
const baseTransaction_js_1 = require("./baseTransaction.js");
const TRANSACTION_TYPE = 1;
const TRANSACTION_TYPE_UINT8ARRAY = (0, web3_utils_1.hexToBytes)(TRANSACTION_TYPE.toString(16).padStart(2, '0'));
/**
 * Typed transaction with optional access lists
 *
 * - TransactionType: 1
 * - EIP: [EIP-2930](https://eips.ethereum.org/EIPS/eip-2930)
 */
// eslint-disable-next-line no-use-before-define
class AccessListEIP2930Transaction extends baseTransaction_js_1.BaseTransaction {
    /**
     * Instantiate a transaction from a data dictionary.
     *
     * Format: { chainId, nonce, gasPrice, gasLimit, to, value, data, accessList,
     * v, r, s }
     *
     * Notes:
     * - `chainId` will be set automatically if not provided
     * - All parameters are optional and have some basic default values
     */
    static fromTxData(txData, opts = {}) {
        return new AccessListEIP2930Transaction(txData, opts);
    }
    /**
     * Instantiate a transaction from the serialized tx.
     *
     * Format: `0x01 || rlp([chainId, nonce, gasPrice, gasLimit, to, value, data, accessList,
     * signatureYParity (v), signatureR (r), signatureS (s)])`
     */
    static fromSerializedTx(serialized, opts = {}) {
        if (!(0, web3_utils_1.uint8ArrayEquals)(serialized.subarray(0, 1), TRANSACTION_TYPE_UINT8ARRAY)) {
            throw new Error(`Invalid serialized tx input: not an EIP-2930 transaction (wrong tx type, expected: ${TRANSACTION_TYPE}, received: ${(0, web3_utils_1.bytesToHex)(serialized.subarray(0, 1))}`);
        }
        const values = rlp_1.RLP.decode(Uint8Array.from(serialized.subarray(1)));
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized tx input: must be array');
        }
        // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
        return AccessListEIP2930Transaction.fromValuesArray(values, opts);
    }
    /**
     * Create a transaction from a values array.
     *
     * Format: `[chainId, nonce, gasPrice, gasLimit, to, value, data, accessList,
     * signatureYParity (v), signatureR (r), signatureS (s)]`
     */
    static fromValuesArray(values, opts = {}) {
        if (values.length !== 8 && values.length !== 11) {
            throw new Error('Invalid EIP-2930 transaction. Only expecting 8 values (for unsigned tx) or 11 values (for signed tx).');
        }
        const [chainId, nonce, gasPrice, gasLimit, to, value, data, accessList, v, r, s] = values;
        this._validateNotArray({ chainId, v });
        (0, web3_validator_1.validateNoLeadingZeroes)({ nonce, gasPrice, gasLimit, value, v, r, s });
        const emptyAccessList = [];
        return new AccessListEIP2930Transaction({
            chainId: (0, utils_js_2.uint8ArrayToBigInt)(chainId),
            nonce,
            gasPrice,
            gasLimit,
            to,
            value,
            data,
            accessList: accessList !== null && accessList !== void 0 ? accessList : emptyAccessList,
            v: v !== undefined ? (0, utils_js_2.uint8ArrayToBigInt)(v) : undefined, // EIP2930 supports v's with value 0 (empty Uint8Array)
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
        /**
         * The default HF if the tx type is active on that HF
         * or the first greater HF where the tx is active.
         *
         * @hidden
         */
        this.DEFAULT_HARDFORK = 'berlin';
        const { chainId, accessList, gasPrice } = txData;
        this.common = this._getCommon(opts.common, chainId);
        this.chainId = this.common.chainId();
        // EIP-2718 check is done in Common
        if (!this.common.isActivatedEIP(2930)) {
            throw new Error('EIP-2930 not enabled on Common');
        }
        this.activeCapabilities = this.activeCapabilities.concat([2718, 2930]);
        // Populate the access list fields
        const accessListData = (0, utils_js_1.getAccessListData)(accessList !== null && accessList !== void 0 ? accessList : []);
        this.accessList = accessListData.accessList;
        this.AccessListJSON = accessListData.AccessListJSON;
        // Verify the access list format.
        (0, utils_js_1.verifyAccessList)(this.accessList);
        this.gasPrice = (0, utils_js_2.uint8ArrayToBigInt)((0, utils_js_2.toUint8Array)(gasPrice === '' ? '0x' : gasPrice));
        this._validateCannotExceedMaxInteger({
            gasPrice: this.gasPrice,
        });
        baseTransaction_js_1.BaseTransaction._validateNotArray(txData);
        if (this.gasPrice * this.gasLimit > constants_js_1.MAX_INTEGER) {
            const msg = this._errorMsg('gasLimit * gasPrice cannot exceed MAX_INTEGER');
            throw new Error(msg);
        }
        this._validateYParity();
        this._validateHighS();
        const freeze = (_a = opts === null || opts === void 0 ? void 0 : opts.freeze) !== null && _a !== void 0 ? _a : true;
        if (freeze) {
            Object.freeze(this);
        }
    }
    /**
     * The amount of gas paid for the data in this tx
     */
    getDataFee() {
        if (this.cache.dataFee && this.cache.dataFee.hardfork === this.common.hardfork()) {
            return this.cache.dataFee.value;
        }
        let cost = super.getDataFee();
        cost += BigInt((0, utils_js_1.getDataFeeEIP2930)(this.accessList, this.common));
        if (Object.isFrozen(this)) {
            this.cache.dataFee = {
                value: cost,
                hardfork: this.common.hardfork(),
            };
        }
        return cost;
    }
    /**
     * The up front amount that an account must have for this transaction to be valid
     */
    getUpfrontCost() {
        return this.gasLimit * this.gasPrice + this.value;
    }
    /**
     * Returns a Uint8Array Array of the raw Uint8Arrays of the EIP-2930 transaction, in order.
     *
     * Format: `[chainId, nonce, gasPrice, gasLimit, to, value, data, accessList,
     * signatureYParity (v), signatureR (r), signatureS (s)]`
     *
     * Use {@link AccessListEIP2930Transaction.serialize} to add a transaction to a block
     * with {@link Block.fromValuesArray}.
     *
     * For an unsigned tx this method uses the empty UINT8ARRAY values for the
     * signature parameters `v`, `r` and `s` for encoding. For an EIP-155 compliant
     * representation for external signing use {@link AccessListEIP2930Transaction.getMessageToSign}.
     */
    raw() {
        return [
            (0, utils_js_2.bigIntToUnpaddedUint8Array)(this.chainId),
            (0, utils_js_2.bigIntToUnpaddedUint8Array)(this.nonce),
            (0, utils_js_2.bigIntToUnpaddedUint8Array)(this.gasPrice),
            (0, utils_js_2.bigIntToUnpaddedUint8Array)(this.gasLimit),
            this.to !== undefined ? this.to.buf : Uint8Array.from([]),
            (0, utils_js_2.bigIntToUnpaddedUint8Array)(this.value),
            this.data,
            this.accessList,
            this.v !== undefined ? (0, utils_js_2.bigIntToUnpaddedUint8Array)(this.v) : Uint8Array.from([]),
            this.r !== undefined ? (0, utils_js_2.bigIntToUnpaddedUint8Array)(this.r) : Uint8Array.from([]),
            this.s !== undefined ? (0, utils_js_2.bigIntToUnpaddedUint8Array)(this.s) : Uint8Array.from([]),
        ];
    }
    /**
     * Returns the serialized encoding of the EIP-2930 transaction.
     *
     * Format: `0x01 || rlp([chainId, nonce, gasPrice, gasLimit, to, value, data, accessList,
     * signatureYParity (v), signatureR (r), signatureS (s)])`
     *
     * Note that in contrast to the legacy tx serialization format this is not
     * valid RLP any more due to the raw tx type preceding and concatenated to
     * the RLP encoding of the values.
     */
    serialize() {
        const base = this.raw();
        return (0, web3_utils_1.uint8ArrayConcat)(TRANSACTION_TYPE_UINT8ARRAY, rlp_1.RLP.encode(base));
    }
    /**
     * Returns the serialized unsigned tx (hashed or raw), which can be used
     * to sign the transaction (e.g. for sending to a hardware wallet).
     *
     * Note: in contrast to the legacy tx the raw message format is already
     * serialized and doesn't need to be RLP encoded any more.
     *
     * ```javascript
     * const serializedMessage = tx.getMessageToSign(false) // use this for the HW wallet input
     * ```
     *
     * @param hashMessage - Return hashed message if set to true (default: true)
     */
    getMessageToSign(hashMessage = true) {
        const base = this.raw().slice(0, 8);
        const message = (0, web3_utils_1.uint8ArrayConcat)(TRANSACTION_TYPE_UINT8ARRAY, rlp_1.RLP.encode(base));
        if (hashMessage) {
            return (0, keccak_js_1.keccak256)(message);
        }
        return message;
    }
    /**
     * Computes a sha3-256 hash of the serialized tx.
     *
     * This method can only be used for signed txs (it throws otherwise).
     * Use {@link AccessListEIP2930Transaction.getMessageToSign} to get a tx hash for the purpose of signing.
     */
    hash() {
        if (!this.isSigned()) {
            const msg = this._errorMsg('Cannot call hash method if transaction is not signed');
            throw new Error(msg);
        }
        if (Object.isFrozen(this)) {
            if (!this.cache.hash) {
                this.cache.hash = (0, keccak_js_1.keccak256)(this.serialize());
            }
            return this.cache.hash;
        }
        return (0, keccak_js_1.keccak256)(this.serialize());
    }
    /**
     * Computes a sha3-256 hash which can be used to verify the signature
     */
    getMessageToVerifySignature() {
        return this.getMessageToSign();
    }
    /**
     * Returns the public key of the sender
     */
    getSenderPublicKey() {
        if (!this.isSigned()) {
            const msg = this._errorMsg('Cannot call this method if transaction is not signed');
            throw new Error(msg);
        }
        const msgHash = this.getMessageToVerifySignature();
        const { v, r, s } = this;
        this._validateHighS();
        try {
            return (0, utils_js_2.ecrecover)(msgHash, v + BigInt(27), // Recover the 27 which was stripped from ecsign
            (0, utils_js_2.bigIntToUnpaddedUint8Array)(r), (0, utils_js_2.bigIntToUnpaddedUint8Array)(s));
        }
        catch (e) {
            const msg = this._errorMsg('Invalid Signature');
            throw new Error(msg);
        }
    }
    _processSignature(v, r, s) {
        const opts = Object.assign(Object.assign({}, this.txOptions), { common: this.common });
        return AccessListEIP2930Transaction.fromTxData({
            chainId: this.chainId,
            nonce: this.nonce,
            gasPrice: this.gasPrice,
            gasLimit: this.gasLimit,
            to: this.to,
            value: this.value,
            data: this.data,
            accessList: this.accessList,
            v: v - BigInt(27), // This looks extremely hacky: /util actually adds 27 to the value, the recovery bit is either 0 or 1.
            r: (0, utils_js_2.uint8ArrayToBigInt)(r),
            s: (0, utils_js_2.uint8ArrayToBigInt)(s),
        }, opts);
    }
    /**
     * Returns an object with the JSON representation of the transaction
     */
    toJSON() {
        const accessListJSON = (0, utils_js_1.getAccessListJSON)(this.accessList);
        return {
            chainId: (0, utils_js_2.bigIntToHex)(this.chainId),
            nonce: (0, utils_js_2.bigIntToHex)(this.nonce),
            gasPrice: (0, utils_js_2.bigIntToHex)(this.gasPrice),
            gasLimit: (0, utils_js_2.bigIntToHex)(this.gasLimit),
            to: this.to !== undefined ? this.to.toString() : undefined,
            value: (0, utils_js_2.bigIntToHex)(this.value),
            data: (0, web3_utils_1.bytesToHex)(this.data),
            accessList: accessListJSON,
            v: this.v !== undefined ? (0, utils_js_2.bigIntToHex)(this.v) : undefined,
            r: this.r !== undefined ? (0, utils_js_2.bigIntToHex)(this.r) : undefined,
            s: this.s !== undefined ? (0, utils_js_2.bigIntToHex)(this.s) : undefined,
        };
    }
    /**
     * Return a compact error string representation of the object
     */
    errorStr() {
        var _a, _b;
        let errorStr = this._getSharedErrorPostfix();
        // Keep ? for this.accessList since this otherwise causes Hardhat E2E tests to fail
        errorStr += ` gasPrice=${this.gasPrice} accessListCount=${(_b = (_a = this.accessList) === null || _a === void 0 ? void 0 : _a.length) !== null && _b !== void 0 ? _b : 0}`;
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
exports.AccessListEIP2930Transaction = AccessListEIP2930Transaction;
//# sourceMappingURL=eip2930Transaction.js.map