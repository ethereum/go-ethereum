import { RLP } from '@nomicfoundation/ethereumjs-rlp';
import { BIGINT_2, BIGINT_8, MAX_INTEGER, bigIntToHex, bigIntToUnpaddedBytes, bytesToBigInt, toBytes, unpadBytes, validateNoLeadingZeroes, } from '@nomicfoundation/ethereumjs-util';
import { keccak256 as bufferKeccak256 } from 'ethereum-cryptography/keccak.js';
import { BaseTransaction } from './baseTransaction.js';
import * as Legacy from './capabilities/legacy.js';
import { Capability, TransactionType } from './types.js';
function keccak256(msg) {
    return new Uint8Array(bufferKeccak256(Buffer.from(msg)));
}
function meetsEIP155(_v, chainId) {
    const v = Number(_v);
    const chainIdDoubled = Number(chainId) * 2;
    return v === chainIdDoubled + 35 || v === chainIdDoubled + 36;
}
/**
 * An Ethereum non-typed (legacy) transaction
 */
export class LegacyTransaction extends BaseTransaction {
    /**
     * This constructor takes the values, validates them, assigns them and freezes the object.
     *
     * It is not recommended to use this constructor directly. Instead use
     * the static factory methods to assist in creating a Transaction object from
     * varying data types.
     */
    constructor(txData, opts = {}) {
        super({ ...txData, type: TransactionType.Legacy }, opts);
        this.common = this._validateTxV(this.v, opts.common);
        this.keccakFunction = this.common.customCrypto.keccak256 ?? keccak256;
        this.gasPrice = bytesToBigInt(toBytes(txData.gasPrice === '' ? '0x' : txData.gasPrice));
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
                if (meetsEIP155(this.v, this.common.chainId())) {
                    this.activeCapabilities.push(Capability.EIP155ReplayProtection);
                }
            }
        }
        const freeze = opts?.freeze ?? true;
        if (freeze) {
            Object.freeze(this);
        }
    }
    /**
     * Instantiate a transaction from a data dictionary.
     *
     * Format: { nonce, gasPrice, gasLimit, to, value, data, v, r, s }
     *
     * Notes:
     * - All parameters are optional and have some basic default values
     */
    static fromTxData(txData, opts = {}) {
        return new LegacyTransaction(txData, opts);
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
        // If length is not 6, it has length 9. If v/r/s are empty Uint8Arrays, it is still an unsigned transaction
        // This happens if you get the RLP data from `raw()`
        if (values.length !== 6 && values.length !== 9) {
            throw new Error('Invalid transaction. Only expecting 6 values (for unsigned tx) or 9 values (for signed tx).');
        }
        const [nonce, gasPrice, gasLimit, to, value, data, v, r, s] = values;
        validateNoLeadingZeroes({ nonce, gasPrice, gasLimit, value, v, r, s });
        return new LegacyTransaction({
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
     * Returns a Uint8Array Array of the raw Bytes of the legacy transaction, in order.
     *
     * Format: `[nonce, gasPrice, gasLimit, to, value, data, v, r, s]`
     *
     * For legacy txs this is also the correct format to add transactions
     * to a block with {@link Block.fromValuesArray} (use the `serialize()` method
     * for typed txs).
     *
     * For an unsigned tx this method returns the empty Bytes values
     * for the signature parameters `v`, `r` and `s`. For an EIP-155 compliant
     * representation have a look at {@link Transaction.getMessageToSign}.
     */
    raw() {
        return [
            bigIntToUnpaddedBytes(this.nonce),
            bigIntToUnpaddedBytes(this.gasPrice),
            bigIntToUnpaddedBytes(this.gasLimit),
            this.to !== undefined ? this.to.bytes : new Uint8Array(0),
            bigIntToUnpaddedBytes(this.value),
            this.data,
            this.v !== undefined ? bigIntToUnpaddedBytes(this.v) : new Uint8Array(0),
            this.r !== undefined ? bigIntToUnpaddedBytes(this.r) : new Uint8Array(0),
            this.s !== undefined ? bigIntToUnpaddedBytes(this.s) : new Uint8Array(0),
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
    /**
     * Returns the raw unsigned tx, which can be used
     * to sign the transaction (e.g. for sending to a hardware wallet).
     *
     * Note: the raw message message format for the legacy tx is not RLP encoded
     * and you might need to do yourself with:
     *
     * ```javascript
     * import { RLP } from '@nomicfoundation/ethereumjs-rlp'
     * const message = tx.getMessageToSign()
     * const serializedMessage = RLP.encode(message)) // use this for the HW wallet input
     * ```
     */
    getMessageToSign() {
        const message = [
            bigIntToUnpaddedBytes(this.nonce),
            bigIntToUnpaddedBytes(this.gasPrice),
            bigIntToUnpaddedBytes(this.gasLimit),
            this.to !== undefined ? this.to.bytes : new Uint8Array(0),
            bigIntToUnpaddedBytes(this.value),
            this.data,
        ];
        if (this.supports(Capability.EIP155ReplayProtection)) {
            message.push(bigIntToUnpaddedBytes(this.common.chainId()));
            message.push(unpadBytes(toBytes(0)));
            message.push(unpadBytes(toBytes(0)));
        }
        return message;
    }
    /**
     * Returns the hashed serialized unsigned tx, which can be used
     * to sign the transaction (e.g. for sending to a hardware wallet).
     */
    getHashedMessageToSign() {
        const message = this.getMessageToSign();
        return this.keccakFunction(RLP.encode(message));
    }
    /**
     * The amount of gas paid for the data in this tx
     */
    getDataFee() {
        return Legacy.getDataFee(this);
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
        return Legacy.hash(this);
    }
    /**
     * Computes a sha3-256 hash which can be used to verify the signature
     */
    getMessageToVerifySignature() {
        if (!this.isSigned()) {
            const msg = this._errorMsg('This transaction is not signed');
            throw new Error(msg);
        }
        return this.getHashedMessageToSign();
    }
    /**
     * Returns the public key of the sender
     */
    _getSenderPublicKey() {
        return Legacy.getSenderPublicKey(this);
    }
    /**
     * Process the v, r, s values from the `sign` method of the base transaction.
     */
    _processSignature(v, r, s) {
        if (this.supports(Capability.EIP155ReplayProtection)) {
            v += this.common.chainId() * BIGINT_2 + BIGINT_8;
        }
        const opts = { ...this.txOptions, common: this.common };
        return LegacyTransaction.fromTxData({
            nonce: this.nonce,
            gasPrice: this.gasPrice,
            gasLimit: this.gasLimit,
            to: this.to,
            value: this.value,
            data: this.data,
            v,
            r: bytesToBigInt(r),
            s: bytesToBigInt(s),
        }, opts);
    }
    /**
     * Returns an object with the JSON representation of the transaction.
     */
    toJSON() {
        const baseJson = super.toJSON();
        return {
            ...baseJson,
            gasPrice: bigIntToHex(this.gasPrice),
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
                chainIdBigInt = BigInt(v - numSub) / BIGINT_2;
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
        return Legacy.errorMsg(this, msg);
    }
}
//# sourceMappingURL=legacyTransaction.js.map