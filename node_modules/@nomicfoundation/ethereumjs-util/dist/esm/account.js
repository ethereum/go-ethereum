import { RLP } from '@nomicfoundation/ethereumjs-rlp';
import { keccak256 } from 'ethereum-cryptography/keccak.js';
import { privateKeyVerify, publicKeyConvert, publicKeyCreate, publicKeyVerify, } from 'ethereum-cryptography/secp256k1';
import { bigIntToUnpaddedBytes, bytesToBigInt, bytesToHex, concatBytes, equalsBytes, hexToBytes, toBytes, utf8ToBytes, zeros, } from './bytes.js';
import { BIGINT_0, KECCAK256_NULL, KECCAK256_RLP } from './constants.js';
import { assertIsBytes, assertIsHexString, assertIsString } from './helpers.js';
import { stripHexPrefix } from './internal.js';
export class Account {
    /**
     * This constructor assigns and validates the values.
     * Use the static factory methods to assist in creating an Account from varying data types.
     */
    constructor(nonce = BIGINT_0, balance = BIGINT_0, storageRoot = KECCAK256_RLP, codeHash = KECCAK256_NULL) {
        this.nonce = nonce;
        this.balance = balance;
        this.storageRoot = storageRoot;
        this.codeHash = codeHash;
        this._validate();
    }
    static fromAccountData(accountData) {
        const { nonce, balance, storageRoot, codeHash } = accountData;
        return new Account(nonce !== undefined ? bytesToBigInt(toBytes(nonce)) : undefined, balance !== undefined ? bytesToBigInt(toBytes(balance)) : undefined, storageRoot !== undefined ? toBytes(storageRoot) : undefined, codeHash !== undefined ? toBytes(codeHash) : undefined);
    }
    static fromRlpSerializedAccount(serialized) {
        const values = RLP.decode(serialized);
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized account input. Must be array');
        }
        return this.fromValuesArray(values);
    }
    static fromValuesArray(values) {
        const [nonce, balance, storageRoot, codeHash] = values;
        return new Account(bytesToBigInt(nonce), bytesToBigInt(balance), storageRoot, codeHash);
    }
    _validate() {
        if (this.nonce < BIGINT_0) {
            throw new Error('nonce must be greater than zero');
        }
        if (this.balance < BIGINT_0) {
            throw new Error('balance must be greater than zero');
        }
        if (this.storageRoot.length !== 32) {
            throw new Error('storageRoot must have a length of 32');
        }
        if (this.codeHash.length !== 32) {
            throw new Error('codeHash must have a length of 32');
        }
    }
    /**
     * Returns an array of Uint8Arrays of the raw bytes for the account, in order.
     */
    raw() {
        return [
            bigIntToUnpaddedBytes(this.nonce),
            bigIntToUnpaddedBytes(this.balance),
            this.storageRoot,
            this.codeHash,
        ];
    }
    /**
     * Returns the RLP serialization of the account as a `Uint8Array`.
     */
    serialize() {
        return RLP.encode(this.raw());
    }
    /**
     * Returns a `Boolean` determining if the account is a contract.
     */
    isContract() {
        return !equalsBytes(this.codeHash, KECCAK256_NULL);
    }
    /**
     * Returns a `Boolean` determining if the account is empty complying to the definition of
     * account emptiness in [EIP-161](https://eips.ethereum.org/EIPS/eip-161):
     * "An account is considered empty when it has no code and zero nonce and zero balance."
     */
    isEmpty() {
        return (this.balance === BIGINT_0 &&
            this.nonce === BIGINT_0 &&
            equalsBytes(this.codeHash, KECCAK256_NULL));
    }
}
/**
 * Checks if the address is a valid. Accepts checksummed addresses too.
 */
export const isValidAddress = function (hexAddress) {
    try {
        assertIsString(hexAddress);
    }
    catch (e) {
        return false;
    }
    return /^0x[0-9a-fA-F]{40}$/.test(hexAddress);
};
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
export const toChecksumAddress = function (hexAddress, eip1191ChainId) {
    assertIsHexString(hexAddress);
    const address = stripHexPrefix(hexAddress).toLowerCase();
    let prefix = '';
    if (eip1191ChainId !== undefined) {
        const chainId = bytesToBigInt(toBytes(eip1191ChainId));
        prefix = chainId.toString() + '0x';
    }
    const bytes = utf8ToBytes(prefix + address);
    const hash = bytesToHex(keccak256(Buffer.from(bytes))).slice(2);
    let ret = '0x';
    for (let i = 0; i < address.length; i++) {
        if (parseInt(hash[i], 16) >= 8) {
            ret += address[i].toUpperCase();
        }
        else {
            ret += address[i];
        }
    }
    return ret;
};
/**
 * Checks if the address is a valid checksummed address.
 *
 * See toChecksumAddress' documentation for details about the eip1191ChainId parameter.
 */
export const isValidChecksumAddress = function (hexAddress, eip1191ChainId) {
    return isValidAddress(hexAddress) && toChecksumAddress(hexAddress, eip1191ChainId) === hexAddress;
};
/**
 * Generates an address of a newly created contract.
 * @param from The address which is creating this new address
 * @param nonce The nonce of the from account
 */
