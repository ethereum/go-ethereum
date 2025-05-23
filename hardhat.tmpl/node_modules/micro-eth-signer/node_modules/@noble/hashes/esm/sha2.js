/**
 * SHA2 hash function. A.k.a. sha256, sha512, sha512_256, etc.
 * @module
 */
// Usually you either use sha256, or sha512. We re-export them as sha2 for naming consistency.
export { sha224, sha256 } from "./sha256.js";
export { sha384, sha512, sha512_224, sha512_256 } from "./sha512.js";
//# sourceMappingURL=sha2.js.map