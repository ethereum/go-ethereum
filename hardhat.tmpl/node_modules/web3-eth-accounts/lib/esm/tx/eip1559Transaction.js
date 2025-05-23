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
import { keccak256 } from 'ethereum-cryptography/keccak.js';
import { validateNoLeadingZeroes } from 'web3-validator';
import { RLP } from '@ethereumjs/rlp';
import { bytesToHex, hexToBytes, uint8ArrayConcat, uint8ArrayEquals } from 'web3-utils';
import { MAX_INTEGER } from './constants.js';
import { BaseTransaction } from './baseTransaction.js';
import { getAccessListData, getAccessListJSON, getDataFeeEIP2930, verifyAccessList, } from './utils.js';
import { bigIntToHex, toUint8Array, ecrecover, uint8ArrayToBigInt, bigIntToUnpaddedUint8Array, } from '../common/utils.js';
const TRANSACTION_TYPE = 2;
const TRANSACTION_TYPE_UINT8ARRAY = hexToBytes(TRANSACTION_TYPE.toString(16).padStart(2, '0'));
/**
 * Typed transaction with a new gas fee market mechanism
 *
 * - TransactionType: 2
 * - EIP: [EIP-1559](https://eips.ethereum.org/EIPS/eip-1559)
 */
// eslint-disable-next-line no-use-before-define
export class FeeMarketEIP1559Transaction extends BaseTransaction {
    /**
     * Instantiate a transaction from a data dictionary.
     *
     * Format: { chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data,
     * accessList, v, r, s }
     *
     * Notes:
     * - `chainId` will be set automatically if not provided
     * - All parameters are optional and have some basic default values
     */
    static fromTxData(txData, opts = {}) {
        return new FeeMarketEIP1559Transaction(txData, opts);
    }
    /**
     * Instantiate a transaction from the serialized tx.
     *
     * Format: `0x02 || rlp([chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data,
     * accessList, signatureYParity, signatureR, signatureS])`
     */
    static fromSerializedTx(serialized, opts = {}) {
        if (!uint8ArrayEquals(serialized.subarray(0, 1), TRANSACTION_TYPE_UINT8ARRAY)) {
            throw new Error(`Invalid serialized tx input: not an EIP-1559 transaction (wrong tx type, expected: ${TRANSACTION_TYPE}, received: ${bytesToHex(serialized.subarray(0, 1))}`);
        }
        const values = RLP.decode(serialized.subarray(1));
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized tx input: must be array');
        }
        // eslint-disable-next-line @typescript-eslint/no-unsafe-argument
        return FeeMarketEIP1559Transaction.fromValuesArray(values, opts);
    }
    /**
     * Create a transaction from a values array.
     *
     * Format: `[chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data,
     * accessList, signatureYParity, signatureR, signatureS]`
     */
    static fromValuesArray(values, opts = {}) {
        if (values.length !== 9 && values.length !== 12) {
            throw new Error('Invalid EIP-1559 transaction. Only expecting 9 values (for unsigned tx) or 12 values (for signed tx).');
        }
        const [chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data, accessList, v, r, s,] = values;
        this._validateNotArray({ chainId, v });
        validateNoLeadingZeroes({
            nonce,
            maxPriorityFeePerGas,
            maxFeePerGas,
            gasLimit,
            value,
            v,
            r,
            s,
        });
        return new FeeMarketEIP1559Transaction({
            chainId: uint8ArrayToBigInt(chainId),
            nonce,
            maxPriorityFeePerGas,
            maxFeePerGas,
            gasLimit,
            to,
            value,
            data,
            accessList: accessList !== null && accessList !== void 0 ? accessList : [],
            v: v !== undefined ? uint8ArrayToBigInt(v) : undefined, // EIP2930 supports v's with value 0 (empty Uint8Array)
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
        this.DEFAULT_HARDFORK = 'london';
        const { chainId, accessList, maxFeePerGas, maxPriorityFeePerGas } = txData;
        this.common = this._getCommon(opts.common, chainId);
        this.chainId = this.common.chainId();
        if (!this.common.isActivatedEIP(1559)) {
            throw new Error('EIP-1559 not enabled on Common');
        }
        this.activeCapabilities = this.activeCapabilities.concat([1559, 2718, 2930]);
        // Populate the access list fields
        const accessListData = getAccessListData(accessList !== null && accessList !== void 0 ? accessList : []);
        this.accessList = accessListData.accessList;
        this.AccessListJSON = accessListData.AccessListJSON;
        // Verify the access list format.
        verifyAccessList(this.accessList);
        this.maxFeePerGas = uint8ArrayToBigInt(toUint8Array(maxFeePerGas === '' ? '0x' : maxFeePerGas));
        this.maxPriorityFeePerGas = uint8ArrayToBigInt(toUint8Array(maxPriorityFeePerGas === '' ? '0x' : maxPriorityFeePerGas));
        this._validateCannotExceedMaxInteger({
            maxFeePerGas: this.maxFeePerGas,
            maxPriorityFeePerGas: this.maxPriorityFeePerGas,
        });
        BaseTransaction._validateNotArray(txData);
        if (this.gasLimit * this.maxFeePerGas > MAX_INTEGER) {
            const msg = this._errorMsg('gasLimit * maxFeePerGas cannot exceed MAX_INTEGER (2^256-1)');
            throw new Error(msg);
        }
        if (this.maxFeePerGas < this.maxPriorityFeePerGas) {
            const msg = this._errorMsg('maxFeePerGas cannot be less than maxPriorityFeePerGas (The total must be the larger of the two)');
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
        cost += BigInt(getDataFeeEIP2930(this.accessList, this.common));
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
     * @param baseFee The base fee of the block (will be set to 0 if not provided)
     */
    getUpfrontCost(baseFee = BigInt(0)) {
        const prio = this.maxPriorityFeePerGas;
        const maxBase = this.maxFeePerGas - baseFee;
        const inclusionFeePerGas = prio < maxBase ? prio : maxBase;
        const gasPrice = inclusionFeePerGas + baseFee;
        return this.gasLimit * gasPrice + this.value;
    }
    /**
     * Returns a Uint8Array Array of the raw Uint8Arrays of the EIP-1559 transaction, in order.
     *
     * Format: `[chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data,
     * accessList, signatureYParity, signatureR, signatureS]`
     *
     * Use {@link FeeMarketEIP1559Transaction.serialize} to add a transaction to a block
     * with {@link Block.fromValuesArray}.
     *
     * For an unsigned tx this method uses the empty Uint8Array values for the
     * signature parameters `v`, `r` and `s` for encoding. For an EIP-155 compliant
     * representation for external signing use {@link FeeMarketEIP1559Transaction.getMessageToSign}.
     */
    raw() {
        return [
            bigIntToUnpaddedUint8Array(this.chainId),
            bigIntToUnpaddedUint8Array(this.nonce),
            bigIntToUnpaddedUint8Array(this.maxPriorityFeePerGas),
            bigIntToUnpaddedUint8Array(this.maxFeePerGas),
            bigIntToUnpaddedUint8Array(this.gasLimit),
            this.to !== undefined ? this.to.buf : Uint8Array.from([]),
            bigIntToUnpaddedUint8Array(this.value),
            this.data,
            this.accessList,
            this.v !== undefined ? bigIntToUnpaddedUint8Array(this.v) : Uint8Array.from([]),
            this.r !== undefined ? bigIntToUnpaddedUint8Array(this.r) : Uint8Array.from([]),
            this.s !== undefined ? bigIntToUnpaddedUint8Array(this.s) : Uint8Array.from([]),
        ];
    }
    /**
     * Returns the serialized encoding of the EIP-1559 transaction.
     *
     * Format: `0x02 || rlp([chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data,
     * accessList, signatureYParity, signatureR, signatureS])`
     *
     * Note that in contrast to the legacy tx serialization format this is not
     * valid RLP any more due to the raw tx type preceding and concatenated to
     * the RLP encoding of the values.
     */
    serialize() {
        const base = this.raw();
        return uint8ArrayConcat(TRANSACTION_TYPE_UINT8ARRAY, RLP.encode(base));
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
        const base = this.raw().slice(0, 9);
        const message = uint8ArrayConcat(TRANSACTION_TYPE_UINT8ARRAY, RLP.encode(base));
        if (hashMessage) {
            return keccak256(message);
        }
        return message;
    }
    /**
     * Computes a sha3-256 hash of the serialized tx.
     *
     * This method can only be used for signed txs (it throws otherwise).
     * Use {@link FeeMarketEIP1559Transaction.getMessageToSign} to get a tx hash for the purpose of signing.
     */
    hash() {
        if (!this.isSigned()) {
            const msg = this._errorMsg('Cannot call hash method if transaction is not signed');
            throw new Error(msg);
        }
        if (Object.isFrozen(this)) {
            if (!this.cache.hash) {
                this.cache.hash = keccak256(this.serialize());
            }
            return this.cache.hash;
        }
        return keccak256(this.serialize());
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
            return ecrecover(msgHash, v + BigInt(27), // Recover the 27 which was stripped from ecsign
            bigIntToUnpaddedUint8Array(r), bigIntToUnpaddedUint8Array(s));
        }
        catch (e) {
            const msg = this._errorMsg('Invalid Signature');
            throw new Error(msg);
        }
    }
    _processSignature(v, r, s) {
        const opts = Object.assign(Object.assign({}, this.txOptions), { common: this.common });
        return FeeMarketEIP1559Transaction.fromTxData({
            chainId: this.chainId,
            nonce: this.nonce,
            maxPriorityFeePerGas: this.maxPriorityFeePerGas,
            maxFeePerGas: this.maxFeePerGas,
            gasLimit: this.gasLimit,
            to: this.to,
            value: this.value,
            data: this.data,
            accessList: this.accessList,
            v: v - BigInt(27), // This looks extremely hacky: /util actually adds 27 to the value, the recovery bit is either 0 or 1.
            r: uint8ArrayToBigInt(r),
            s: uint8ArrayToBigInt(s),
        }, opts);
    }
    /**
     * Returns an object with the JSON representation of the transaction
     */
    toJSON() {
        const accessListJSON = getAccessListJSON(this.accessList);
        return {
            chainId: bigIntToHex(this.chainId),
            nonce: bigIntToHex(this.nonce),
            maxPriorityFeePerGas: bigIntToHex(this.maxPriorityFeePerGas),
            maxFeePerGas: bigIntToHex(this.maxFeePerGas),
            gasLimit: bigIntToHex(this.gasLimit),
            to: this.to !== undefined ? this.to.toString() : undefined,
            value: bigIntToHex(this.value),
            data: bytesToHex(this.data),
            accessList: accessListJSON,
            v: this.v !== undefined ? bigIntToHex(this.v) : undefined,
            r: this.r !== undefined ? bigIntToHex(this.r) : undefined,
            s: this.s !== undefined ? bigIntToHex(this.s) : undefined,
        };
    }
    /**
     * Return a compact error string representation of the object
     */
    errorStr() {
        let errorStr = this._getSharedErrorPostfix();
        errorStr += ` maxFeePerGas=${this.maxFeePerGas} maxPriorityFeePerGas=${this.maxPriorityFeePerGas}`;
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
//# sourceMappingURL=eip1559Transaction.js.map