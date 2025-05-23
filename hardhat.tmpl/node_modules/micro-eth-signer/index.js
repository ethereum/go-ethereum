"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Transaction = exports.authorization = exports.weigwei = exports.weieth = exports.addr = void 0;
/*! micro-eth-signer - MIT License (c) 2021 Paul Miller (paulmillr.com) */
const sha3_1 = require("@noble/hashes/sha3");
const utils_1 = require("@noble/hashes/utils");
const address_ts_1 = require("./address.js");
Object.defineProperty(exports, "addr", { enumerable: true, get: function () { return address_ts_1.addr; } });
// prettier-ignore
const rlp_ts_1 = require("./rlp.js");
const tx_ts_1 = require("./tx.js");
// prettier-ignore
const utils_ts_1 = require("./utils.js");
Object.defineProperty(exports, "weieth", { enumerable: true, get: function () { return utils_ts_1.weieth; } });
Object.defineProperty(exports, "weigwei", { enumerable: true, get: function () { return utils_ts_1.weigwei; } });
// The file exports Transaction, but actual (RLP) parsing logic is done in `./tx`
/**
 * EIP-7702 Authorizations
 */
exports.authorization = {
    _getHash(req) {
        const msg = rlp_ts_1.RLP.encode(tx_ts_1.authorizationRequest.decode(req));
        return (0, sha3_1.keccak_256)((0, utils_1.concatBytes)(new Uint8Array([0x05]), msg));
    },
    sign(req, privateKey) {
        (0, utils_ts_1.astr)(privateKey);
        const sig = (0, utils_ts_1.sign)(this._getHash(req), utils_ts_1.ethHex.decode(privateKey));
        return { ...req, r: sig.r, s: sig.s, yParity: sig.recovery };
    },
    getAuthority(item) {
        const { r, s, yParity, ...req } = item;
        const hash = this._getHash(req);
        const sig = (0, utils_ts_1.initSig)({ r, s }, yParity);
        const point = sig.recoverPublicKey(hash);
        return address_ts_1.addr.fromPublicKey(point.toHex(false));
    },
};
// Transaction-related utils.
// 4 fields are required. Others are pre-filled with default values.
const TX_DEFAULTS = {
    accessList: [], // needs to be .slice()-d to create new reference
    authorizationList: [],
    chainId: BigInt(1), // mainnet
    data: '',
    gasLimit: BigInt(21000), // TODO: investigate if limit is smaller in eip4844 txs
    maxPriorityFeePerGas: (BigInt(1) * utils_ts_1.amounts.GWEI), // Reduce fingerprinting by using standard, popular value
    type: 'eip1559',
};
// Changes:
// - legacy: instead of hardfork now accepts additional param chainId
//           if chainId is present, we enable relay protection
//           This removes hardfork param and simplifies replay protection logic
// - tx parametrized over type: you cannot access fields from different tx version
// - legacy: 'v' param is hidden in coders. Transaction operates in terms chainId and yParity.
// TODO: tx is kinda immutable, but user can change .raw values before signing
// need to think about re-validation?
class Transaction {
    // Doesn't force any defaults, catches if fields incompatible with type
    constructor(type, raw, strict = true, allowSignatureFields = true) {
        this.type = type;
        this.raw = raw;
        (0, tx_ts_1.validateFields)(type, raw, strict, allowSignatureFields);
        this.isSigned = typeof raw.r === 'bigint' && typeof raw.s === 'bigint';
    }
    static prepare(data, strict = true) {
        const type = (data.type !== undefined ? data.type : TX_DEFAULTS.type);
        if (!tx_ts_1.TxVersions.hasOwnProperty(type))
            throw new Error(`wrong transaction type=${type}`);
        const coder = tx_ts_1.TxVersions[type];
        const fields = new Set(coder.fields);
        // Copy default fields, but only if the field is present on the tx type.
        const raw = { type };
        for (const f in TX_DEFAULTS) {
            if (f !== 'type' && fields.has(f)) {
                raw[f] = TX_DEFAULTS[f];
                if (['accessList', 'authorizationList'].includes(f))
                    raw[f] = (0, utils_ts_1.cloneDeep)(raw[f]);
            }
        }
        // Copy all fields, so we can validate unexpected ones.
        return new Transaction(type, (0, tx_ts_1.sortRawData)(Object.assign(raw, data)), strict, false);
    }
    /**
     * Creates transaction which sends whole account balance. Does two things:
     * 1. `amount = accountBalance - maxFeePerGas * gasLimit`
     * 2. `maxPriorityFeePerGas = maxFeePerGas`
     *
     * Every eth block sets a fee for all its transactions, called base fee.
     * maxFeePerGas indicates how much gas user is able to spend in the worst case.
     * If the block's base fee is 5 gwei, while user is able to spend 10 gwei in maxFeePerGas,
     * the transaction would only consume 5 gwei. That means, base fee is unknown
     * before the transaction is included in a block.
     *
     * By setting priorityFee to maxFee, we make the process deterministic:
     * `maxFee = 10, maxPriority = 10, baseFee = 5` would always spend 10 gwei.
     * In the end, the balance would become 0.
     *
     * WARNING: using the method would decrease privacy of a transfer, because
     * payments for services have specific amounts, and not *the whole amount*.
     * @param accountBalance - account balance in wei
     * @param burnRemaining - send unspent fee to miners. When false, some "small amount" would remain
     * @returns new transaction with adjusted amounts
     */
    setWholeAmount(accountBalance, burnRemaining = true) {
        const _0n = BigInt(0);
        if (typeof accountBalance !== 'bigint' || accountBalance <= _0n)
            throw new Error('account balance must be bigger than 0');
        const fee = this.fee;
        const amountToSend = accountBalance - fee;
        if (amountToSend <= _0n)
            throw new Error('account balance must be bigger than fee of ' + fee);
        const raw = { ...this.raw, value: amountToSend };
        if (!['legacy', 'eip2930'].includes(this.type) && burnRemaining) {
            const r = raw;
            r.maxPriorityFeePerGas = r.maxFeePerGas;
        }
        return new Transaction(this.type, raw);
    }
    static fromRawBytes(bytes, strict = false) {
        const raw = tx_ts_1.RawTx.decode(bytes);
        return new Transaction(raw.type, raw.data, strict);
    }
    static fromHex(hex, strict = false) {
        return Transaction.fromRawBytes(utils_ts_1.ethHexNoLeadingZero.decode(hex), strict);
    }
    assertIsSigned() {
        if (!this.isSigned)
            throw new Error('expected signed transaction');
    }
    /**
     * Converts transaction to RLP.
     * @param includeSignature whether to include signature
     */
    toRawBytes(includeSignature = this.isSigned) {
        // cloneDeep is not necessary here
        let data = Object.assign({}, this.raw);
        if (includeSignature) {
            this.assertIsSigned();
        }
        else {
            (0, tx_ts_1.removeSig)(data);
        }
        return tx_ts_1.RawTx.encode({ type: this.type, data }); // TODO: remove any
    }
    /**
     * Converts transaction to hex.
     * @param includeSignature whether to include signature
     */
    toHex(includeSignature = this.isSigned) {
        return utils_ts_1.ethHex.encode(this.toRawBytes(includeSignature));
    }
    /** Calculates keccak-256 hash of signed transaction. Used in block explorers. */
    get hash() {
        this.assertIsSigned();
        return (0, utils_1.bytesToHex)(this.calcHash(true));
    }
    /** Returns sender's address. */
    get sender() {
        return this.recoverSender().address;
    }
    /**
     * For legacy transactions, but can be used with libraries when yParity presented as v.
     */
    get v() {
        return (0, tx_ts_1.decodeLegacyV)(this.raw);
    }
    calcHash(includeSignature) {
        return (0, sha3_1.keccak_256)(this.toRawBytes(includeSignature));
    }
    /** Calculates MAXIMUM fee in wei that could be spent. */
    get fee() {
        const { type, raw } = this;
        // Fee calculation is not exact, real fee can be smaller
        let gasFee;
        if (type === 'legacy' || type === 'eip2930') {
            // Because TypeScript is not smart enough to narrow down types here :(
            const r = raw;
            gasFee = r.gasPrice;
        }
        else {
            const r = raw;
            // maxFeePerGas is absolute limit, you never pay more than that
            // maxFeePerGas = baseFeePerGas[*2] + maxPriorityFeePerGas
            gasFee = r.maxFeePerGas;
        }
        // TODO: how to calculate 4844 fee?
        return raw.gasLimit * gasFee;
    }
    clone() {
        return new Transaction(this.type, (0, utils_ts_1.cloneDeep)(this.raw));
    }
    verifySignature() {
        this.assertIsSigned();
        const { r, s } = this.raw;
        return (0, utils_ts_1.verify)({ r: r, s: s }, this.calcHash(false), (0, utils_1.hexToBytes)(this.recoverSender().publicKey));
    }
    removeSignature() {
        return new Transaction(this.type, (0, tx_ts_1.removeSig)((0, utils_ts_1.cloneDeep)(this.raw)));
    }
    /**
     * Signs transaction with a private key.
     * @param privateKey key in hex or Uint8Array format
     * @param opts extraEntropy will increase security of sig by mixing rfc6979 randomness
     * @returns new "same" transaction, but signed
     */
    signBy(privateKey, extraEntropy = true) {
        if (this.isSigned)
            throw new Error('expected unsigned transaction');
        const priv = (0, utils_ts_1.isBytes)(privateKey) ? privateKey : (0, utils_1.hexToBytes)((0, utils_ts_1.strip0x)(privateKey));
        const hash = this.calcHash(false);
        const { r, s, recovery } = (0, utils_ts_1.sign)(hash, priv, extraEntropy);
        const sraw = Object.assign((0, utils_ts_1.cloneDeep)(this.raw), { r, s, yParity: recovery });
        // The copied result is validated in non-strict way, strict is only for user input.
        return new Transaction(this.type, sraw, false);
    }
    /** Calculates public key and address from signed transaction's signature. */
    recoverSender() {
        this.assertIsSigned();
        const { r, s, yParity } = this.raw;
        const sig = (0, utils_ts_1.initSig)({ r: r, s: s }, yParity);
        // Will crash on 'chainstart' hardfork
        if (sig.hasHighS())
            throw new Error('invalid s');
        const point = sig.recoverPublicKey(this.calcHash(false));
        return { publicKey: point.toHex(true), address: address_ts_1.addr.fromPublicKey(point.toHex(false)) };
    }
}
exports.Transaction = Transaction;
//# sourceMappingURL=index.js.map