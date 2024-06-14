"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.Address = void 0;
const account_js_1 = require("./account.js");
const bytes_js_1 = require("./bytes.js");
const constants_js_1 = require("./constants.js");
/**
 * Handling and generating Ethereum addresses
 */
class Address {
    constructor(bytes) {
        if (bytes.length !== 20) {
            throw new Error('Invalid address length');
        }
        this.bytes = bytes;
    }
    /**
     * Returns the zero address.
     */
    static zero() {
        return new Address((0, bytes_js_1.zeros)(20));
    }
    /**
     * Returns an Address object from a hex-encoded string.
     * @param str - Hex-encoded address
     */
    static fromString(str) {
        if (!(0, account_js_1.isValidAddress)(str)) {
            throw new Error('Invalid address');
        }
        return new Address((0, bytes_js_1.toBytes)(str));
    }
    /**
     * Returns an address for a given public key.
     * @param pubKey The two points of an uncompressed key
     */
    static fromPublicKey(pubKey) {
        if (!(pubKey instanceof Uint8Array)) {
            throw new Error('Public key should be Uint8Array');
        }
        const bytes = (0, account_js_1.pubToAddress)(pubKey);
        return new Address(bytes);
    }
    /**
     * Returns an address for a given private key.
     * @param privateKey A private key must be 256 bits wide
     */
    static fromPrivateKey(privateKey) {
        if (!(privateKey instanceof Uint8Array)) {
            throw new Error('Private key should be Uint8Array');
        }
        const bytes = (0, account_js_1.privateToAddress)(privateKey);
        return new Address(bytes);
    }
    /**
     * Generates an address for a newly created contract.
     * @param from The address which is creating this new address
     * @param nonce The nonce of the from account
     */
    static generate(from, nonce) {
        if (typeof nonce !== 'bigint') {
            throw new Error('Expected nonce to be a bigint');
        }
        return new Address((0, account_js_1.generateAddress)(from.bytes, (0, bytes_js_1.bigIntToBytes)(nonce)));
    }
    /**
     * Generates an address for a contract created using CREATE2.
     * @param from The address which is creating this new address
     * @param salt A salt
     * @param initCode The init code of the contract being created
     */
    static generate2(from, salt, initCode) {
        if (!(salt instanceof Uint8Array)) {
            throw new Error('Expected salt to be a Uint8Array');
        }
        if (!(initCode instanceof Uint8Array)) {
            throw new Error('Expected initCode to be a Uint8Array');
        }
        return new Address((0, account_js_1.generateAddress2)(from.bytes, salt, initCode));
    }
    /**
     * Is address equal to another.
     */
    equals(address) {
        return (0, bytes_js_1.equalsBytes)(this.bytes, address.bytes);
    }
    /**
     * Is address zero.
     */
    isZero() {
        return this.equals(Address.zero());
    }
    /**
     * True if address is in the address range defined
     * by EIP-1352
     */
    isPrecompileOrSystemAddress() {
        const address = (0, bytes_js_1.bytesToBigInt)(this.bytes);
        const rangeMin = constants_js_1.BIGINT_0;
        const rangeMax = BigInt('0xffff');
        return address >= rangeMin && address <= rangeMax;
    }
    /**
     * Returns hex encoding of address.
     */
    toString() {
        return (0, bytes_js_1.bytesToHex)(this.bytes);
    }
    /**
     * Returns a new Uint8Array representation of address.
     */
    toBytes() {
        return new Uint8Array(this.bytes);
    }
}
exports.Address = Address;
//# sourceMappingURL=address.js.map