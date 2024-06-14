"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.assertIsString = exports.assertIsArray = exports.assertIsBytes = exports.assertIsHexString = void 0;
const internal_js_1 = require("./internal.js");
/**
 * Throws if a string is not hex prefixed
 * @param {string} input string to check hex prefix of
 */
const assertIsHexString = function (input) {
    if (!(0, internal_js_1.isHexString)(input)) {
        const msg = `This method only supports 0x-prefixed hex strings but input was: ${input}`;
        throw new Error(msg);
    }
};
exports.assertIsHexString = assertIsHexString;
/**
 * Throws if input is not a buffer
 * @param {Buffer} input value to check
 */
const assertIsBytes = function (input) {
    if (!(input instanceof Uint8Array)) {
        const msg = `This method only supports Uint8Array but input was: ${input}`;
        throw new Error(msg);
    }
};
exports.assertIsBytes = assertIsBytes;
/**
 * Throws if input is not an array
 * @param {number[]} input value to check
 */
const assertIsArray = function (input) {
    if (!Array.isArray(input)) {
        const msg = `This method only supports number arrays but input was: ${input}`;
        throw new Error(msg);
    }
};
exports.assertIsArray = assertIsArray;
/**
 * Throws if input is not a string
 * @param {string} input value to check
 */
const assertIsString = function (input) {
    if (typeof input !== 'string') {
        const msg = `This method only supports strings but input was: ${input}`;
        throw new Error(msg);
    }
};
exports.assertIsString = assertIsString;
//# sourceMappingURL=helpers.js.map