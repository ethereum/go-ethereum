"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.BlobEIP4844Transaction = void 0;
const ethereumjs_rlp_1 = require("@nomicfoundation/ethereumjs-rlp");
const ethereumjs_util_1 = require("@nomicfoundation/ethereumjs-util");
const baseTransaction_js_1 = require("./baseTransaction.js");
const EIP1559 = require("./capabilities/eip1559.js");
const EIP2718 = require("./capabilities/eip2718.js");
const EIP2930 = require("./capabilities/eip2930.js");
const Legacy = require("./capabilities/legacy.js");
const constants_js_1 = require("./constants.js");
const types_js_1 = require("./types.js");
const util_js_1 = require("./util.js");
const validateBlobTransactionNetworkWrapper = (blobVersionedHashes, blobs, commitments, kzgProofs, version) => {
    if (!(blobVersionedHashes.length === blobs.length && blobs.length === commitments.length)) {
        throw new Error('Number of blobVersionedHashes, blobs, and commitments not all equal');
    }
    if (blobVersionedHashes.length === 0) {
        throw new Error('Invalid transaction with empty blobs');
    }
    let isValid;
    try {
        isValid = ethereumjs_util_1.kzg.verifyBlobKzgProofBatch(blobs, commitments, kzgProofs);
    }
    catch (error) {
        throw new Error(`KZG verification of blobs fail with error=${error}`);
    }
    if (!isValid) {
        throw new Error('KZG proof cannot be verified from blobs/commitments');
    }
    for (let x = 0; x < blobVersionedHashes.length; x++) {
        const computedVersionedHash = (0, ethereumjs_util_1.computeVersionedHash)(commitments[x], version);
        if (!(0, ethereumjs_util_1.equalsBytes)(computedVersionedHash, blobVersionedHashes[x])) {
            throw new Error(`commitment for blob at index ${x} does not match versionedHash`);
        }
    }
};
/**
 * Typed transaction with a new gas fee market mechanism for transactions that include "blobs" of data
 *
 * - TransactionType: 3
 * - EIP: [EIP-4844](https://eips.ethereum.org/EIPS/eip-4844)
 */