export const generateAddress = function (from, nonce) {
    assertIsBytes(from);
    assertIsBytes(nonce);
    if (bytesToBigInt(nonce) === BIGINT_0) {
        // in RLP we want to encode null in the case of zero nonce
        // read the RLP documentation for an answer if you dare
        return keccak256(Buffer.from(RLP.encode([from, Uint8Array.from([])]))).subarray(-20);
    }
    // Only take the lower 160bits of the hash
    return keccak256(Buffer.from(RLP.encode([from, nonce]))).subarray(-20);
};
/**
 * Generates an address for a contract created using CREATE2.
 * @param from The address which is creating this new address
 * @param salt A salt
 * @param initCode The init code of the contract being created
 */
export const generateAddress2 = function (from, salt, initCode) {
    assertIsBytes(from);
    assertIsBytes(salt);
    assertIsBytes(initCode);
    if (from.length !== 20) {
        throw new Error('Expected from to be of length 20');
    }
    if (salt.length !== 32) {
        throw new Error('Expected salt to be of length 32');
    }
    const address = keccak256(Buffer.from(concatBytes(hexToBytes('0xff'), from, salt, keccak256(Buffer.from(initCode)))));
    return address.subarray(-20);
};
/**
 * Checks if the private key satisfies the rules of the curve secp256k1.
 */
export const isValidPrivate = function (privateKey) {
    try {
        return privateKeyVerify(privateKey);
    }
    catch {
        return false;
    }
};
/**
 * Checks if the public key satisfies the rules of the curve secp256k1
 * and the requirements of Ethereum.
 * @param publicKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export const isValidPublic = function (publicKey, sanitize = false) {
    assertIsBytes(publicKey);
    if (publicKey.length === 64) {
        // Convert to SEC1 for secp256k1
        return publicKeyVerify(Buffer.concat([Buffer.from([4]), publicKey]));
    }
    if (!sanitize) {
        return false;
    }
    return publicKeyVerify(publicKey);
};
/**
 * Returns the ethereum address of a given public key.
 * Accepts "Ethereum public keys" and SEC1 encoded keys.
 * @param pubKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
export const pubToAddress = function (pubKey, sanitize = false) {
    assertIsBytes(pubKey);
    if (sanitize && pubKey.length !== 64) {
        pubKey = Buffer.from(publicKeyConvert(pubKey, false).slice(1));
    }
    if (pubKey.length !== 64) {
        throw new Error('Expected pubKey to be of length 64');
    }
    // Only take the lower 160bits of the hash
    return Buffer.from(keccak256(Buffer.from(pubKey))).slice(-20);
};
export const publicToAddress = pubToAddress;
/**
 * Returns the ethereum public key of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export const privateToPublic = function (privateKey) {
    assertIsBytes(privateKey);
    // skip the type flag and use the X, Y points
    return Buffer.from(publicKeyCreate(privateKey, false)).slice(1);
};
/**
 * Returns the ethereum address of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
export const privateToAddress = function (privateKey) {
    return publicToAddress(privateToPublic(privateKey));
};
/**
 * Converts a public key to the Ethereum format.
 */
export const importPublic = function (publicKey) {
    assertIsBytes(publicKey);
    if (publicKey.length !== 64) {
        publicKey = Buffer.from(publicKeyConvert(publicKey, false).slice(1));
    }
    return publicKey;
};
/**
 * Returns the zero address.
 */
export const zeroAddress = function () {
    const addressLength = 20;
    const addr = zeros(addressLength);
    return bytesToHex(addr);
};
/**
 * Checks if a given address is the zero address.
 */
export const isZeroAddress = function (hexAddress) {
    try {
        assertIsString(hexAddress);
    }
    catch (e) {
        return false;
    }
    const zeroAddr = zeroAddress();
    return zeroAddr === hexAddress;
};
export function accountBodyFromSlim(body) {
    const [nonce, balance, storageRoot, codeHash] = body;
    return [
        nonce,
        balance,
        storageRoot.length === 0 ? KECCAK256_RLP : storageRoot,
        codeHash.length === 0 ? KECCAK256_NULL : codeHash,
    ];
}
const emptyUint8Arr = new Uint8Array(0);
export function accountBodyToSlim(body) {
    const [nonce, balance, storageRoot, codeHash] = body;
    return [
        nonce,
        balance,
        equalsBytes(storageRoot, KECCAK256_RLP) ? emptyUint8Arr : storageRoot,
        equalsBytes(codeHash, KECCAK256_NULL) ? emptyUint8Arr : codeHash,
    ];
}
/**
 * Converts a slim account (per snap protocol spec) to the RLP encoded version of the account
 * @param body Array of 4 Uint8Array-like items to represent the account
 * @returns RLP encoded version of the account
 */
export function accountBodyToRLP(body, couldBeSlim = true) {
    const accountBody = couldBeSlim ? accountBodyFromSlim(body) : body;
    return RLP.encode(accountBody);
}
//# sourceMappingURL=account.js.map