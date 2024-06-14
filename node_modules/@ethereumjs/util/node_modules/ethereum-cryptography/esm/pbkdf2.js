import { pbkdf2 as _pbkdf2, pbkdf2Async as _pbkdf2Async } from "@noble/hashes/pbkdf2";
import { sha256 } from "@noble/hashes/sha256";
import { sha512 } from "@noble/hashes/sha512";
import { assertBytes } from "./utils.js";
export async function pbkdf2(password, salt, iterations, keylen, digest) {
    if (!["sha256", "sha512"].includes(digest)) {
        throw new Error("Only sha256 and sha512 are supported");
    }
    assertBytes(password);
    assertBytes(salt);
    return _pbkdf2Async(digest === "sha256" ? sha256 : sha512, password, salt, {
        c: iterations,
        dkLen: keylen
    });
}
export function pbkdf2Sync(password, salt, iterations, keylen, digest) {
    if (!["sha256", "sha512"].includes(digest)) {
        throw new Error("Only sha256 and sha512 are supported");
    }
    assertBytes(password);
    assertBytes(salt);
    return _pbkdf2(digest === "sha256" ? sha256 : sha512, password, salt, {
        c: iterations,
        dkLen: keylen
    });
}
