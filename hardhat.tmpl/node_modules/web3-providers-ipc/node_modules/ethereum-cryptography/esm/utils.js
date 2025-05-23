import assert from "@noble/hashes/_assert";
import { hexToBytes as _hexToBytes } from "@noble/hashes/utils";
const assertBool = assert.bool;
const assertBytes = assert.bytes;
export { assertBool, assertBytes };
export { bytesToHex, bytesToHex as toHex, concatBytes, createView, utf8ToBytes } from "@noble/hashes/utils";
// buf.toString('utf8') -> bytesToUtf8(buf)
export function bytesToUtf8(data) {
    if (!(data instanceof Uint8Array)) {
        throw new TypeError(`bytesToUtf8 expected Uint8Array, got ${typeof data}`);
    }
    return new TextDecoder().decode(data);
}
export function hexToBytes(data) {
    const sliced = data.startsWith("0x") ? data.substring(2) : data;
    return _hexToBytes(sliced);
}
// buf.equals(buf2) -> equalsBytes(buf, buf2)
export function equalsBytes(a, b) {
    if (a.length !== b.length) {
        return false;
    }
    for (let i = 0; i < a.length; i++) {
        if (a[i] !== b[i]) {
            return false;
        }
    }
    return true;
}
// Internal utils
export function wrapHash(hash) {
    return (msg) => {
        assert.bytes(msg);
        return hash(msg);
    };
}
// TODO(v3): switch away from node crypto, remove this unnecessary variable.
export const crypto = (() => {
    const webCrypto = typeof globalThis === "object" && "crypto" in globalThis ? globalThis.crypto : undefined;
    const nodeRequire = typeof module !== "undefined" &&
        typeof module.require === "function" &&
        module.require.bind(module);
    return {
        node: nodeRequire && !webCrypto ? nodeRequire("crypto") : undefined,
        web: webCrypto
    };
})();
