"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.replaceAll = exports.pluralize = void 0;
/**
 * Returns the plural form of a word.
 *
 * @param n The number of things to represent. This dictates whether to return
 * the singular or plural form of the word.
 * @param singular The singular form of the word.
 * @param plural An optional plural form of the word. If non is given, the
 * plural form is constructed by appending an "s" to the singular form.
 */
function pluralize(n, singular, plural) {
    if (n === 1) {
        return singular;
    }
    if (plural !== undefined) {
        return plural;
    }
    return `${singular}s`;
}
exports.pluralize = pluralize;
/**
 * Replaces all the instances of [[toReplace]] by [[replacement]] in [[str]].
 */
function replaceAll(str, toReplace, replacement) {
    return str.split(toReplace).join(replacement);
}
exports.replaceAll = replaceAll;
//# sourceMappingURL=strings.js.map