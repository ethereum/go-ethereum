"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.accountBodyToRLP = exports.accountBodyToSlim = exports.accountBodyFromSlim = exports.isZeroAddress = exports.zeroAddress = exports.importPublic = exports.privateToAddress = exports.privateToPublic = exports.publicToAddress = exports.pubToAddress = exports.isValidPublic = exports.isValidPrivate = exports.generateAddress2 = exports.generateAddress = exports.isValidChecksumAddress = exports.toChecksumAddress = exports.isValidAddress = exports.Account = void 0;
const rlp_1 = require("@ethereumjs/rlp");
const keccak_1 = require("ethereum-cryptography/keccak");
const secp256k1_1 = require("ethereum-cryptography/secp256k1");
const utils_1 = require("ethereum-cryptography/utils");
const bytes_1 = require("./bytes");
const constants_1 = require("./constants");
const helpers_1 = require("./helpers");
const internal_1 = require("./internal");
const _0n = BigInt(0);
class Account {
    /**
     * This constructor assigns and validates the values.
     * Use the static factory methods to assist in creating an Account from varying data types.
     */
    constructor(nonce = _0n, balance = _0n, storageRoot = constants_1.KECCAK256_RLP, codeHash = constants_1.KECCAK256_NULL) {
        this.nonce = nonce;
        this.balance = balance;
        this.storageRoot = storageRoot;
        this.codeHash = codeHash;
        this._validate();
    }
    static fromAccountData(accountData) {
        const { nonce, balance, storageRoot, codeHash } = accountData;
        return new Account(nonce !== undefined ? (0, bytes_1.bufferToBigInt)((0, bytes_1.toBuffer)(nonce)) : undefined, balance !== undefined ? (0, bytes_1.bufferToBigInt)((0, bytes_1.toBuffer)(balance)) : undefined, storageRoot !== undefined ? (0, bytes_1.toBuffer)(storageRoot) : undefined, codeHash !== undefined ? (0, bytes_1.toBuffer)(codeHash) : undefined);
    }
    static fromRlpSerializedAccount(serialized) {
        const values = (0, bytes_1.arrToBufArr)(rlp_1.RLP.decode(Uint8Array.from(serialized)));
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized account input. Must be array');
        }
        return this.fromValuesArray(values);
    }
    static fromValuesArray(values) {
        const [nonce, balance, storageRoot, codeHash] = values;
        return new Account((0, bytes_1.bufferToBigInt)(nonce), (0, bytes_1.bufferToBigInt)(balance), storageRoot, codeHash);
    }
    _validate() {
        if (this.nonce < _0n) {
            throw new Error('nonce must be greater than zero');
        }
        if (this.balance < _0n) {
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
     * Returns a Buffer Array of the raw Buffers for the account, in order.
     */
    raw() {
        return [
            (0, bytes_1.bigIntToUnpaddedBuffer)(this.nonce),
            (0, bytes_1.bigIntToUnpaddedBuffer)(this.balance),
            this.storageRoot,
            this.codeHash,
        ];
    }
    /**
     * Returns the RLP serialization of the account as a `Buffer`.
     */
    serialize() {
        return Buffer.from(rlp_1.RLP.encode((0, bytes_1.bufArrToArr)(this.raw())));
    }
    /**
     * Returns a `Boolean` determining if the account is a contract.
     */
    isContract() {
        return !this.codeHash.equals(constants_1.KECCAK256_NULL);
    }
    /**
     * Returns a `Boolean` determining if the account is empty complying to the definition of
     * account emptiness in [EIP-161](https://eips.ethereum.org/EIPS/eip-161):
     * "An account is considered empty when it has no code and zero nonce and zero balance."
     */
    isEmpty() {
        return this.balance === _0n && this.nonce === _0n && this.codeHash.equals(constants_1.KECCAK256_NULL);
    }
}
exports.Account = Account;
/**
 * Checks if the address is a valid. Accepts checksummed addresses too.
 */
const isValidAddress = function (hexAddress) {
    try {
        (0, helpers_1.assertIsString)(hexAddress);
    }
    catch (e) {
        return false;
    }
    return /^0x[0-9a-fA-F]{40}$/.test(hexAddress);
};
exports.isValidAddress = isValidAddress;
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
const toChecksumAddress = function (hexAddress, eip1191ChainId) {
    (0, helpers_1.assertIsHexString)(hexAddress);
    const address = (0, internal_1.stripHexPrefix)(hexAddress).toLowerCase();
    let prefix = '';
    if (eip1191ChainId !== undefined) {
        const chainId = (0, bytes_1.bufferToBigInt)((0, bytes_1.toBuffer)(eip1191ChainId));
        prefix = chainId.toString() + '0x';
    }
    const buf = Buffer.from(prefix + address, 'utf8');
    const hash = (0, utils_1.bytesToHex)((0, keccak_1.keccak256)(buf));
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
exports.toChecksumAddress = toChecksumAddress;
/**
 * Checks if the address is a valid checksummed address.
 *
 * See toChecksumAddress' documentation for details about the eip1191ChainId parameter.
 */
const isValidChecksumAddress = function (hexAddress, eip1191ChainId) {
    return (0, exports.isValidAddress)(hexAddress) && (0, exports.toChecksumAddress)(hexAddress, eip1191ChainId) === hexAddress;
};
exports.isValidChecksumAddress = isValidChecksumAddress;
/**
 * Generates an address of a newly created contract.
 * @param from The address which is creating this new address
 * @param nonce The nonce of the from account
 */
const generateAddress = function (from, nonce) {
    (0, helpers_1.assertIsBuffer)(from);
    (0, helpers_1.assertIsBuffer)(nonce);
    if ((0, bytes_1.bufferToBigInt)(nonce) === BigInt(0)) {
        // in RLP we want to encode null in the case of zero nonce
        // read the RLP documentation for an answer if you dare
        return Buffer.from((0, keccak_1.keccak256)(rlp_1.RLP.encode((0, bytes_1.bufArrToArr)([from, null])))).slice(-20);
    }
    // Only take the lower 160bits of the hash
    return Buffer.from((0, keccak_1.keccak256)(rlp_1.RLP.encode((0, bytes_1.bufArrToArr)([from, nonce])))).slice(-20);
};
exports.generateAddress = generateAddress;
/**
 * Generates an address for a contract created using CREATE2.
 * @param from The address which is creating this new address
 * @param salt A salt
 * @param initCode The init code of the contract being created
 */
const generateAddress2 = function (from, salt, initCode) {
    (0, helpers_1.assertIsBuffer)(from);
    (0, helpers_1.assertIsBuffer)(salt);
    (0, helpers_1.assertIsBuffer)(initCode);
    if (from.length !== 20) {
        throw new Error('Expected from to be of length 20');
    }
    if (salt.length !== 32) {
        throw new Error('Expected salt to be of length 32');
    }
    const address = (0, keccak_1.keccak256)(Buffer.concat([Buffer.from('ff', 'hex'), from, salt, (0, keccak_1.keccak256)(initCode)]));
    return (0, bytes_1.toBuffer)(address).slice(-20);
};
exports.generateAddress2 = generateAddress2;
/**
 * Checks if the private key satisfies the rules of the curve secp256k1.
 */
const isValidPrivate = function (privateKey) {
    return secp256k1_1.secp256k1.utils.isValidPrivateKey(privateKey);
};
exports.isValidPrivate = isValidPrivate;
/**
 * Checks if the public key satisfies the rules of the curve secp256k1
 * and the requirements of Ethereum.
 * @param publicKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
const isValidPublic = function (publicKey, sanitize = false) {
    (0, helpers_1.assertIsBuffer)(publicKey);
    if (publicKey.length === 64) {
        // Convert to SEC1 for secp256k1
        // Automatically checks whether point is on curve
        try {
            secp256k1_1.secp256k1.ProjectivePoint.fromHex(Buffer.concat([Buffer.from([4]), publicKey]));
            return true;
        }
        catch (e) {
            return false;
        }
    }
    if (!sanitize) {
        return false;
    }
    try {
        secp256k1_1.secp256k1.ProjectivePoint.fromHex(publicKey);
        return true;
    }
    catch (e) {
        return false;
    }
};
exports.isValidPublic = isValidPublic;
/**
 * Returns the ethereum address of a given public key.
 * Accepts "Ethereum public keys" and SEC1 encoded keys.
 * @param pubKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
const pubToAddress = function (pubKey, sanitize = false) {
    (0, helpers_1.assertIsBuffer)(pubKey);
    if (sanitize && pubKey.length !== 64) {
        pubKey = Buffer.from(secp256k1_1.secp256k1.ProjectivePoint.fromHex(pubKey).toRawBytes(false).slice(1));
    }
    if (pubKey.length !== 64) {
        throw new Error('Expected pubKey to be of length 64');
    }
    // Only take the lower 160bits of the hash
    return Buffer.from((0, keccak_1.keccak256)(pubKey)).slice(-20);
};
exports.pubToAddress = pubToAddress;
exports.publicToAddress = exports.pubToAddress;
/**
 * Returns the ethereum public key of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
const privateToPublic = function (privateKey) {
    (0, helpers_1.assertIsBuffer)(privateKey);
    // skip the type flag and use the X, Y points
    return Buffer.from(secp256k1_1.secp256k1.ProjectivePoint.fromPrivateKey(privateKey).toRawBytes(false).slice(1));
};
exports.privateToPublic = privateToPublic;
/**
 * Returns the ethereum address of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
const privateToAddress = function (privateKey) {
    return (0, exports.publicToAddress)((0, exports.privateToPublic)(privateKey));
};
exports.privateToAddress = privateToAddress;
/**
 * Converts a public key to the Ethereum format.
 */
const importPublic = function (publicKey) {
    (0, helpers_1.assertIsBuffer)(publicKey);
    if (publicKey.length !== 64) {
        publicKey = Buffer.from(secp256k1_1.secp256k1.ProjectivePoint.fromHex(publicKey).toRawBytes(false).slice(1));
    }
    return publicKey;
};
exports.importPublic = importPublic;
/**
 * Returns the zero address.
 */
const zeroAddress = function () {
    const addressLength = 20;
    const addr = (0, bytes_1.zeros)(addressLength);
    return (0, bytes_1.bufferToHex)(addr);
};
exports.zeroAddress = zeroAddress;
/**
 * Checks if a given address is the zero address.
 */
const isZeroAddress = function (hexAddress) {
    try {
        (0, helpers_1.assertIsString)(hexAddress);
    }
    catch (e) {
        return false;
    }
    const zeroAddr = (0, exports.zeroAddress)();
    return zeroAddr === hexAddress;
};
exports.isZeroAddress = isZeroAddress;
function accountBodyFromSlim(body) {
    const [nonce, balance, storageRoot, codeHash] = body;
    return [
        nonce,
        balance,
        (0, bytes_1.arrToBufArr)(storageRoot).length === 0 ? constants_1.KECCAK256_RLP : storageRoot,
        (0, bytes_1.arrToBufArr)(codeHash).length === 0 ? constants_1.KECCAK256_NULL : codeHash,
    ];
}
exports.accountBodyFromSlim = accountBodyFromSlim;
const emptyUint8Arr = new Uint8Array(0);
function accountBodyToSlim(body) {
    const [nonce, balance, storageRoot, codeHash] = body;
    return [
        nonce,
        balance,
        (0, bytes_1.arrToBufArr)(storageRoot).equals(constants_1.KECCAK256_RLP) ? emptyUint8Arr : storageRoot,
        (0, bytes_1.arrToBufArr)(codeHash).equals(constants_1.KECCAK256_NULL) ? emptyUint8Arr : codeHash,
    ];
}
exports.accountBodyToSlim = accountBodyToSlim;
/**
 * Converts a slim account (per snap protocol spec) to the RLP encoded version of the account
 * @param body Array of 4 Buffer-like items to represent the account
 * @returns RLP encoded version of the account
 */
function accountBodyToRLP(body, couldBeSlim = true) {
    const accountBody = couldBeSlim ? accountBodyFromSlim(body) : body;
    return (0, bytes_1.arrToBufArr)(rlp_1.RLP.encode(accountBody));
}
exports.accountBodyToRLP = accountBodyToRLP;
//# sourceMappingURL=account.js.map