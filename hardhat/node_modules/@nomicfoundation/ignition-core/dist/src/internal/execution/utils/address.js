"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.equalAddresses = exports.toChecksumFormat = exports.isAddress = void 0;
const assertions_1 = require("../../utils/assertions");
/**
 * Is the string a valid ethereum address?
 */
function isAddress(address) {
    const { isAddress: ethersIsAddress } = require("ethers");
    return ethersIsAddress(address);
}
exports.isAddress = isAddress;
/**
 * Returns a normalized and checksumed address for the given address.
 *
 * @param address - the address to reformat
 * @returns checksumed address
 */
function toChecksumFormat(address) {
    (0, assertions_1.assertIgnitionInvariant)(isAddress(address), `Expected ${address} to be an address`);
    const { getAddress } = require("ethers");
    return getAddress(address);
}
exports.toChecksumFormat = toChecksumFormat;
/**
 * Determine if two addresses are equal ignoring case (which is a consideration
 * because of checksumming).
 */
function equalAddresses(leftAddress, rightAddress) {
    return toChecksumFormat(leftAddress) === toChecksumFormat(rightAddress);
}
exports.equalAddresses = equalAddresses;
//# sourceMappingURL=address.js.map