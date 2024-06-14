/**
 * RLP Encoding based on https://ethereum.org/en/developers/docs/data-structures-and-encoding/rlp/
 * This function takes in data, converts it to Uint8Array if not,
 * and adds a length for recursion.
 * @param input Will be converted to Uint8Array
 * @returns Uint8Array of encoded data
 **/
export function encode(input) {
    if (Array.isArray(input)) {
        const output = [];
        let outputLength = 0;
        for (let i = 0; i < input.length; i++) {
            const encoded = encode(input[i]);
            output.push(encoded);
            outputLength += encoded.length;
        }
        return concatBytes(encodeLength(outputLength, 192), ...output);
    }
    const inputBuf = toBytes(input);
    if (inputBuf.length === 1 && inputBuf[0] < 128) {
        return inputBuf;
    }
    return concatBytes(encodeLength(inputBuf.length, 128), inputBuf);
}
/**
 * Slices a Uint8Array, throws if the slice goes out-of-bounds of the Uint8Array.
 * E.g. `safeSlice(hexToBytes('aa'), 1, 2)` will throw.
 * @param input
 * @param start
 * @param end
 */
function safeSlice(input, start, end) {
    if (end > input.length) {
        throw new Error('invalid RLP (safeSlice): end slice of Uint8Array out-of-bounds');
    }
    return input.slice(start, end);
}
/**
 * Parse integers. Check if there is no leading zeros
 * @param v The value to parse
 */
