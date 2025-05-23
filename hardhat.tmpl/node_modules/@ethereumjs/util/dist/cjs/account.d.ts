import type { BigIntLike, BytesLike, PrefixedHexString } from './types.js';
export interface AccountData {
    nonce?: BigIntLike;
    balance?: BigIntLike;
    storageRoot?: BytesLike;
    codeHash?: BytesLike;
}
export interface PartialAccountData {
    nonce?: BigIntLike | null;
    balance?: BigIntLike | null;
    storageRoot?: BytesLike | null;
    codeHash?: BytesLike | null;
    codeSize?: BigIntLike | null;
    version?: BigIntLike | null;
}
export declare type AccountBodyBytes = [Uint8Array, Uint8Array, Uint8Array, Uint8Array];
/**
 * Account class to load and maintain the  basic account objects.
 * Supports partial loading and access required for verkle with null
 * as the placeholder.
 *
 * Note: passing undefined in constructor is different from null
 * While undefined leads to default assignment, null is retained
 * to track the information not available/loaded because of partial
 * witness access
 */
export declare class Account {
    _nonce: bigint | null;
    _balance: bigint | null;
    _storageRoot: Uint8Array | null;
    _codeHash: Uint8Array | null;
    _codeSize: number | null;
    _version: number | null;
    get version(): number;
    set version(_version: number);
    get nonce(): bigint;
    set nonce(_nonce: bigint);
    get balance(): bigint;
    set balance(_balance: bigint);
    get storageRoot(): Uint8Array;
    set storageRoot(_storageRoot: Uint8Array);
    get codeHash(): Uint8Array;
    set codeHash(_codeHash: Uint8Array);
    get codeSize(): number;
    set codeSize(_codeSize: number);
    static fromAccountData(accountData: AccountData): Account;
    static fromPartialAccountData(partialAccountData: PartialAccountData): Account;
    static fromRlpSerializedAccount(serialized: Uint8Array): Account;
    static fromRlpSerializedPartialAccount(serialized: Uint8Array): Account;
    static fromValuesArray(values: Uint8Array[]): Account;
    /**
     * This constructor assigns and validates the values.
     * Use the static factory methods to assist in creating an Account from varying data types.
     * undefined get assigned with the defaults present, but null args are retained as is
     */
    constructor(nonce?: bigint | null, balance?: bigint | null, storageRoot?: Uint8Array | null, codeHash?: Uint8Array | null, codeSize?: number | null, version?: number | null);
    private _validate;
    /**
     * Returns an array of Uint8Arrays of the raw bytes for the account, in order.
     */
    raw(): Uint8Array[];
    /**
     * Returns the RLP serialization of the account as a `Uint8Array`.
     */
    serialize(): Uint8Array;
    serializeWithPartialInfo(): Uint8Array;
    /**
     * Returns a `Boolean` determining if the account is a contract.
     */
    isContract(): boolean;
    /**
     * Returns a `Boolean` determining if the account is empty complying to the definition of
     * account emptiness in [EIP-161](https://eips.ethereum.org/EIPS/eip-161):
     * "An account is considered empty when it has no code and zero nonce and zero balance."
     */
    isEmpty(): boolean;
}
/**
 * Checks if the address is a valid. Accepts checksummed addresses too.
 */
export declare const isValidAddress: (hexAddress: string) => hexAddress is `0x${string}`;
/**
 * Returns a checksummed address.
 *
 * If an eip1191ChainId is provided, the chainId will be included in the checksum calculation. This
 * has the effect of checksummed addresses for one chain having invalid checksums for others.
 * For more details see [EIP-1191](https://eips.ethereum.org/EIPS/eip-1191).
 *
 * WARNING: Checksums with and without the chainId will differ and the EIP-1191 checksum is not
 * backwards compatible to the original widely adopted checksum format standard introduced in
 * [EIP-55](https://eips.ethereum.org/EIPS/eip-55), so this will break in existing applications.
 * Usage of this EIP is therefore discouraged unless you have a very targeted use case.
 */
export declare const toChecksumAddress: (hexAddress: string, eip1191ChainId?: BigIntLike) => PrefixedHexString;
/**
 * Checks if the address is a valid checksummed address.
 *
 * See toChecksumAddress' documentation for details about the eip1191ChainId parameter.
 */
export declare const isValidChecksumAddress: (hexAddress: string, eip1191ChainId?: BigIntLike) => boolean;
/**
 * Generates an address of a newly created contract.
 * @param from The address which is creating this new address
 * @param nonce The nonce of the from account
 */
export declare const generateAddress: (from: Uint8Array, nonce: Uint8Array) => Uint8Array;
/**
 * Generates an address for a contract created using CREATE2.
 * @param from The address which is creating this new address
 * @param salt A salt
 * @param initCode The init code of the contract being created
 */
export declare const generateAddress2: (from: Uint8Array, salt: Uint8Array, initCode: Uint8Array) => Uint8Array;
/**
 * Checks if the private key satisfies the rules of the curve secp256k1.
 */
export declare const isValidPrivate: (privateKey: Uint8Array) => boolean;
/**
 * Checks if the public key satisfies the rules of the curve secp256k1
 * and the requirements of Ethereum.
 * @param publicKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export declare const isValidPublic: (publicKey: Uint8Array, sanitize?: boolean) => boolean;
/**
 * Returns the ethereum address of a given public key.
 * Accepts "Ethereum public keys" and SEC1 encoded keys.
 * @param pubKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export declare const pubToAddress: (pubKey: Uint8Array, sanitize?: boolean) => Uint8Array;
export declare const publicToAddress: (pubKey: Uint8Array, sanitize?: boolean) => Uint8Array;
/**
 * Returns the ethereum public key of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export declare const privateToPublic: (privateKey: Uint8Array) => Uint8Array;
/**
 * Returns the ethereum address of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export declare const privateToAddress: (privateKey: Uint8Array) => Uint8Array;
/**
 * Converts a public key to the Ethereum format.
 */
export declare const importPublic: (publicKey: Uint8Array) => Uint8Array;
/**
 * Returns the zero address.
 */
export declare const zeroAddress: () => string;
/**
 * Checks if a given address is the zero address.
 */
export declare const isZeroAddress: (hexAddress: string) => boolean;
export declare function accountBodyFromSlim(body: AccountBodyBytes): Uint8Array[];
export declare function accountBodyToSlim(body: AccountBodyBytes): Uint8Array[];
/**
 * Converts a slim account (per snap protocol spec) to the RLP encoded version of the account
 * @param body Array of 4 Uint8Array-like items to represent the account
 * @returns RLP encoded version of the account
 */
export declare function accountBodyToRLP(body: AccountBodyBytes, couldBeSlim?: boolean): Uint8Array;
//# sourceMappingURL=account.d.ts.map