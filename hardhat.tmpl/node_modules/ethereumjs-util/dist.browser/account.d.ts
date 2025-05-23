/// <reference types="bn.js" />
/// <reference types="node" />
import { BN } from './externals';
import { BNLike, BufferLike } from './types';
export interface AccountData {
    nonce?: BNLike;
    balance?: BNLike;
    stateRoot?: BufferLike;
    codeHash?: BufferLike;
}
export declare class Account {
    nonce: BN;
    balance: BN;
    stateRoot: Buffer;
    codeHash: Buffer;
    static fromAccountData(accountData: AccountData): Account;
    static fromRlpSerializedAccount(serialized: Buffer): Account;
    static fromValuesArray(values: Buffer[]): Account;
    /**
     * This constructor assigns and validates the values.
     * Use the static factory methods to assist in creating an Account from varying data types.
     */
    constructor(nonce?: BN, balance?: BN, stateRoot?: Buffer, codeHash?: Buffer);
    private _validate;
    /**
     * Returns a Buffer Array of the raw Buffers for the account, in order.
     */
    raw(): Buffer[];
    /**
     * Returns the RLP serialization of the account as a `Buffer`.
     */
    serialize(): Buffer;
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
export declare const isValidAddress: (hexAddress: string) => boolean;
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
export declare const toChecksumAddress: (hexAddress: string, eip1191ChainId?: BNLike) => string;
/**
 * Checks if the address is a valid checksummed address.
 *
 * See toChecksumAddress' documentation for details about the eip1191ChainId parameter.
 */
export declare const isValidChecksumAddress: (hexAddress: string, eip1191ChainId?: BNLike) => boolean;
/**
 * Generates an address of a newly created contract.
 * @param from The address which is creating this new address
 * @param nonce The nonce of the from account
 */
export declare const generateAddress: (from: Buffer, nonce: Buffer) => Buffer;
/**
 * Generates an address for a contract created using CREATE2.
 * @param from The address which is creating this new address
 * @param salt A salt
 * @param initCode The init code of the contract being created
 */
export declare const generateAddress2: (from: Buffer, salt: Buffer, initCode: Buffer) => Buffer;
/**
 * Checks if the private key satisfies the rules of the curve secp256k1.
 */
export declare const isValidPrivate: (privateKey: Buffer) => boolean;
/**
 * Checks if the public key satisfies the rules of the curve secp256k1
 * and the requirements of Ethereum.
 * @param publicKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export declare const isValidPublic: (publicKey: Buffer, sanitize?: boolean) => boolean;
/**
 * Returns the ethereum address of a given public key.
 * Accepts "Ethereum public keys" and SEC1 encoded keys.
 * @param pubKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export declare const pubToAddress: (pubKey: Buffer, sanitize?: boolean) => Buffer;
export declare const publicToAddress: (pubKey: Buffer, sanitize?: boolean) => Buffer;
/**
 * Returns the ethereum public key of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export declare const privateToPublic: (privateKey: Buffer) => Buffer;
/**
 * Returns the ethereum address of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export declare const privateToAddress: (privateKey: Buffer) => Buffer;
/**
 * Converts a public key to the Ethereum format.
 */
export declare const importPublic: (publicKey: Buffer) => Buffer;
/**
 * Returns the zero address.
 */
export declare const zeroAddress: () => string;
/**
 * Checks if a given address is the zero address.
 */
export declare const isZeroAddress: (hexAddress: string) => boolean;
