"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.findDistance = exports.parseName = exports.parseFullyQualifiedName = exports.isFullyQualifiedName = exports.getFullyQualifiedName = void 0;
const errors_1 = require("../internal/core/errors");
const errors_list_1 = require("../internal/core/errors-list");
/**
 * Returns a fully qualified name from a sourceName and contractName.
 */
function getFullyQualifiedName(sourceName, contractName) {
    return `${sourceName}:${contractName}`;
}
exports.getFullyQualifiedName = getFullyQualifiedName;
/**
 * Returns true if a name is fully qualified, and not just a bare contract name.
 */
function isFullyQualifiedName(name) {
    return name.includes(":");
}
exports.isFullyQualifiedName = isFullyQualifiedName;
/**
 * Parses a fully qualified name.
 *
 * @param fullyQualifiedName It MUST be a fully qualified name.
 * @throws {HardhatError} If the name is not fully qualified.
 */
function parseFullyQualifiedName(fullyQualifiedName) {
    const { sourceName, contractName } = parseName(fullyQualifiedName);
    if (sourceName === undefined) {
        throw new errors_1.HardhatError(errors_list_1.ERRORS.CONTRACT_NAMES.INVALID_FULLY_QUALIFIED_NAME, {
            name: fullyQualifiedName,
        });
    }
    return { sourceName, contractName };
}
exports.parseFullyQualifiedName = parseFullyQualifiedName;
/**
 * Parses a name, which can be a bare contract name, or a fully qualified name.
 */
function parseName(name) {
    const parts = name.split(":");
    if (parts.length === 1) {
        return { contractName: parts[0] };
    }
    const contractName = parts[parts.length - 1];
    const sourceName = parts.slice(0, parts.length - 1).join(":");
    return { sourceName, contractName };
}
exports.parseName = parseName;
/**
 * Returns the edit-distance between two given strings using Levenshtein distance.
 *
 * @param a First string being compared
 * @param b Second string being compared
 * @returns distance between the two strings (lower number == more similar)
 * @see https://github.com/gustf/js-levenshtein
 * @license MIT - https://github.com/gustf/js-levenshtein/blob/master/LICENSE
 */
function findDistance(a, b) {
    function _min(_d0, _d1, _d2, _bx, _ay) {
        return _d0 < _d1 || _d2 < _d1
            ? _d0 > _d2
                ? _d2 + 1
                : _d0 + 1
            : _bx === _ay
                ? _d1
                : _d1 + 1;
    }
    if (a === b) {
        return 0;
    }
    if (a.length > b.length) {
        [a, b] = [b, a];
    }
    let la = a.length;
    let lb = b.length;
    while (la > 0 && a.charCodeAt(la - 1) === b.charCodeAt(lb - 1)) {
        la--;
        lb--;
    }
    let offset = 0;
    while (offset < la && a.charCodeAt(offset) === b.charCodeAt(offset)) {
        offset++;
    }
    la -= offset;
    lb -= offset;
    if (la === 0 || lb < 3) {
        return lb;
    }
    let x = 0;
    let y;
    let d0;
    let d1;
    let d2;
    let d3;
    let dd = 0; // typescript gets angry if we don't assign here
    let dy;
    let ay;
    let bx0;
    let bx1;
    let bx2;
    let bx3;
    const vector = [];
    for (y = 0; y < la; y++) {
        vector.push(y + 1);
        vector.push(a.charCodeAt(offset + y));
    }
    const len = vector.length - 1;
    for (; x < lb - 3;) {
        bx0 = b.charCodeAt(offset + (d0 = x));
        bx1 = b.charCodeAt(offset + (d1 = x + 1));
        bx2 = b.charCodeAt(offset + (d2 = x + 2));
        bx3 = b.charCodeAt(offset + (d3 = x + 3));
        dd = x += 4;
        for (y = 0; y < len; y += 2) {
            dy = vector[y];
            ay = vector[y + 1];
            d0 = _min(dy, d0, d1, bx0, ay);
            d1 = _min(d0, d1, d2, bx1, ay);
            d2 = _min(d1, d2, d3, bx2, ay);
            dd = _min(d2, d3, dd, bx3, ay);
            vector[y] = dd;
            d3 = d2;
            d2 = d1;
            d1 = d0;
            d0 = dy;
        }
    }
    for (; x < lb;) {
        bx0 = b.charCodeAt(offset + (d0 = x));
        dd = ++x;
        for (y = 0; y < len; y += 2) {
            dy = vector[y];
            vector[y] = dd = _min(dy, d0, dd, bx0, vector[y + 1]);
            d0 = dy;
        }
    }
    return dd;
}
exports.findDistance = findDistance;
//# sourceMappingURL=contract-names.js.map