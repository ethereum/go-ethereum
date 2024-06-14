"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.compactBytesToNibbles = exports.bytesToNibbles = exports.nibblesToCompactBytes = exports.nibblesToBytes = exports.hasTerminator = void 0;
// Reference: https://ethereum.org/en/developers/docs/data-structures-and-encoding/patricia-merkle-trie/
/**
 *
 * @param s byte sequence
 * @returns boolean indicating if input hex nibble sequence has terminator indicating leaf-node
 *          terminator is represented with 16 because a nibble ranges from 0 - 15(f)
 */
const hasTerminator = (nibbles) => {
    return nibbles.length > 0 && nibbles[nibbles.length - 1] === 16;
};
exports.hasTerminator = hasTerminator;
const nibblesToBytes = (nibbles, bytes) => {
    for (let bi = 0, ni = 0; ni < nibbles.length; bi += 1, ni += 2) {
        bytes[bi] = (nibbles[ni] << 4) | nibbles[ni + 1];
    }
};
exports.nibblesToBytes = nibblesToBytes;
const nibblesToCompactBytes = (nibbles) => {
    let terminator = 0;
    if ((0, exports.hasTerminator)(nibbles)) {
        terminator = 1;
        // Remove the terminator from the sequence
        nibbles = nibbles.subarray(0, nibbles.length - 1);
    }
    const buf = new Uint8Array(nibbles.length / 2 + 1);
    // Shift the terminator info into the first nibble of buf[0]
    buf[0] = terminator << 5;
    // If odd length, then add that flag into the first nibble and put the odd nibble to
    // second part of buf[0] which otherwise will be left padded with a 0
    if ((nibbles.length & 1) === 1) {
        buf[0] |= 1 << 4;
        buf[0] |= nibbles[0];
        nibbles = nibbles.subarray(1);
    }
    // create bytes out of the rest even nibbles
    (0, exports.nibblesToBytes)(nibbles, buf.subarray(1));
    return buf;
};
exports.nibblesToCompactBytes = nibblesToCompactBytes;
const bytesToNibbles = (str) => {
    const l = str.length * 2 + 1;
    const nibbles = new Uint8Array(l);
    for (let i = 0; i < str.length; i++) {
        const b = str[i];
        nibbles[i * 2] = b / 16;
        nibbles[i * 2 + 1] = b % 16;
    }
    // This will get removed from calling function if the first nibble
    // indicates that terminator is not present
    nibbles[l - 1] = 16;
    return nibbles;
};
exports.bytesToNibbles = bytesToNibbles;
const compactBytesToNibbles = (compact) => {
    if (compact.length === 0) {
        return compact;
    }
    let base = (0, exports.bytesToNibbles)(compact);
    // delete terminator flag if terminator flag was not in first nibble
    if (base[0] < 2) {
        base = base.subarray(0, base.length - 1);
    }
    // chop the terminator nibble and the even padding (if there is one)
    // i.e.  chop 2 left nibbles when even else 1 when odd
    const chop = 2 - (base[0] & 1);
    return base.subarray(chop);
};
exports.compactBytesToNibbles = compactBytesToNibbles;
/**
 * A test helper to generates compact path for a subset of key bytes
 *
 * TODO: Commenting the code for now as this seems to be helper function
 * (from geth codebase )
 *
 */
//
//
// export const getPathTo = (tillBytes: number, key: Buffer) => {
//   const hexNibbles = bytesToNibbles(key).subarray(0, tillBytes)
//   // Remove the terminator if its there, although it would be there only if tillBytes >= key.length
//   // This seems to be a test helper to generate paths so correctness of this isn't necessary
//   hexNibbles[hexNibbles.length - 1] = 0
//   const compactBytes = nibblesToCompactBytes(hexNibbles)
//   return [Buffer.from(compactBytes)]
// }
//# sourceMappingURL=encoding.js.map