function decodeLength(v) {
    if (v[0] === 0) {
        throw new Error('invalid RLP: extra zeros');
    }
    return parseHexByte(bytesToHex(v));
}
function encodeLength(len, offset) {
    if (len < 56) {
        return Uint8Array.from([len + offset]);
    }
    const hexLength = numberToHex(len);
    const lLength = hexLength.length / 2;
    const firstByte = numberToHex(offset + 55 + lLength);
    return Uint8Array.from(hexToBytes(firstByte + hexLength));
}
export function decode(input, stream = false) {
    if (typeof input === 'undefined' || input === null || input.length === 0) {
        return Uint8Array.from([]);
    }
    const inputBytes = toBytes(input);
    const decoded = _decode(inputBytes);
    if (stream) {
        return decoded;
    }
    if (decoded.remainder.length !== 0) {
        throw new Error('invalid RLP: remainder must be zero');
    }
    return decoded.data;
}
/** Decode an input with RLP */
function _decode(input) {
    let length, llength, data, innerRemainder, d;
    const decoded = [];
    const firstByte = input[0];
    if (firstByte <= 0x7f) {
        // a single byte whose value is in the [0x00, 0x7f] range, that byte is its own RLP encoding.
        return {
            data: input.slice(0, 1),
            remainder: input.slice(1),
        };
    }
    else if (firstByte <= 0xb7) {
        // string is 0-55 bytes long. A single byte with value 0x80 plus the length of the string followed by the string
        // The range of the first byte is [0x80, 0xb7]
        length = firstByte - 0x7f;
        // set 0x80 null to 0
        if (firstByte === 0x80) {
            data = Uint8Array.from([]);
        }
        else {
            data = safeSlice(input, 1, length);
        }
        if (length === 2 && data[0] < 0x80) {
            throw new Error('invalid RLP encoding: invalid prefix, single byte < 0x80 are not prefixed');
        }
        return {
            data,
            remainder: input.slice(length),
        };
    }
    else if (firstByte <= 0xbf) {
        // string is greater than 55 bytes long. A single byte with the value (0xb7 plus the length of the length),
        // followed by the length, followed by the string
        llength = firstByte - 0xb6;
        if (input.length - 1 < llength) {
            throw new Error('invalid RLP: not enough bytes for string length');
        }
        length = decodeLength(safeSlice(input, 1, llength));
        if (length <= 55) {
            throw new Error('invalid RLP: expected string length to be greater than 55');
        }
        data = safeSlice(input, llength, length + llength);
        return {
            data,
            remainder: input.slice(length + llength),
        };
    }
    else if (firstByte <= 0xf7) {
        // a list between 0-55 bytes long
        length = firstByte - 0xbf;
        innerRemainder = safeSlice(input, 1, length);
        while (innerRemainder.length) {
            d = _decode(innerRemainder);
            decoded.push(d.data);
            innerRemainder = d.remainder;
        }
        return {
            data: decoded,
            remainder: input.slice(length),
        };
    }
    else {
        // a list over 55 bytes long
        llength = firstByte - 0xf6;
        length = decodeLength(safeSlice(input, 1, llength));
        if (length < 56) {
            throw new Error('invalid RLP: encoded list too short');
        }
        const totalLength = llength + length;
        if (totalLength > input.length) {
            throw new Error('invalid RLP: total length is larger than the data');
        }
        innerRemainder = safeSlice(input, llength, totalLength);
        while (innerRemainder.length) {
            d = _decode(innerRemainder);
            decoded.push(d.data);
            innerRemainder = d.remainder;
        }
        return {
            data: decoded,
            remainder: input.slice(totalLength),
        };
    }
}
const cachedHexes = Array.from({ length: 256 }, (_v, i) => i.toString(16).padStart(2, '0'));
function bytesToHex(uint8a) {
    // Pre-caching chars with `cachedHexes` speeds this up 6x
    let hex = '';
    for (let i = 0; i < uint8a.length; i++) {
        hex += cachedHexes[uint8a[i]];
    }
    return hex;
}
function parseHexByte(hexByte) {
    const byte = Number.parseInt(hexByte, 16);
    if (Number.isNaN(byte))
        throw new Error('Invalid byte sequence');
    return byte;
}
// Caching slows it down 2-3x
function hexToBytes(hex) {
    if (typeof hex !== 'string') {
        throw new TypeError('hexToBytes: expected string, got ' + typeof hex);
    }
    if (hex.length % 2)
        throw new Error('hexToBytes: received invalid unpadded hex');
    const array = new Uint8Array(hex.length / 2);
    for (let i = 0; i < array.length; i++) {
        const j = i * 2;
        array[i] = parseHexByte(hex.slice(j, j + 2));
    }
    return array;
}
/** Concatenates two Uint8Arrays into one. */
function concatBytes(...arrays) {
    if (arrays.length === 1)
        return arrays[0];
    const length = arrays.reduce((a, arr) => a + arr.length, 0);
    const result = new Uint8Array(length);
    for (let i = 0, pad = 0; i < arrays.length; i++) {
        const arr = arrays[i];
        result.set(arr, pad);
        pad += arr.length;
    }
    return result;
}
function utf8ToBytes(utf) {
    return new TextEncoder().encode(utf);
}
/** Transform an integer into its hexadecimal value */
function numberToHex(integer) {
    if (integer < 0) {
        throw new Error('Invalid integer as argument, must be unsigned!');
    }
    const hex = integer.toString(16);
    return hex.length % 2 ? `0${hex}` : hex;
}
/** Pad a string to be even */
function padToEven(a) {
    return a.length % 2 ? `0${a}` : a;
}
/** Check if a string is prefixed by 0x */
function isHexPrefixed(str) {
    return str.length >= 2 && str[0] === '0' && str[1] === 'x';
}
/** Removes 0x from a given String */
function stripHexPrefix(str) {
    if (typeof str !== 'string') {
        return str;
    }
    return isHexPrefixed(str) ? str.slice(2) : str;
}
/** Transform anything into a Uint8Array */
function toBytes(v) {
    if (v instanceof Uint8Array) {
        return v;
    }
    if (typeof v === 'string') {
        if (isHexPrefixed(v)) {
            return hexToBytes(padToEven(stripHexPrefix(v)));
        }
        return utf8ToBytes(v);
    }
    if (typeof v === 'number' || typeof v === 'bigint') {
        if (!v) {
            return Uint8Array.from([]);
        }
        return hexToBytes(numberToHex(v));
    }
    if (v === null || v === undefined) {
        return Uint8Array.from([]);
    }
    throw new Error('toBytes: received unsupported type ' + typeof v);
}
export const utils = {
    bytesToHex,
    concatBytes,
    hexToBytes,
    utf8ToBytes,
};
export const RLP = { encode, decode };
//# sourceMappingURL=index.js.map