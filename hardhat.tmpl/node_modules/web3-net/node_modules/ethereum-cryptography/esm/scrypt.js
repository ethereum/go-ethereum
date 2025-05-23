import { scrypt as _sync, scryptAsync as _async } from "@noble/hashes/scrypt";
import { assertBytes } from "./utils.js";
export async function scrypt(password, salt, n, p, r, dkLen, onProgress) {
    assertBytes(password);
    assertBytes(salt);
    return _async(password, salt, { N: n, r, p, dkLen, onProgress });
}
export function scryptSync(password, salt, n, p, r, dkLen, onProgress) {
    assertBytes(password);
    assertBytes(salt);
    return _sync(password, salt, { N: n, r, p, dkLen, onProgress });
}
