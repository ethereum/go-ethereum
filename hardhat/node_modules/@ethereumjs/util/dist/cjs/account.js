"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.accountBodyToRLP = exports.accountBodyToSlim = exports.accountBodyFromSlim = exports.isZeroAddress = exports.zeroAddress = exports.importPublic = exports.privateToAddress = exports.privateToPublic = exports.publicToAddress = exports.pubToAddress = exports.isValidPublic = exports.isValidPrivate = exports.generateAddress2 = exports.generateAddress = exports.isValidChecksumAddress = exports.toChecksumAddress = exports.isValidAddress = exports.Account = void 0;
const rlp_1 = require("@ethereumjs/rlp");
const keccak_js_1 = require("ethereum-cryptography/keccak.js");
const secp256k1_js_1 = require("ethereum-cryptography/secp256k1.js");
const bytes_js_1 = require("./bytes.js");
const constants_js_1 = require("./constants.js");
const helpers_js_1 = require("./helpers.js");
const internal_js_1 = require("./internal.js");
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
class Account {
    /**
     * This constructor assigns and validates the values.
     * Use the static factory methods to assist in creating an Account from varying data types.
     * undefined get assigned with the defaults present, but null args are retained as is
     */
    constructor(nonce = constants_js_1.BIGINT_0, balance = constants_js_1.BIGINT_0, storageRoot = constants_js_1.KECCAK256_RLP, codeHash = constants_js_1.KECCAK256_NULL, codeSize = null, version = 0) {
        this._nonce = null;
        this._balance = null;
        this._storageRoot = null;
        this._codeHash = null;
        // codeSize and version is separately stored in VKT
        this._codeSize = null;
        this._version = null;
        this._nonce = nonce;
        this._balance = balance;
        this._storageRoot = storageRoot;
        this._codeHash = codeHash;
        if (codeSize === null && codeHash !== null && !this.isContract()) {
            codeSize = 0;
        }
        this._codeSize = codeSize;
        this._version = version;
        this._validate();
    }
    get version() {
        if (this._version !== null) {
            return this._version;
        }
        else {
            throw Error(`version=${this._version} not loaded`);
        }
    }
    set version(_version) {
        this._version = _version;
    }
    get nonce() {
        if (this._nonce !== null) {
            return this._nonce;
        }
        else {
            throw Error(`nonce=${this._nonce} not loaded`);
        }
    }
    set nonce(_nonce) {
        this._nonce = _nonce;
    }
    get balance() {
        if (this._balance !== null) {
            return this._balance;
        }
        else {
            throw Error(`balance=${this._balance} not loaded`);
        }
    }
    set balance(_balance) {
        this._balance = _balance;
    }
    get storageRoot() {
        if (this._storageRoot !== null) {
            return this._storageRoot;
        }
        else {
            throw Error(`storageRoot=${this._storageRoot} not loaded`);
        }
    }
    set storageRoot(_storageRoot) {
        this._storageRoot = _storageRoot;
    }
    get codeHash() {
        if (this._codeHash !== null) {
            return this._codeHash;
        }
        else {
            throw Error(`codeHash=${this._codeHash} not loaded`);
        }
    }
    set codeHash(_codeHash) {
        this._codeHash = _codeHash;
    }
    get codeSize() {
        if (this._codeSize !== null) {
            return this._codeSize;
        }
        else {
            throw Error(`codeHash=${this._codeSize} not loaded`);
        }
    }
    set codeSize(_codeSize) {
        this._codeSize = _codeSize;
    }
    static fromAccountData(accountData) {
        const { nonce, balance, storageRoot, codeHash } = accountData;
        if (nonce === null || balance === null || storageRoot === null || codeHash === null) {
            throw Error(`Partial fields not supported in fromAccountData`);
        }
        return new Account(nonce !== undefined ? (0, bytes_js_1.bytesToBigInt)((0, bytes_js_1.toBytes)(nonce)) : undefined, balance !== undefined ? (0, bytes_js_1.bytesToBigInt)((0, bytes_js_1.toBytes)(balance)) : undefined, storageRoot !== undefined ? (0, bytes_js_1.toBytes)(storageRoot) : undefined, codeHash !== undefined ? (0, bytes_js_1.toBytes)(codeHash) : undefined);
    }
    static fromPartialAccountData(partialAccountData) {
        const { nonce, balance, storageRoot, codeHash, codeSize, version } = partialAccountData;
        if (nonce === null &&
            balance === null &&
            storageRoot === null &&
            codeHash === null &&
            codeSize === null &&
            version === null) {
            throw Error(`All partial fields null`);
        }
        return new Account(nonce !== undefined && nonce !== null ? (0, bytes_js_1.bytesToBigInt)((0, bytes_js_1.toBytes)(nonce)) : nonce, balance !== undefined && balance !== null ? (0, bytes_js_1.bytesToBigInt)((0, bytes_js_1.toBytes)(balance)) : balance, storageRoot !== undefined && storageRoot !== null ? (0, bytes_js_1.toBytes)(storageRoot) : storageRoot, codeHash !== undefined && codeHash !== null ? (0, bytes_js_1.toBytes)(codeHash) : codeHash, codeSize !== undefined && codeSize !== null ? (0, bytes_js_1.bytesToInt)((0, bytes_js_1.toBytes)(codeSize)) : codeSize, version !== undefined && version !== null ? (0, bytes_js_1.bytesToInt)((0, bytes_js_1.toBytes)(version)) : version);
    }
    static fromRlpSerializedAccount(serialized) {
        const values = rlp_1.RLP.decode(serialized);
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized account input. Must be array');
        }
        return this.fromValuesArray(values);
    }
    static fromRlpSerializedPartialAccount(serialized) {
        const values = rlp_1.RLP.decode(serialized);
        if (!Array.isArray(values)) {
            throw new Error('Invalid serialized account input. Must be array');
        }
        let nonce = null;
        if (!Array.isArray(values[0])) {
            throw new Error('Invalid partial nonce encoding. Must be array');
        }
        else {
            const isNotNullIndicator = (0, bytes_js_1.bytesToInt)(values[0][0]);
            if (isNotNullIndicator !== 0 && isNotNullIndicator !== 1) {
                throw new Error(`Invalid isNullIndicator=${isNotNullIndicator} for nonce`);
            }
            if (isNotNullIndicator === 1) {
                nonce = (0, bytes_js_1.bytesToBigInt)(values[0][1]);
            }
        }
        let balance = null;
        if (!Array.isArray(values[1])) {
            throw new Error('Invalid partial balance encoding. Must be array');
        }
        else {
            const isNotNullIndicator = (0, bytes_js_1.bytesToInt)(values[1][0]);
            if (isNotNullIndicator !== 0 && isNotNullIndicator !== 1) {
                throw new Error(`Invalid isNullIndicator=${isNotNullIndicator} for balance`);
            }
            if (isNotNullIndicator === 1) {
                balance = (0, bytes_js_1.bytesToBigInt)(values[1][1]);
            }
        }
        let storageRoot = null;
        if (!Array.isArray(values[2])) {
            throw new Error('Invalid partial storageRoot encoding. Must be array');
        }
        else {
            const isNotNullIndicator = (0, bytes_js_1.bytesToInt)(values[2][0]);
            if (isNotNullIndicator !== 0 && isNotNullIndicator !== 1) {
                throw new Error(`Invalid isNullIndicator=${isNotNullIndicator} for storageRoot`);
            }
            if (isNotNullIndicator === 1) {
                storageRoot = values[2][1];
            }
        }
        let codeHash = null;
        if (!Array.isArray(values[3])) {
            throw new Error('Invalid partial codeHash encoding. Must be array');
        }
        else {
            const isNotNullIndicator = (0, bytes_js_1.bytesToInt)(values[3][0]);
            if (isNotNullIndicator !== 0 && isNotNullIndicator !== 1) {
                throw new Error(`Invalid isNullIndicator=${isNotNullIndicator} for codeHash`);
            }
            if (isNotNullIndicator === 1) {
                codeHash = values[3][1];
            }
        }
        let codeSize = null;
        if (!Array.isArray(values[4])) {
            throw new Error('Invalid partial codeSize encoding. Must be array');
        }
        else {
            const isNotNullIndicator = (0, bytes_js_1.bytesToInt)(values[4][0]);
            if (isNotNullIndicator !== 0 && isNotNullIndicator !== 1) {
                throw new Error(`Invalid isNullIndicator=${isNotNullIndicator} for codeSize`);
            }
            if (isNotNullIndicator === 1) {
                codeSize = (0, bytes_js_1.bytesToInt)(values[4][1]);
            }
        }
        let version = null;
        if (!Array.isArray(values[5])) {
            throw new Error('Invalid partial version encoding. Must be array');
        }
        else {
            const isNotNullIndicator = (0, bytes_js_1.bytesToInt)(values[5][0]);
            if (isNotNullIndicator !== 0 && isNotNullIndicator !== 1) {
                throw new Error(`Invalid isNullIndicator=${isNotNullIndicator} for version`);
            }
            if (isNotNullIndicator === 1) {
                version = (0, bytes_js_1.bytesToInt)(values[5][1]);
            }
        }
        return this.fromPartialAccountData({ balance, nonce, storageRoot, codeHash, codeSize, version });
    }
    static fromValuesArray(values) {
        const [nonce, balance, storageRoot, codeHash] = values;
        return new Account((0, bytes_js_1.bytesToBigInt)(nonce), (0, bytes_js_1.bytesToBigInt)(balance), storageRoot, codeHash);
    }
    _validate() {
        if (this._nonce !== null && this._nonce < constants_js_1.BIGINT_0) {
            throw new Error('nonce must be greater than zero');
        }
        if (this._balance !== null && this._balance < constants_js_1.BIGINT_0) {
            throw new Error('balance must be greater than zero');
        }
        if (this._storageRoot !== null && this._storageRoot.length !== 32) {
            throw new Error('storageRoot must have a length of 32');
        }
        if (this._codeHash !== null && this._codeHash.length !== 32) {
            throw new Error('codeHash must have a length of 32');
        }
        if (this._codeSize !== null && this._codeSize < constants_js_1.BIGINT_0) {
            throw new Error('codeSize must be greater than zero');
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
        return rlp_1.RLP.encode(this.raw());
    }
    serializeWithPartialInfo() {
        const partialData = [];
        const zeroEncoded = (0, bytes_js_1.intToUnpaddedBytes)(0);
        const oneEncoded = (0, bytes_js_1.intToUnpaddedBytes)(1);
        if (this._nonce !== null) {
            partialData.push([oneEncoded, (0, bytes_js_1.bigIntToUnpaddedBytes)(this._nonce)]);
        }
        else {
            partialData.push([zeroEncoded]);
        }
        if (this._balance !== null) {
            partialData.push([oneEncoded, (0, bytes_js_1.bigIntToUnpaddedBytes)(this._balance)]);
        }
        else {
            partialData.push([zeroEncoded]);
        }
        if (this._storageRoot !== null) {
            partialData.push([oneEncoded, this._storageRoot]);
        }
        else {
            partialData.push([zeroEncoded]);
        }
        if (this._codeHash !== null) {
            partialData.push([oneEncoded, this._codeHash]);
        }
        else {
            partialData.push([zeroEncoded]);
        }
        if (this._codeSize !== null) {
            partialData.push([oneEncoded, (0, bytes_js_1.intToUnpaddedBytes)(this._codeSize)]);
        }
        else {
            partialData.push([zeroEncoded]);
        }
        if (this._version !== null) {
            partialData.push([oneEncoded, (0, bytes_js_1.intToUnpaddedBytes)(this._version)]);
        }
        else {
            partialData.push([zeroEncoded]);
        }
        return rlp_1.RLP.encode(partialData);
    }
    /**
     * Returns a `Boolean` determining if the account is a contract.
     */
    isContract() {
        if (this._codeHash === null && this._codeSize === null) {
            throw Error(`Insufficient data as codeHash=null and codeSize=null`);
        }
        return ((this._codeHash !== null && !(0, bytes_js_1.equalsBytes)(this._codeHash, constants_js_1.KECCAK256_NULL)) ||
            (this._codeSize !== null && this._codeSize !== 0));
    }
    /**
     * Returns a `Boolean` determining if the account is empty complying to the definition of
     * account emptiness in [EIP-161](https://eips.ethereum.org/EIPS/eip-161):
     * "An account is considered empty when it has no code and zero nonce and zero balance."
     */
    isEmpty() {
        // helpful for determination in partial accounts
        if ((this._balance !== null && this.balance !== constants_js_1.BIGINT_0) ||
            (this._nonce === null && this.nonce !== constants_js_1.BIGINT_0) ||
            (this._codeHash !== null && !(0, bytes_js_1.equalsBytes)(this.codeHash, constants_js_1.KECCAK256_NULL))) {
            return false;
        }
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
    const hash = (0, bytes_js_1.bytesToHex)((0, keccak_js_1.keccak256)(bytes)).slice(2);
    let ret = '';
    for (let i = 0; i < address.length; i++) {
        if (parseInt(hash[i], 16) >= 8) {
            ret += address[i].toUpperCase();
        }
        else {
            ret += address[i];
        }
    }
    return `0x${ret}`;
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
        return (0, keccak_js_1.keccak256)(rlp_1.RLP.encode([from, Uint8Array.from([])])).subarray(-20);
    }
    // Only take the lower 160bits of the hash
    return (0, keccak_js_1.keccak256)(rlp_1.RLP.encode([from, nonce])).subarray(-20);
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
    const address = (0, keccak_js_1.keccak256)((0, bytes_js_1.concatBytes)((0, bytes_js_1.hexToBytes)('0xff'), from, salt, (0, keccak_js_1.keccak256)(initCode)));
    return address.subarray(-20);
};
exports.generateAddress2 = generateAddress2;
/**
 * Checks if the private key satisfies the rules of the curve secp256k1.
 */
