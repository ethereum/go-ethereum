"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Address = void 0;
const assert_1 = __importDefault(require("assert"));
const externals_1 = require("./externals");
const bytes_1 = require("./bytes");
const account_1 = require("./account");
class Address {
    constructor(buf) {
        (0, assert_1.default)(buf.length === 20, 'Invalid address length');
        this.buf = buf;
    }
    /**
     * Returns the zero address.
     */
    static zero() {
        return new Address((0, bytes_1.zeros)(20));
    }
    /**
     * Returns an Address object from a hex-encoded string.
     * @param str - Hex-encoded address
     */
    static fromString(str) {
        (0, assert_1.default)((0, account_1.isValidAddress)(str), 'Invalid address');
        return new Address((0, bytes_1.toBuffer)(str));
    }
    /**
     * Returns an address for a given public key.
     * @param pubKey The two points of an uncompressed key
     */
    static fromPublicKey(pubKey) {
        (0, assert_1.default)(Buffer.isBuffer(pubKey), 'Public key should be Buffer');
        const buf = (0, account_1.pubToAddress)(pubKey);
        return new Address(buf);
    }
    /**
     * Returns an address for a given private key.
     * @param privateKey A private key must be 256 bits wide
     */
    static fromPrivateKey(privateKey) {
        (0, assert_1.default)(Buffer.isBuffer(privateKey), 'Private key should be Buffer');
        const buf = (0, account_1.privateToAddress)(privateKey);
        return new Address(buf);
    }
    /**
     * Generates an address for a newly created contract.
     * @param from The address which is creating this new address
     * @param nonce The nonce of the from account
     */
    static generate(from, nonce) {
        (0, assert_1.default)(externals_1.BN.isBN(nonce));
        return new Address((0, account_1.generateAddress)(from.buf, nonce.toArrayLike(Buffer)));
    }
    /**
     * Generates an address for a contract created using CREATE2.
     * @param from The address which is creating this new address
     * @param salt A salt
     * @param initCode The init code of the contract being created
     */
    static generate2(from, salt, initCode) {
        (0, assert_1.default)(Buffer.isBuffer(salt));
        (0, assert_1.default)(Buffer.isBuffer(initCode));
        return new Address((0, account_1.generateAddress2)(from.buf, salt, initCode));
    }
    /**
     * Is address equal to another.
     */
    equals(address) {
        return this.buf.equals(address.buf);
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
        const addressBN = new externals_1.BN(this.buf);
        const rangeMin = new externals_1.BN(0);
        const rangeMax = new externals_1.BN('ffff', 'hex');
        return addressBN.gte(rangeMin) && addressBN.lte(rangeMax);
    }
    /**
     * Returns hex encoding of address.
     */
    toString() {
        return '0x' + this.buf.toString('hex');
    }
    /**
     * Returns Buffer representation of address.
     */
    toBuffer() {
        return Buffer.from(this.buf);
    }
}
exports.Address = Address;
//# sourceMappingURL=address.js.map