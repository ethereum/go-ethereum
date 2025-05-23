import { addr } from './address.ts';
import { type AuthorizationItem, type AuthorizationRequest, type TxCoder, type TxType } from './tx.ts';
import { weieth, weigwei } from './utils.ts';
export { addr, weieth, weigwei };
/**
 * EIP-7702 Authorizations
 */
export declare const authorization: {
    _getHash(req: AuthorizationRequest): Uint8Array;
    sign(req: AuthorizationRequest, privateKey: string): AuthorizationItem;
    getAuthority(item: AuthorizationItem): string;
};
declare const TX_DEFAULTS: {
    readonly accessList: readonly [];
    readonly authorizationList: readonly [];
    readonly chainId: bigint;
    readonly data: "";
    readonly gasLimit: bigint;
    readonly maxPriorityFeePerGas: bigint;
    readonly type: "eip1559";
};
type DefaultField = keyof typeof TX_DEFAULTS;
type DefaultType = (typeof TX_DEFAULTS)['type'];
type DefaultsOptional<T> = {
    [P in keyof T as P extends DefaultField ? P : never]?: T[P];
} & {
    [P in keyof T as P extends DefaultField ? never : P]: T[P];
};
type HumanInputInner<T extends TxType> = DefaultsOptional<{
    type: T;
} & TxCoder<T>>;
type HumanInputInnerDefault = DefaultsOptional<TxCoder<DefaultType>>;
type Required<T> = T extends undefined ? never : T;
type HumanInput<T extends TxType | undefined> = T extends undefined ? HumanInputInnerDefault : HumanInputInner<Required<T>>;
export declare class Transaction<T extends TxType> {
    readonly type: T;
    readonly raw: TxCoder<T>;
    readonly isSigned: boolean;
    constructor(type: T, raw: TxCoder<T>, strict?: boolean, allowSignatureFields?: boolean);
    static prepare<T extends {
        type: undefined;
    }>(data: T & HumanInputInnerDefault, strict?: boolean): Transaction<(typeof TX_DEFAULTS)['type']>;
    static prepare<TT extends TxType, T extends {
        type: TT;
    } & HumanInput<TT>>(data: HumanInput<TT>, strict?: boolean): Transaction<T['type']>;
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
    setWholeAmount(accountBalance: bigint, burnRemaining?: boolean): Transaction<T>;
    static fromRawBytes(bytes: Uint8Array, strict?: boolean): Transaction<'legacy' | 'eip2930' | 'eip1559' | 'eip4844' | 'eip7702'>;
    static fromHex(hex: string, strict?: boolean): Transaction<'eip1559' | 'legacy' | 'eip2930' | 'eip4844' | 'eip7702'>;
    private assertIsSigned;
    /**
     * Converts transaction to RLP.
     * @param includeSignature whether to include signature
     */
    toRawBytes(includeSignature?: boolean): Uint8Array;
    /**
     * Converts transaction to hex.
     * @param includeSignature whether to include signature
     */
    toHex(includeSignature?: boolean): string;
    /** Calculates keccak-256 hash of signed transaction. Used in block explorers. */
    get hash(): string;
    /** Returns sender's address. */
    get sender(): string;
    /**
     * For legacy transactions, but can be used with libraries when yParity presented as v.
     */
    get v(): bigint | undefined;
    private calcHash;
    /** Calculates MAXIMUM fee in wei that could be spent. */
    get fee(): bigint;
    clone(): Transaction<T>;
    verifySignature(): boolean;
    removeSignature(): Transaction<T>;
    /**
     * Signs transaction with a private key.
     * @param privateKey key in hex or Uint8Array format
     * @param opts extraEntropy will increase security of sig by mixing rfc6979 randomness
     * @returns new "same" transaction, but signed
     */
    signBy(privateKey: string | Uint8Array, extraEntropy?: boolean | Uint8Array): Transaction<T>;
    /** Calculates public key and address from signed transaction's signature. */
    recoverSender(): {
        publicKey: string;
        address: string;
    };
}
//# sourceMappingURL=index.d.ts.map