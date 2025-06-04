"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.unsafeObjectEntries = exports.unsafeObjectKeys = void 0;
/**
 * This function is a typed version of `Object.keys`. Note that it's type
 * unsafe. You have to be sure that `o` has exactly the same keys as `T`.
 */
exports.unsafeObjectKeys = Object.keys;
/**
 * This function is a typed version of `Object.entries`. Note that it's type
 * unsafe. You have to be sure that `o` has exactly the same keys as `T`.
 */
function unsafeObjectEntries(o) {
    return Object.entries(o);
}
exports.unsafeObjectEntries = unsafeObjectEntries;
//# sourceMappingURL=unsafe.js.map