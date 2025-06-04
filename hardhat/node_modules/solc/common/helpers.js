"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.isObject = exports.isNil = void 0;
/**
 * Returns true if and only if the value is null or undefined.
 *
 * @param value
 */
function isNil(value) {
    // Uses == over === which compares both null and undefined.
    return value == null;
}
exports.isNil = isNil;
/**
 * Returns true if and only if the value is an object and not an array.
 *
 * @param value
 */
function isObject(value) {
    // typeof [] will result in an 'object' so this additionally uses Array.isArray
    // to confirm it's just an object.
    return typeof value === 'object' && !Array.isArray(value);
}
exports.isObject = isObject;
