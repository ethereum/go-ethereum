"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeName = void 0;
const lodash_1 = require("lodash");
/**
 * Converts valid file names to valid javascript symbols and does best effort to make them readable. Example: ds-token.test becomes DsTokenTest
 */
function normalizeName(rawName) {
    const transformations = [
        (s) => s.replace(/\s+/g, '-'),
        (s) => s.replace(/\./g, '-'),
        (s) => s.replace(/-[a-z]/g, (match) => match.substr(-1).toUpperCase()),
        (s) => s.replace(/-/g, ''),
        (s) => s.replace(/^\d+/, ''),
        (s) => (0, lodash_1.upperFirst)(s),
    ];
    const finalName = transformations.reduce((s, t) => t(s), rawName);
    if (finalName === '') {
        throw new Error(`Can't guess class name, please rename file: ${rawName}`);
    }
    return finalName;
}
exports.normalizeName = normalizeName;
//# sourceMappingURL=normalizeName.js.map