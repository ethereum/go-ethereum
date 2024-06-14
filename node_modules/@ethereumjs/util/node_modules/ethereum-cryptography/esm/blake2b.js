import { blake2b as _blake2b } from "@noble/hashes/blake2b";
import { assertBytes } from "./utils.js";
export const blake2b = (msg, outputLength = 64) => {
    assertBytes(msg);
    if (outputLength <= 0 || outputLength > 64) {
        throw Error("Invalid outputLength");
    }
    return _blake2b(msg, { dkLen: outputLength });
};
