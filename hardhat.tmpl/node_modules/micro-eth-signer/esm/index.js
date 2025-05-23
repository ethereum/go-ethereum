/*! micro-eth-signer - MIT License (c) 2021 Paul Miller (paulmillr.com) */
import { keccak_256 } from '@noble/hashes/sha3';
import { bytesToHex, concatBytes, hexToBytes } from '@noble/hashes/utils';
import { addr } from "./address.js";
// prettier-ignore
import { RLP } from "./rlp.js";
import { RawTx, TxVersions, authorizationRequest, decodeLegacyV, removeSig, sortRawData, validateFields, } from "./tx.js";
// prettier-ignore
import { amounts, astr, cloneDeep, ethHex, ethHexNoLeadingZero, initSig, isBytes, sign, strip0x, verify, weieth, weigwei } from "./utils.js";
export { addr, weieth, weigwei };
// The file exports Transaction, but actual (RLP) parsing logic is done in `./tx`
/**
 * EIP-7702 Authorizations
 */
export const authorization = {
    _getHash(req) {
        const msg = RLP.encode(authorizationRequest.decode(req));
        return keccak_256(concatBytes(new Uint8Array([0x05]), msg));
    },
    sign(req, privateKey) {
        astr(privateKey);
        const sig = sign(this._getHash(req), ethHex.decode(privateKey));
        return { ...req, r: sig.r, s: sig.s, yParity: sig.recovery };
    },
    getAuthority(item) {
        const { r, s, yParity, ...req } = item;
        const hash = this._getHash(req);
        const sig = initSig({ r, s }, yParity);
        const point = sig.recoverPublicKey(hash);
        return addr.fromPublicKey(point.toHex(false));
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
    maxPriorityFeePerGas: (BigInt(1) * amounts.GWEI), // Reduce fingerprinting by using standard, popular value
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
export class Transaction {
    // Doesn't force any defaults, catches if fields incompatible with type
    constructor(type, raw, strict = true, allowSignatureFields = true) {
        this.type = type;
        this.raw = raw;
        validateFields(type, raw, strict, allowSignatureFields);
        this.isSigned = typeof raw.r === 'bigint' && typeof raw.s === 'bigint';
    }
    static prepare(data, strict = true) {
        const type = (data.type !== undefined ? data.type : TX_DEFAULTS.type);
        if (!TxVersions.hasOwnProperty(type))
            throw new Error(`wrong transaction type=${type}`);
        const coder = TxVersions[type];
        const fields = new Set(coder.fields);
        // Copy default fields, but only if the field is present on the tx type.
        const raw = { type };
        for (const f in TX_DEFAULTS) {
            if (f !== 'type' && fields.has(f)) {
                raw[f] = TX_DEFAULTS[f];
                if (['accessList', 'authorizationList'].includes(f))
                    raw[f] = cloneDeep(raw[f]);
            }
        }
        // Copy all fields, so we can validate unexpected ones.
        return new Transaction(type, sortRawData(Object.assign(raw, data)), strict, false);
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
        const raw = RawTx.decode(bytes);
        return new Transaction(raw.type, raw.data, strict);
    }
    static fromHex(hex, strict = false) {
        return Transaction.fromRawBytes(ethHexNoLeadingZero.decode(hex), strict);
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
            removeSig(data);
        }
        return RawTx.encode({ type: this.type, data }); // TODO: remove any
    }
    /**
     * Converts transaction to hex.
     * @param includeSignature whether to include signature
     */
    toHex(includeSignature = this.isSigned) {
        return ethHex.encode(this.toRawBytes(includeSignature));
    }
    /** Calculates keccak-256 hash of signed transaction. Used in block explorers. */
    get hash() {
        this.assertIsSigned();
        return bytesToHex(this.calcHash(true));
    }
    /** Returns sender's address. */
    get sender() {
        return this.recoverSender().address;
    }
    /**
     * For legacy transactions, but can be used with libraries when yParity presented as v.
     */
    get v() {
        return decodeLegacyV(this.raw);
    }
    calcHash(includeSignature) {
        return keccak_256(this.toRawBytes(includeSignature));
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
        return new Transaction(this.type, cloneDeep(this.raw));
    }
    verifySignature() {
        this.assertIsSigned();
        const { r, s } = this.raw;
        return verify({ r: r, s: s }, this.calcHash(false), hexToBytes(this.recoverSender().publicKey));
    }
    removeSignature() {
        return new Transaction(this.type, removeSig(cloneDeep(this.raw)));
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
        const priv = isBytes(privateKey) ? privateKey : hexToBytes(strip0x(privateKey));
        const hash = this.calcHash(false);
        const { r, s, recovery } = sign(hash, priv, extraEntropy);
        const sraw = Object.assign(cloneDeep(this.raw), { r, s, yParity: recovery });
        // The copied result is validated in non-strict way, strict is only for user input.
        return new Transaction(this.type, sraw, false);
    }
    /** Calculates public key and address from signed transaction's signature. */
    recoverSender() {
        this.assertIsSigned();
        const { r, s, yParity } = this.raw;
        const sig = initSig({ r: r, s: s }, yParity);
        // Will crash on 'chainstart' hardfork
        if (sig.hasHighS())
            throw new Error('invalid s');
        const point = sig.recoverPublicKey(this.calcHash(false));
        return { publicKey: point.toHex(true), address: addr.fromPublicKey(point.toHex(false)) };
    }
}
//# sourceMappingURL=index.js.map