"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.accountBodyToRLP = exports.accountBodyToSlim = exports.accountBodyFromSlim = exports.isZeroAddress = exports.zeroAddress = exports.importPublic = exports.privateToAddress = exports.privateToPublic = exports.publicToAddress = exports.pubToAddress = exports.isValidPublic = exports.isValidPrivate = exports.generateAddress2 = exports.generateAddress = exports.isValidChecksumAddress = exports.toChecksumAddress = exports.isValidAddress = exports.Account = void 0;
const ethereumjs_rlp_1 = require("@nomicfoundation/ethereumjs-rlp");
const keccak_js_1 = require("ethereum-cryptography/keccak.js");
const secp256k1_1 = require("ethereum-cryptography/secp256k1");
const bytes_js_1 = require("./bytes.js");
const constants_js_1 = require("./constants.js");
const helpers_js_1 = require("./helpers.js");
const internal_js_1 = require("./internal.js");
class Account {
    /**
     * This constructor assigns and validates the values.
     * Use the static factory methods to assist in creating an Account from varying data types.
     */
    constructor(nonce = constants_js_1.BIGINT_0, balance = constants_js_1.BIGINT_0, storageRoot = constants_js_1.KECCAK256_RLP, codeHash = constants_js_1.KECCAK256_NULL) {
        this.nonce = nonce;
        this.balance = balance;
        this.storageRoot = storageRoot;
        this.codeHash = codeHash;
        this._validate();
    }
    static fromAccountData(accountData) {
        const { nonce, balance, storageRoot, codeHash } = accountData;
        return new Account(nonce !== undefined ? (0, bytes_js_1.bytesToBigInt)((0, bytes_js_1.toBytes)(nonce)) : undefined, balance !== undefined ? (0, bytes_js_1.bytesToBigInt)((0, bytes_js_1.toBytes)(balance)) : undefined, storageRoot !== undefined ? (0, bytes_js_1.toBytes)(storageRoot) : undefined, codeHash !== undefined ? (0, bytes_js_1.toBytes)(codeHash) : undefined);
    }
    static fromRlpSerializedAccount(serialized) {
        const values = ethereumjs_rlp_1.RLP.decode(serialized);
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized account input. Must be array');
        }
        return this.fromValuesArray(values);
    }
    static fromValuesArray(values) {
        const [nonce, balance, storageRoot, codeHash] = values;
        return new Account((0, bytes_js_1.bytesToBigInt)(nonce), (0, bytes_js_1.bytesToBigInt)(balance), storageRoot, codeHash);
    }
    _validate() {
        if (this.nonce < constants_js_1.BIGINT_0) {
            throw new Error('nonce must be greater than zero');
        }
        if (this.balance < constants_js_1.BIGINT_0) {
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
            (0, bytes_js_1.bigIntToUnpaddedBytes)(this.nonce),
            (0, bytes_js_1.bigIntToUnpaddedBytes)(this.balance),
            this.storageRoot,
            this.codeHash,
        ];
    }
    /**
     * Returns the RLP serialization of the account as a `Uint8Array`.
     */
    serialize() {
        return ethereumjs_rlp_1.RLP.encode(this.raw());
    }
    /**
     * Returns a `Boolean` determining if the account is a contract.
     */
    isContract() {
        return !(0, bytes_js_1.equalsBytes)(this.codeHash, constants_js_1.KECCAK256_NULL);
    }
    /**
     * Returns a `Boolean` determining if the account is empty complying to the definition of
     * account emptiness in [EIP-161](https://eips.ethereum.org/EIPS/eip-161):
     * "An account is considered empty when it has no code and zero nonce and zero balance."
     */
    isEmpty() {
        return (this.balance === constants_js_1.BIGINT_0 &&
            this.nonce === constants_js_1.BIGINT_0 &&
            (0, bytes_js_1.equalsBytes)(this.codeHash, constants_js_1.KECCAK256_NULL));
    }
}
exports.Account = Account;
/**
 * Checks if the address is a valid. Accepts checksummed addresses too.
 */
