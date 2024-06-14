"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.normalizeDirName = void 0;
const lodash_1 = require("lodash");
/**
 * Converts valid directory name to valid variable name. Example: 0directory-name becomes _0DirectoryName
 */
function normalizeDirName(rawName) {
    const transformations = [
        (s) => (0, lodash_1.camelCase)(s),
        (s) => s.replace(/^\d/g, (match) => '_' + match), // prepend '_' if contains a leading number
    ];
    return transformations.reduce((s, t) => t(s), rawName);
}
exports.normalizeDirName = normalizeDirName;
//# sourceMappingURL=normalizeDirName.js.map