"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.FeeMarketEIP1559Transaction = void 0;
const ethereumjs_rlp_1 = require("@nomicfoundation/ethereumjs-rlp");
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const baseTransaction_js_1 = require("./baseTransaction.js");
const EIP1559 = require("./capabilities/eip1559.js");
const EIP2718 = require("./capabilities/eip2718.js");
const EIP2930 = require("./capabilities/eip2930.js");
const Legacy = require("./capabilities/legacy.js");
const types_js_1 = require("./types.js");
const util_js_1 = require("./util.js");
/**
 * Typed transaction with a new gas fee market mechanism
 *
 * - TransactionType: 2
 * - EIP: [EIP-1559](https://eips.ethereum.org/EIPS/eip-1559)
 */
class FeeMarketEIP1559Transaction extends baseTransaction_js_1.BaseTransaction {
    /**
     * This constructor takes the values, validates them, assigns them and freezes the object.
     *
     * It is not recommended to use this constructor directly. Instead use
     * the static factory methods to assist in creating a Transaction object from
     * varying data types.
     */
    constructor(txData, opts = {}) {
        super({ ...txData, type: types_js_1.TransactionType.FeeMarketEIP1559 }, opts);
        const { chainId, accessList, maxFeePerGas, maxPriorityFeePerGas } = txData;
        this.common = this._getCommon(opts.common, chainId);
        this.chainId = this.common.chainId();
        if (this.common.isActivatedEIP(1559) === false) {
            throw new Error('EIP-1559 not enabled on Common');
        }
        this.activeCapabilities = this.activeCapabilities.concat([1559, 2718, 2930]);
        // Populate the access list fields
        const accessListData = util_js_1.AccessLists.getAccessListData(accessList ?? []);
        this.accessList = accessListData.accessList;
        this.AccessListJSON = accessListData.AccessListJSON;
        // Verify the access list format.
        util_js_1.AccessLists.verifyAccessList(this.accessList);
        this.maxFeePerGas = (0, ethereumjs_util_1.bytesToBigInt)((0, ethereumjs_util_1.toBytes)(maxFeePerGas === '' ? '0x' : maxFeePerGas));
        this.maxPriorityFeePerGas = (0, ethereumjs_util_1.bytesToBigInt)((0, ethereumjs_util_1.toBytes)(maxPriorityFeePerGas === '' ? '0x' : maxPriorityFeePerGas));
        this._validateCannotExceedMaxInteger({
            maxFeePerGas: this.maxFeePerGas,
            maxPriorityFeePerGas: this.maxPriorityFeePerGas,
        });
        baseTransaction_js_1.BaseTransaction._validateNotArray(txData);
        if (this.gasLimit * this.maxFeePerGas > ethereumjs_util_1.MAX_INTEGER) {
            const msg = this._errorMsg('gasLimit * maxFeePerGas cannot exceed MAX_INTEGER (2^256-1)');
            throw new Error(msg);
        }
        if (this.maxFeePerGas < this.maxPriorityFeePerGas) {
            const msg = this._errorMsg('maxFeePerGas cannot be less than maxPriorityFeePerGas (The total must be the larger of the two)');
            throw new Error(msg);
        }
        EIP2718.validateYParity(this);
        Legacy.validateHighS(this);
        const freeze = opts?.freeze ?? true;
        if (freeze) {
            Object.freeze(this);
        }
    }
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
        if ((0, ethereumjs_util_1.equalsBytes)(serialized.subarray(0, 1), (0, util_js_1.txTypeBytes)(types_js_1.TransactionType.FeeMarketEIP1559)) ===
            false) {
            throw new Error(`Invalid serialized tx input: not an EIP-1559 transaction (wrong tx type, expected: ${types_js_1.TransactionType.FeeMarketEIP1559}, received: ${(0, ethereumjs_util_1.bytesToHex)(serialized.subarray(0, 1))}`);
        }
        const values = ethereumjs_rlp_1.RLP.decode(serialized.subarray(1));
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized tx input: must be array');
        }
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
        (0, ethereumjs_util_1.validateNoLeadingZeroes)({ nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, value, v, r, s });
        return new FeeMarketEIP1559Transaction({
            chainId: (0, ethereumjs_util_1.bytesToBigInt)(chainId),
            nonce,
            maxPriorityFeePerGas,
            maxFeePerGas,
            gasLimit,
            to,
            value,
            data,
            accessList: accessList ?? [],
            v: v !== undefined ? (0, ethereumjs_util_1.bytesToBigInt)(v) : undefined,
            r,
            s,
        }, opts);
    }
    /**
     * The amount of gas paid for the data in this tx
     */
    getDataFee() {
        return EIP2930.getDataFee(this);
    }
    /**
     * The up front amount that an account must have for this transaction to be valid
     * @param baseFee The base fee of the block (will be set to 0 if not provided)
     */
    getUpfrontCost(baseFee = ethereumjs_util_1.BIGINT_0) {
        return EIP1559.getUpfrontCost(this, baseFee);
    }
    /**
     * Returns a Uint8Array Array of the raw Bytes of the EIP-1559 transaction, in order.
     *
     * Format: `[chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data,
     * accessList, signatureYParity, signatureR, signatureS]`
     *
     * Use {@link FeeMarketEIP1559Transaction.serialize} to add a transaction to a block
     * with {@link Block.fromValuesArray}.
     *
     * For an unsigned tx this method uses the empty Bytes values for the
     * signature parameters `v`, `r` and `s` for encoding. For an EIP-155 compliant
     * representation for external signing use {@link FeeMarketEIP1559Transaction.getMessageToSign}.
     */
    raw() {
        return [
            (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.chainId),
            (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.nonce),
            (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.maxPriorityFeePerGas),
            (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.maxFeePerGas),
            (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.gasLimit),
            this.to !== undefined ? this.to.bytes : new Uint8Array(0),
            (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.value),
            this.data,
            this.accessList,
            this.v !== undefined ? (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.v) : new Uint8Array(0),
            this.r !== undefined ? (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.r) : new Uint8Array(0),
            this.s !== undefined ? (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.s) : new Uint8Array(0),
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
        return EIP2718.serialize(this);
    }
    /**
     * Returns the raw serialized unsigned tx, which can be used
     * to sign the transaction (e.g. for sending to a hardware wallet).
     *
     * Note: in contrast to the legacy tx the raw message format is already
     * serialized and doesn't need to be RLP encoded any more.
     *
     * ```javascript
     * const serializedMessage = tx.getMessageToSign() // use this for the HW wallet input
     * ```
     */
    getMessageToSign() {
        return EIP2718.serialize(this, this.raw().slice(0, 9));
    }
    /**
     * Returns the hashed serialized unsigned tx, which can be used
     * to sign the transaction (e.g. for sending to a hardware wallet).
     *
     * Note: in contrast to the legacy tx the raw message format is already
     * serialized and doesn't need to be RLP encoded any more.
     */
    getHashedMessageToSign() {
        return EIP2718.getHashedMessageToSign(this);
    }
    /**
     * Computes a sha3-256 hash of the serialized tx.
     *
     * This method can only be used for signed txs (it throws otherwise).
     * Use {@link FeeMarketEIP1559Transaction.getMessageToSign} to get a tx hash for the purpose of signing.
     */
    hash() {
        return Legacy.hash(this);
    }
    /**
     * Computes a sha3-256 hash which can be used to verify the signature
     */
    getMessageToVerifySignature() {
        return this.getHashedMessageToSign();
    }
    /**
     * Returns the public key of the sender
     */
    _getSenderPublicKey() {
        return Legacy.getSenderPublicKey(this);
    }
    _processSignature(v, r, s) {
        const opts = { ...this.txOptions, common: this.common };
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
            v: v - ethereumjs_util_1.BIGINT_27,
            r: (0, ethereumjs_util_1.bytesToBigInt)(r),
            s: (0, ethereumjs_util_1.bytesToBigInt)(s),
        }, opts);
    }
    /**
     * Returns an object with the JSON representation of the transaction
     */
    toJSON() {
        const accessListJSON = util_js_1.AccessLists.getAccessListJSON(this.accessList);
        const baseJson = super.toJSON();
        return {
            ...baseJson,
            chainId: (0, ethereumjs_util_1.bigIntToHex)(this.chainId),
            maxPriorityFeePerGas: (0, ethereumjs_util_1.bigIntToHex)(this.maxPriorityFeePerGas),
            maxFeePerGas: (0, ethereumjs_util_1.bigIntToHex)(this.maxFeePerGas),
            accessList: accessListJSON,
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
        return Legacy.errorMsg(this, msg);
    }
}
exports.FeeMarketEIP1559Transaction = FeeMarketEIP1559Transaction;
//# sourceMappingURL=eip1559Transaction.js.map