"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Address = void 0;
var assert_1 = __importDefault(require("assert"));
var externals_1 = require("./externals");
var bytes_1 = require("./bytes");
var account_1 = require("./account");
var Address = /** @class */ (function () {
    function Address(buf) {
        (0, assert_1.default)(buf.length === 20, 'Invalid address length');
        this.buf = buf;
    }
    /**
     * Returns the zero address.
     */
    Address.zero = function () {
        return new Address((0, bytes_1.zeros)(20));
    };
    /**
     * Returns an Address object from a hex-encoded string.
     * @param str - Hex-encoded address
     */
    Address.fromString = function (str) {
        (0, assert_1.default)((0, account_1.isValidAddress)(str), 'Invalid address');
        return new Address((0, bytes_1.toBuffer)(str));
    };
    /**
     * Returns an address for a given public key.
     * @param pubKey The two points of an uncompressed key
     */
    Address.fromPublicKey = function (pubKey) {
        (0, assert_1.default)(Buffer.isBuffer(pubKey), 'Public key should be Buffer');
        var buf = (0, account_1.pubToAddress)(pubKey);
        return new Address(buf);
    };
    /**
     * Returns an address for a given private key.
     * @param privateKey A private key must be 256 bits wide
     */
    Address.fromPrivateKey = function (privateKey) {
        (0, assert_1.default)(Buffer.isBuffer(privateKey), 'Private key should be Buffer');
        var buf = (0, account_1.privateToAddress)(privateKey);
        return new Address(buf);
    };
    /**
     * Generates an address for a newly created contract.
     * @param from The address which is creating this new address
     * @param nonce The nonce of the from account
     */
    Address.generate = function (from, nonce) {
        (0, assert_1.default)(externals_1.BN.isBN(nonce));
        return new Address((0, account_1.generateAddress)(from.buf, nonce.toArrayLike(Buffer)));
    };
    /**
     * Generates an address for a contract created using CREATE2.
     * @param from The address which is creating this new address
     * @param salt A salt
     * @param initCode The init code of the contract being created
     */
    Address.generate2 = function (from, salt, initCode) {
        (0, assert_1.default)(Buffer.isBuffer(salt));
        (0, assert_1.default)(Buffer.isBuffer(initCode));
        return new Address((0, account_1.generateAddress2)(from.buf, salt, initCode));
    };
    /**
     * Is address equal to another.
     */
    Address.prototype.equals = function (address) {
        return this.buf.equals(address.buf);
    };
    /**
     * Is address zero.
     */
    Address.prototype.isZero = function () {
        return this.equals(Address.zero());
    };
    /**
     * True if address is in the address range defined
     * by EIP-1352
     */
    Address.prototype.isPrecompileOrSystemAddress = function () {
        var addressBN = new externals_1.BN(this.buf);
        var rangeMin = new externals_1.BN(0);
        var rangeMax = new externals_1.BN('ffff', 'hex');
        return addressBN.gte(rangeMin) && addressBN.lte(rangeMax);
    };
    /**
     * Returns hex encoding of address.
     */
    Address.prototype.toString = function () {
        return '0x' + this.buf.toString('hex');
    };
    /**
     * Returns Buffer representation of address.
     */
    Address.prototype.toBuffer = function () {
        return Buffer.from(this.buf);
    };
    return Address;
}());
exports.Address = Address;
//# sourceMappingURL=address.js.map