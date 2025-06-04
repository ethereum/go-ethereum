"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.randomAddressBuffer = exports.randomAddressString = exports.randomAddress = exports.randomHashBuffer = exports.randomHash = exports.RandomBufferGenerator = void 0;
class RandomBufferGenerator {
    constructor(_nextValue) {
        this._nextValue = _nextValue;
    }
    static create(seed) {
        const { keccak256 } = require("../../../util/keccak");
        const nextValue = keccak256(Buffer.from(seed));
        return new RandomBufferGenerator(nextValue);
    }
    next() {
        const { keccak256 } = require("../../../util/keccak");
        const valueToReturn = this._nextValue;
        this._nextValue = keccak256(this._nextValue);
        return valueToReturn;
    }
    seed() {
        return this._nextValue;
    }
    setNext(nextValue) {
        this._nextValue = Buffer.from(nextValue);
    }
    clone() {
        return new RandomBufferGenerator(this._nextValue);
    }
}
exports.RandomBufferGenerator = RandomBufferGenerator;
const randomHash = () => {
    const { bytesToHex: bufferToHex } = require("@ethereumjs/util");
    return bufferToHex((0, exports.randomHashBuffer)());
};
exports.randomHash = randomHash;
const generator = RandomBufferGenerator.create("seed");
const randomHashBuffer = () => {
    return generator.next();
};
exports.randomHashBuffer = randomHashBuffer;
const randomAddress = () => {
    const { Address } = require("@ethereumjs/util");
    return new Address((0, exports.randomAddressBuffer)());
};
exports.randomAddress = randomAddress;
const randomAddressString = () => {
    const { bytesToHex: bufferToHex } = require("@ethereumjs/util");
    return bufferToHex((0, exports.randomAddressBuffer)());
};
exports.randomAddressString = randomAddressString;
const addressGenerator = RandomBufferGenerator.create("seed");
const randomAddressBuffer = () => {
    return addressGenerator.next().slice(0, 20);
};
exports.randomAddressBuffer = randomAddressBuffer;
//# sourceMappingURL=random.js.map