const isValidPrivate = function (privateKey) {
    return secp256k1_js_1.secp256k1.utils.isValidPrivateKey(privateKey);
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
        // Automatically checks whether point is on curve
        try {
            secp256k1_js_1.secp256k1.ProjectivePoint.fromHex((0, bytes_js_1.concatBytes)(Uint8Array.from([4]), publicKey));
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
        secp256k1_js_1.secp256k1.ProjectivePoint.fromHex(publicKey);
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
    (0, helpers_js_1.assertIsBytes)(pubKey);
    if (sanitize && pubKey.length !== 64) {
        pubKey = secp256k1_js_1.secp256k1.ProjectivePoint.fromHex(pubKey).toRawBytes(false).slice(1);
    }
    if (pubKey.length !== 64) {
        throw new Error('Expected pubKey to be of length 64');
    }
    // Only take the lower 160bits of the hash
    return (0, keccak_js_1.keccak256)(pubKey).subarray(-20);
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
    return secp256k1_js_1.secp256k1.ProjectivePoint.fromPrivateKey(privateKey).toRawBytes(false).slice(1);
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
        publicKey = secp256k1_js_1.secp256k1.ProjectivePoint.fromHex(publicKey).toRawBytes(false).slice(1);
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
    return rlp_1.RLP.encode(accountBody);
}
exports.accountBodyToRLP = accountBodyToRLP;
//# sourceMappingURL=account.js.map