class BlobEIP4844Transaction extends baseTransaction_js_1.BaseTransaction {
    /**
     * This constructor takes the values, validates them, assigns them and freezes the object.
     *
     * It is not recommended to use this constructor directly. Instead use
     * the static constructors or factory methods to assist in creating a Transaction object from
     * varying data types.
     */
    constructor(txData, opts = {}) {
        super({ ...txData, type: types_js_1.TransactionType.BlobEIP4844 }, opts);
        const { chainId, accessList, maxFeePerGas, maxPriorityFeePerGas, maxFeePerBlobGas } = txData;
        this.common = this._getCommon(opts.common, chainId);
        this.chainId = this.common.chainId();
        if (this.common.isActivatedEIP(1559) === false) {
            throw new Error('EIP-1559 not enabled on Common');
        }
        if (this.common.isActivatedEIP(4844) === false) {
            throw new Error('EIP-4844 not enabled on Common');
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
        this.maxFeePerBlobGas = (0, ethereumjs_util_1.bytesToBigInt)((0, ethereumjs_util_1.toBytes)((maxFeePerBlobGas ?? '') === '' ? '0x' : maxFeePerBlobGas));
        this.blobVersionedHashes = (txData.blobVersionedHashes ?? []).map((vh) => (0, ethereumjs_util_1.toBytes)(vh));
        EIP2718.validateYParity(this);
        Legacy.validateHighS(this);
        for (const hash of this.blobVersionedHashes) {
            if (hash.length !== 32) {
                const msg = this._errorMsg('versioned hash is invalid length');
                throw new Error(msg);
            }
            if (BigInt(hash[0]) !== this.common.param('sharding', 'blobCommitmentVersionKzg')) {
                const msg = this._errorMsg('versioned hash does not start with KZG commitment version');
                throw new Error(msg);
            }
        }
        if (this.blobVersionedHashes.length > constants_js_1.LIMIT_BLOBS_PER_TX) {
            const msg = this._errorMsg(`tx can contain at most ${constants_js_1.LIMIT_BLOBS_PER_TX} blobs`);
            throw new Error(msg);
        }
        else if (this.blobVersionedHashes.length === 0) {
            const msg = this._errorMsg(`tx should contain at least one blob`);
            throw new Error(msg);
        }
        if (this.to === undefined) {
            const msg = this._errorMsg(`tx should have a "to" field and cannot be used to create contracts`);
            throw new Error(msg);
        }
        this.blobs = txData.blobs?.map((blob) => (0, ethereumjs_util_1.toBytes)(blob));
        this.kzgCommitments = txData.kzgCommitments?.map((commitment) => (0, ethereumjs_util_1.toBytes)(commitment));
        this.kzgProofs = txData.kzgProofs?.map((proof) => (0, ethereumjs_util_1.toBytes)(proof));
        const freeze = opts?.freeze ?? true;
        if (freeze) {
            Object.freeze(this);
        }
    }
    static fromTxData(txData, opts) {
        if (txData.blobsData !== undefined) {
            if (txData.blobs !== undefined) {
                throw new Error('cannot have both raw blobs data and encoded blobs in constructor');
            }
            if (txData.kzgCommitments !== undefined) {
                throw new Error('cannot have both raw blobs data and KZG commitments in constructor');
            }
            if (txData.blobVersionedHashes !== undefined) {
                throw new Error('cannot have both raw blobs data and versioned hashes in constructor');
            }
            if (txData.kzgProofs !== undefined) {
                throw new Error('cannot have both raw blobs data and KZG proofs in constructor');
            }
            txData.blobs = (0, ethereumjs_util_1.getBlobs)(txData.blobsData.reduce((acc, cur) => acc + cur));
            txData.kzgCommitments = (0, ethereumjs_util_1.blobsToCommitments)(txData.blobs);
            txData.blobVersionedHashes = (0, ethereumjs_util_1.commitmentsToVersionedHashes)(txData.kzgCommitments);
            txData.kzgProofs = (0, ethereumjs_util_1.blobsToProofs)(txData.blobs, txData.kzgCommitments);
        }
        return new BlobEIP4844Transaction(txData, opts);
    }
    /**
     * Creates the minimal representation of a blob transaction from the network wrapper version.
     * The minimal representation is used when adding transactions to an execution payload/block
     * @param txData a {@link BlobEIP4844Transaction} containing optional blobs/kzg commitments
     * @param opts - dictionary of {@link TxOptions}
     * @returns the "minimal" representation of a BlobEIP4844Transaction (i.e. transaction object minus blobs and kzg commitments)
     */
    static minimalFromNetworkWrapper(txData, opts) {
        const tx = BlobEIP4844Transaction.fromTxData({
            ...txData,
            ...{ blobs: undefined, kzgCommitments: undefined, kzgProofs: undefined },
        }, opts);
        return tx;
    }
    /**
     * Instantiate a transaction from the serialized tx.
     *
     * Format: `0x03 || rlp([chain_id, nonce, max_priority_fee_per_gas, max_fee_per_gas, gas_limit, to, value, data,
     * access_list, max_fee_per_data_gas, blob_versioned_hashes, y_parity, r, s])`
     */
    static fromSerializedTx(serialized, opts = {}) {
        if ((0, ethereumjs_util_1.equalsBytes)(serialized.subarray(0, 1), (0, util_js_1.txTypeBytes)(types_js_1.TransactionType.BlobEIP4844)) === false) {
            throw new Error(`Invalid serialized tx input: not an EIP-4844 transaction (wrong tx type, expected: ${types_js_1.TransactionType.BlobEIP4844}, received: ${(0, ethereumjs_util_1.bytesToHex)(serialized.subarray(0, 1))}`);
        }
        const values = ethereumjs_rlp_1.RLP.decode(serialized.subarray(1));
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized tx input: must be array');
        }
        return BlobEIP4844Transaction.fromValuesArray(values, opts);
    }
    /**
     * Create a transaction from a values array.
     *
     * Format: `[chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data,
     * accessList, signatureYParity, signatureR, signatureS]`
     */
    static fromValuesArray(values, opts = {}) {
        if (values.length !== 11 && values.length !== 14) {
            throw new Error('Invalid EIP-4844 transaction. Only expecting 11 values (for unsigned tx) or 14 values (for signed tx).');
        }
        const [chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data, accessList, maxFeePerBlobGas, blobVersionedHashes, v, r, s,] = values;
        this._validateNotArray({ chainId, v });
        (0, ethereumjs_util_1.validateNoLeadingZeroes)({
            nonce,
            maxPriorityFeePerGas,
            maxFeePerGas,
            gasLimit,
            value,
            maxFeePerBlobGas,
            v,
            r,
            s,
        });
        return new BlobEIP4844Transaction({
            chainId: (0, ethereumjs_util_1.bytesToBigInt)(chainId),
            nonce,
            maxPriorityFeePerGas,
            maxFeePerGas,
            gasLimit,
            to,
            value,
            data,
            accessList: accessList ?? [],
            maxFeePerBlobGas,
            blobVersionedHashes,
            v: v !== undefined ? (0, ethereumjs_util_1.bytesToBigInt)(v) : undefined,
            r,
            s,
        }, opts);
    }
    /**
     * Creates a transaction from the network encoding of a blob transaction (with blobs/commitments/proof)
     * @param serialized a buffer representing a serialized BlobTransactionNetworkWrapper
     * @param opts any TxOptions defined
     * @returns a BlobEIP4844Transaction
     */
    static fromSerializedBlobTxNetworkWrapper(serialized, opts) {
        if (!opts || !opts.common) {
            throw new Error('common instance required to validate versioned hashes');
        }
        if ((0, ethereumjs_util_1.equalsBytes)(serialized.subarray(0, 1), (0, util_js_1.txTypeBytes)(types_js_1.TransactionType.BlobEIP4844)) === false) {
            throw new Error(`Invalid serialized tx input: not an EIP-4844 transaction (wrong tx type, expected: ${types_js_1.TransactionType.BlobEIP4844}, received: ${(0, ethereumjs_util_1.bytesToHex)(serialized.subarray(0, 1))}`);
        }
        // Validate network wrapper
        const networkTxValues = ethereumjs_rlp_1.RLP.decode(serialized.subarray(1));
        if (networkTxValues.length !== 4) {
            throw Error(`Expected 4 values in the deserialized network transaction`);
        }
        const [txValues, blobs, kzgCommitments, kzgProofs] = networkTxValues;
        // Construct the tx but don't freeze yet, we will assign blobs etc once validated
        const decodedTx = BlobEIP4844Transaction.fromValuesArray(txValues, { ...opts, freeze: false });
        if (decodedTx.to === undefined) {
            throw Error('BlobEIP4844Transaction can not be send without a valid `to`');
        }
        const version = Number(opts.common.param('sharding', 'blobCommitmentVersionKzg'));
        validateBlobTransactionNetworkWrapper(decodedTx.blobVersionedHashes, blobs, kzgCommitments, kzgProofs, version);
        // set the network blob data on the tx
        decodedTx.blobs = blobs;
        decodedTx.kzgCommitments = kzgCommitments;
        decodedTx.kzgProofs = kzgProofs;
        // freeze the tx
        const freeze = opts?.freeze ?? true;
        if (freeze) {
            Object.freeze(decodedTx);
        }
        return decodedTx;
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
     * Returns a Uint8Array Array of the raw Bytes of the EIP-4844 transaction, in order.
     *
     * Format: [chain_id, nonce, max_priority_fee_per_gas, max_fee_per_gas, gas_limit, to, value, data,
     * access_list, max_fee_per_data_gas, blob_versioned_hashes, y_parity, r, s]`.
     *
     * Use {@link BlobEIP4844Transaction.serialize} to add a transaction to a block
     * with {@link Block.fromValuesArray}.
     *
     * For an unsigned tx this method uses the empty Bytes values for the
     * signature parameters `v`, `r` and `s` for encoding. For an EIP-155 compliant
     * representation for external signing use {@link BlobEIP4844Transaction.getMessageToSign}.
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
            (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.maxFeePerBlobGas),
            this.blobVersionedHashes,
            this.v !== undefined ? (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.v) : new Uint8Array(0),
            this.r !== undefined ? (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.r) : new Uint8Array(0),
            this.s !== undefined ? (0, ethereumjs_util_1.bigIntToUnpaddedBytes)(this.s) : new Uint8Array(0),
        ];
    }
    /**
     * Returns the serialized encoding of the EIP-4844 transaction.
     *
     * Format: `0x03 || rlp([chainId, nonce, maxPriorityFeePerGas, maxFeePerGas, gasLimit, to, value, data,
     * access_list, max_fee_per_data_gas, blob_versioned_hashes, y_parity, r, s])`.
     *
     * Note that in contrast to the legacy tx serialization format this is not
     * valid RLP any more due to the raw tx type preceding and concatenated to
     * the RLP encoding of the values.
     */
    serialize() {
        return EIP2718.serialize(this);
    }
    /**
     * @returns the serialized form of a blob transaction in the network wrapper format (used for gossipping mempool transactions over devp2p)
     */
    serializeNetworkWrapper() {
        if (this.blobs === undefined ||
            this.kzgCommitments === undefined ||
            this.kzgProofs === undefined) {
            throw new Error('cannot serialize network wrapper without blobs, KZG commitments and KZG proofs provided');
        }
        return EIP2718.serialize(this, [this.raw(), this.blobs, this.kzgCommitments, this.kzgProofs]);
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
        return EIP2718.serialize(this, this.raw().slice(0, 11));
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
     * Use {@link BlobEIP4844Transaction.getMessageToSign} to get a tx hash for the purpose of signing.
     */
    hash() {
        return Legacy.hash(this);
    }
    getMessageToVerifySignature() {
        return this.getHashedMessageToSign();
    }
    /**
     * Returns the public key of the sender
     */
    _getSenderPublicKey() {
        return Legacy.getSenderPublicKey(this);
    }
    toJSON() {
        const accessListJSON = util_js_1.AccessLists.getAccessListJSON(this.accessList);
        const baseJson = super.toJSON();
        return {
            ...baseJson,
            chainId: (0, ethereumjs_util_1.bigIntToHex)(this.chainId),
            maxPriorityFeePerGas: (0, ethereumjs_util_1.bigIntToHex)(this.maxPriorityFeePerGas),
            maxFeePerGas: (0, ethereumjs_util_1.bigIntToHex)(this.maxFeePerGas),
            accessList: accessListJSON,
            maxFeePerBlobGas: (0, ethereumjs_util_1.bigIntToHex)(this.maxFeePerBlobGas),
            blobVersionedHashes: this.blobVersionedHashes.map((hash) => (0, ethereumjs_util_1.bytesToHex)(hash)),
        };
    }
    _processSignature(v, r, s) {
        const opts = { ...this.txOptions, common: this.common };
        return BlobEIP4844Transaction.fromTxData({
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
            maxFeePerBlobGas: this.maxFeePerBlobGas,
            blobVersionedHashes: this.blobVersionedHashes,
            blobs: this.blobs,
            kzgCommitments: this.kzgCommitments,
            kzgProofs: this.kzgProofs,
        }, opts);
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
    /**
     * @returns the number of blobs included with this transaction
     */
    numBlobs() {
        return this.blobVersionedHashes.length;
    }
}
exports.BlobEIP4844Transaction = BlobEIP4844Transaction;
//# sourceMappingURL=eip4844Transaction.js.map