const isValidAddress = function (hexAddress) {
    try {
        (0, helpers_js_1.assertIsString)(hexAddress);
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
    (0, helpers_js_1.assertIsHexString)(hexAddress);
    const address = (0, internal_js_1.stripHexPrefix)(hexAddress).toLowerCase();
    let prefix = '';
    if (eip1191ChainId !== undefined) {
        const chainId = (0, bytes_js_1.bytesToBigInt)((0, bytes_js_1.toBytes)(eip1191ChainId));
        prefix = chainId.toString() + '0x';
    }
    const bytes = (0, bytes_js_1.utf8ToBytes)(prefix + address);
    const hash = (0, bytes_js_1.bytesToHex)((0, keccak_js_1.keccak256)(Buffer.from(bytes))).slice(2);
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
    (0, helpers_js_1.assertIsBytes)(from);
    (0, helpers_js_1.assertIsBytes)(nonce);
    if ((0, bytes_js_1.bytesToBigInt)(nonce) === constants_js_1.BIGINT_0) {
        // in RLP we want to encode null in the case of zero nonce
        // read the RLP documentation for an answer if you dare
        return (0, keccak_js_1.keccak256)(Buffer.from(ethereumjs_rlp_1.RLP.encode([from, Uint8Array.from([])]))).subarray(-20);
    }
    // Only take the lower 160bits of the hash
    return (0, keccak_js_1.keccak256)(Buffer.from(ethereumjs_rlp_1.RLP.encode([from, nonce]))).subarray(-20);
};
exports.generateAddress = generateAddress;
/**
 * Generates an address for a contract created using CREATE2.
 * @param from The address which is creating this new address
 * @param salt A salt
 * @param initCode The init code of the contract being created
 */
const generateAddress2 = function (from, salt, initCode) {
    (0, helpers_js_1.assertIsBytes)(from);
    (0, helpers_js_1.assertIsBytes)(salt);
    (0, helpers_js_1.assertIsBytes)(initCode);
    if (from.length !== 20) {
        throw new Error('Expected from to be of length 20');
    }
    if (salt.length !== 32) {
        throw new Error('Expected salt to be of length 32');
    }
    const address = (0, keccak_js_1.keccak256)(Buffer.from((0, bytes_js_1.concatBytes)((0, bytes_js_1.hexToBytes)('0xff'), from, salt, (0, keccak_js_1.keccak256)(Buffer.from(initCode)))));
    return address.subarray(-20);
};
exports.generateAddress2 = generateAddress2;
/**
 * Checks if the private key satisfies the rules of the curve secp256k1.
 */
const isValidPrivate = function (privateKey) {
    try {
        return (0, secp256k1_1.privateKeyVerify)(privateKey);
    }
    catch {
        return false;
    }
};
exports.isValidPrivate = isValidPrivate;
/**
 * Checks if the public key satisfies the rules of the curve secp256k1
 * and the requirements of Ethereum.
 * @param publicKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
const isValidPublic = function (publicKey, sanitize = false) {
    (0, helpers_js_1.assertIsBytes)(publicKey);
    if (publicKey.length === 64) {
        // Convert to SEC1 for secp256k1
        return (0, secp256k1_1.publicKeyVerify)(Buffer.concat([Buffer.from([4]), publicKey]));
    }
    if (!sanitize) {
        return false;
    }
    return (0, secp256k1_1.publicKeyVerify)(publicKey);
};
exports.isValidPublic = isValidPublic;
/**
 * Returns the ethereum address of a given public key.
 * Accepts "Ethereum public keys" and SEC1 encoded keys.
 * @param pubKey The two points of an uncompressed key, unless sanitize is enabled
 * @param sanitize Accept public keys in other formats
 */
const pubToAddress = function (pubKey, sanitize = false) {
    (0, helpers_js_1.assertIsBytes)(pubKey);
    if (sanitize && pubKey.length !== 64) {
        pubKey = Buffer.from((0, secp256k1_1.publicKeyConvert)(pubKey, false).slice(1));
    }
    if (pubKey.length !== 64) {
        throw new Error('Expected pubKey to be of length 64');
    }
    // Only take the lower 160bits of the hash
    return Buffer.from((0, keccak_js_1.keccak256)(Buffer.from(pubKey))).slice(-20);
};
exports.pubToAddress = pubToAddress;
exports.publicToAddress = exports.pubToAddress;
/**
 * Returns the ethereum public key of a given private key.
 * @param privateKey A private key must be 256 bits wide
 */
const privateToPublic = function (privateKey) {
    (0, helpers_js_1.assertIsBytes)(privateKey);
    // skip the type flag and use the X, Y points
    return Buffer.from((0, secp256k1_1.publicKeyCreate)(privateKey, false)).slice(1);
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
    (0, helpers_js_1.assertIsBytes)(publicKey);
    if (publicKey.length !== 64) {
        publicKey = Buffer.from((0, secp256k1_1.publicKeyConvert)(publicKey, false).slice(1));
    }
    return publicKey;
};
exports.importPublic = importPublic;
/**
 * Returns the zero address.
 */
const zeroAddress = function () {
    const addressLength = 20;
    const addr = (0, bytes_js_1.zeros)(addressLength);
    return (0, bytes_js_1.bytesToHex)(addr);
};
exports.zeroAddress = zeroAddress;
/**
 * Checks if a given address is the zero address.
 */
const isZeroAddress = function (hexAddress) {
    try {
        (0, helpers_js_1.assertIsString)(hexAddress);
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
        storageRoot.length === 0 ? constants_js_1.KECCAK256_RLP : storageRoot,
        codeHash.length === 0 ? constants_js_1.KECCAK256_NULL : codeHash,
    ];
}
exports.accountBodyFromSlim = accountBodyFromSlim;
const emptyUint8Arr = new Uint8Array(0);
function accountBodyToSlim(body) {
    const [nonce, balance, storageRoot, codeHash] = body;
    return [
        nonce,
        balance,
        (0, bytes_js_1.equalsBytes)(storageRoot, constants_js_1.KECCAK256_RLP) ? emptyUint8Arr : storageRoot,
        (0, bytes_js_1.equalsBytes)(codeHash, constants_js_1.KECCAK256_NULL) ? emptyUint8Arr : codeHash,
    ];
}
exports.accountBodyToSlim = accountBodyToSlim;
/**
 * Converts a slim account (per snap protocol spec) to the RLP encoded version of the account
 * @param body Array of 4 Uint8Array-like items to represent the account
 * @returns RLP encoded version of the account
 */
function accountBodyToRLP(body, couldBeSlim = true) {
    const accountBody = couldBeSlim ? accountBodyFromSlim(body) : body;
    return ethereumjs_rlp_1.RLP.encode(accountBody);
}
exports.accountBodyToRLP = accountBodyToRLP;
//# sourceMappingURL=account.js.map