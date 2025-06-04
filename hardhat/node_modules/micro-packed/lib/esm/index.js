import { hex as baseHex, utf8 } from '@scure/base';
/**
 * Define complex binary structures using composable primitives.
 * Main ideas:
 * - Encode / decode can be chained, same as in `scure-base`
 * - A complex structure can be created from an array and struct of primitive types
 * - Strings / bytes are arrays with specific optimizations: we can just read bytes directly
 *   without creating plain array first and reading each byte separately.
 * - Types are inferred from definition
 * @module
 * @example
 * import * as P from 'micro-packed';
 * const s = P.struct({
 *   field1: P.U32BE, // 32-bit unsigned big-endian integer
 *   field2: P.string(P.U8), // String with U8 length prefix
 *   field3: P.bytes(32), // 32 bytes
 *   field4: P.array(P.U16BE, P.struct({ // Array of structs with U16BE length
 *     subField1: P.U64BE, // 64-bit unsigned big-endian integer
 *     subField2: P.string(10) // 10-byte string
 *   }))
 * });
 */
// TODO: remove dependency on scure-base & inline?
/*
Exports can be groupped like this:

- Primitive types: P.bytes, P.string, P.hex, P.constant, P.pointer
- Complex types: P.array, P.struct, P.tuple, P.map, P.tag, P.mappedTag
- Padding, prefix, magic: P.padLeft, P.padRight, P.prefix, P.magic, P.magicBytes
- Flags: P.flag, P.flagged, P.optional
- Wrappers: P.apply, P.wrap, P.lazy
- Bit fiddling: P.bits, P.bitset
- utils: P.validate, coders.decimal
- Debugger
*/
/** Shortcut to zero-length (empty) byte array */
export const EMPTY = /* @__PURE__ */ new Uint8Array();
/** Shortcut to one-element (element is 0) byte array */
export const NULL = /* @__PURE__ */ new Uint8Array([0]);
/** Checks if two Uint8Arrays are equal. Not constant-time. */
function equalBytes(a, b) {
    if (a.length !== b.length)
        return false;
    for (let i = 0; i < a.length; i++)
        if (a[i] !== b[i])
            return false;
    return true;
}
/** Checks if the given value is a Uint8Array. */
function isBytes(a) {
    return a instanceof Uint8Array || (ArrayBuffer.isView(a) && a.constructor.name === 'Uint8Array');
}
/**
 * Concatenates multiple Uint8Arrays.
 * Engines limit functions to 65K+ arguments.
 * @param arrays Array of Uint8Array elements
 * @returns Concatenated Uint8Array
 */
function concatBytes(...arrays) {
    let sum = 0;
    for (let i = 0; i < arrays.length; i++) {
        const a = arrays[i];
        if (!isBytes(a))
            throw new Error('Uint8Array expected');
        sum += a.length;
    }
    const res = new Uint8Array(sum);
    for (let i = 0, pad = 0; i < arrays.length; i++) {
        const a = arrays[i];
        res.set(a, pad);
        pad += a.length;
    }
    return res;
}
/**
 * Creates DataView from Uint8Array
 * @param arr - bytes
 * @returns DataView
 */
const createView = (arr) => new DataView(arr.buffer, arr.byteOffset, arr.byteLength);
/**
 * Checks if the provided value is a plain object, not created from any class or special constructor.
 * Array, Uint8Array and others are not plain objects.
 * @param obj - The value to be checked.
 */
function isPlainObject(obj) {
    return Object.prototype.toString.call(obj) === '[object Object]';
}
function isNum(num) {
    return Number.isSafeInteger(num);
}
export const utils = {
    equalBytes,
    isBytes,
    isCoder,
    checkBounds,
    concatBytes,
    createView,
    isPlainObject,
};
// NOTE: we can't have terminator separate function, since it won't know about boundaries
// E.g. array of U16LE ([1,2,3]) would be [1, 0, 2, 0, 3, 0]
// But terminator will find array at index '1', which happens to be inside of an element itself
/**
 * Can be:
 * - Dynamic (CoderType)
 * - Fixed (number)
 * - Terminated (usually zero): Uint8Array with terminator
 * - Field path to field with length (string)
 * - Infinity (null) - decodes until end of buffer
 * Used in:
 * - bytes (string, prefix is implementation of bytes)
 * - array
 */
const lengthCoder = (len) => {
    if (len !== null && typeof len !== 'string' && !isCoder(len) && !isBytes(len) && !isNum(len)) {
        throw new Error(`lengthCoder: expected null | number | Uint8Array | CoderType, got ${len} (${typeof len})`);
    }
    return {
        encodeStream(w, value) {
            if (len === null)
                return;
            if (isCoder(len))
                return len.encodeStream(w, value);
            let byteLen;
            if (typeof len === 'number')
                byteLen = len;
            else if (typeof len === 'string')
                byteLen = Path.resolve(w.stack, len);
            if (typeof byteLen === 'bigint')
                byteLen = Number(byteLen);
            if (byteLen === undefined || byteLen !== value)
                throw w.err(`Wrong length: ${byteLen} len=${len} exp=${value} (${typeof value})`);
        },
        decodeStream(r) {
            let byteLen;
            if (isCoder(len))
                byteLen = Number(len.decodeStream(r));
            else if (typeof len === 'number')
                byteLen = len;
            else if (typeof len === 'string')
                byteLen = Path.resolve(r.stack, len);
            if (typeof byteLen === 'bigint')
                byteLen = Number(byteLen);
            if (typeof byteLen !== 'number')
                throw r.err(`Wrong length: ${byteLen}`);
            return byteLen;
        },
    };
};
/**
 * Small bitset structure to store position of ranges that have been read.
 * Can be more efficient when internal trees are utilized at the cost of complexity.
 * Needs `O(N/8)` memory for parsing.
 * Purpose: if there are pointers in parsed structure,
 * they can cause read of two distinct ranges:
 * [0-32, 64-128], which means 'pos' is not enough to handle them
 */
const Bitset = {
    BITS: 32,
    FULL_MASK: -1 >>> 0, // 1<<32 will overflow
    len: (len) => Math.ceil(len / 32),
    create: (len) => new Uint32Array(Bitset.len(len)),
    clean: (bs) => bs.fill(0),
    debug: (bs) => Array.from(bs).map((i) => (i >>> 0).toString(2).padStart(32, '0')),
    checkLen: (bs, len) => {
        if (Bitset.len(len) === bs.length)
            return;
        throw new Error(`wrong length=${bs.length}. Expected: ${Bitset.len(len)}`);
    },
    chunkLen: (bsLen, pos, len) => {
        if (pos < 0)
            throw new Error(`wrong pos=${pos}`);
        if (pos + len > bsLen)
            throw new Error(`wrong range=${pos}/${len} of ${bsLen}`);
    },
    set: (bs, chunk, value, allowRewrite = true) => {
        if (!allowRewrite && (bs[chunk] & value) !== 0)
            return false;
        bs[chunk] |= value;
        return true;
    },
    pos: (pos, i) => ({
        chunk: Math.floor((pos + i) / 32),
        mask: 1 << (32 - ((pos + i) % 32) - 1),
    }),
    indices: (bs, len, invert = false) => {
        Bitset.checkLen(bs, len);
        const { FULL_MASK, BITS } = Bitset;
        const left = BITS - (len % BITS);
        const lastMask = left ? (FULL_MASK >>> left) << left : FULL_MASK;
        const res = [];
        for (let i = 0; i < bs.length; i++) {
            let c = bs[i];
            if (invert)
                c = ~c; // allows to gen unset elements
            // apply mask to last element, so we won't iterate non-existent items
            if (i === bs.length - 1)
                c &= lastMask;
            if (c === 0)
                continue; // fast-path
            for (let j = 0; j < BITS; j++) {
                const m = 1 << (BITS - j - 1);
                if (c & m)
                    res.push(i * BITS + j);
            }
        }
        return res;
    },
    range: (arr) => {
        const res = [];
        let cur;
        for (const i of arr) {
            if (cur === undefined || i !== cur.pos + cur.length)
                res.push((cur = { pos: i, length: 1 }));
            else
                cur.length += 1;
        }
        return res;
    },
    rangeDebug: (bs, len, invert = false) => `[${Bitset.range(Bitset.indices(bs, len, invert))
        .map((i) => `(${i.pos}/${i.length})`)
        .join(', ')}]`,
    setRange: (bs, bsLen, pos, len, allowRewrite = true) => {
        Bitset.chunkLen(bsLen, pos, len);
        const { FULL_MASK, BITS } = Bitset;
        // Try to set range with maximum efficiency:
        // - first chunk is always    '0000[1111]' (only right ones)
        // - middle chunks are set to '[1111 1111]' (all ones)
        // - last chunk is always     '[1111]0000' (only left ones)
        // - max operations:          (N/32) + 2 (first and last)
        const first = pos % BITS ? Math.floor(pos / BITS) : undefined;
        const lastPos = pos + len;
        const last = lastPos % BITS ? Math.floor(lastPos / BITS) : undefined;
        // special case, whole range inside single chunk
        if (first !== undefined && first === last)
            return Bitset.set(bs, first, (FULL_MASK >>> (BITS - len)) << (BITS - len - pos), allowRewrite);
        if (first !== undefined) {
            if (!Bitset.set(bs, first, FULL_MASK >>> pos % BITS, allowRewrite))
                return false; // first chunk
        }
        // middle chunks
        const start = first !== undefined ? first + 1 : pos / BITS;
        const end = last !== undefined ? last : lastPos / BITS;
        for (let i = start; i < end; i++)
            if (!Bitset.set(bs, i, FULL_MASK, allowRewrite))
                return false;
        if (last !== undefined && first !== last)
            if (!Bitset.set(bs, last, FULL_MASK << (BITS - (lastPos % BITS)), allowRewrite))
                return false; // last chunk
        return true;
    },
};
const Path = {
    /**
     * Internal method for handling stack of paths (debug, errors, dynamic fields via path)
     * This is looks ugly (callback), but allows us to force stack cleaning by construction (.pop always after function).
     * Also, this makes impossible:
     * - pushing field when stack is empty
     * - pushing field inside of field (real bug)
     * NOTE: we don't want to do '.pop' on error!
     */
    pushObj: (stack, obj, objFn) => {
        const last = { obj };
        stack.push(last);
        objFn((field, fieldFn) => {
            last.field = field;
            fieldFn();
            last.field = undefined;
        });
        stack.pop();
    },
    path: (stack) => {
        const res = [];
        for (const i of stack)
            if (i.field !== undefined)
                res.push(i.field);
        return res.join('/');
    },
    err: (name, stack, msg) => {
        const err = new Error(`${name}(${Path.path(stack)}): ${typeof msg === 'string' ? msg : msg.message}`);
        if (msg instanceof Error && msg.stack)
            err.stack = msg.stack;
        return err;
    },
    resolve: (stack, path) => {
        const parts = path.split('/');
        const objPath = stack.map((i) => i.obj);
        let i = 0;
        for (; i < parts.length; i++) {
            if (parts[i] === '..')
                objPath.pop();
            else
                break;
        }
        let cur = objPath.pop();
        for (; i < parts.length; i++) {
            if (!cur || cur[parts[i]] === undefined)
                return undefined;
            cur = cur[parts[i]];
        }
        return cur;
    },
};
/**
 * Internal structure. Reader class for reading from a byte array.
 * `stack` is internal: for debugger and logging
 * @class Reader
 */
class _Reader {
    constructor(data, opts = {}, stack = [], parent = undefined, parentOffset = 0) {
        this.pos = 0;
        this.bitBuf = 0;
        this.bitPos = 0;
        this.data = data;
        this.opts = opts;
        this.stack = stack;
        this.parent = parent;
        this.parentOffset = parentOffset;
        this.view = createView(data);
    }
    /** Internal method for pointers. */
    _enablePointers() {
        if (this.parent)
            return this.parent._enablePointers();
        if (this.bs)
            return;
        this.bs = Bitset.create(this.data.length);
        Bitset.setRange(this.bs, this.data.length, 0, this.pos, this.opts.allowMultipleReads);
    }
    markBytesBS(pos, len) {
        if (this.parent)
            return this.parent.markBytesBS(this.parentOffset + pos, len);
        if (!len)
            return true;
        if (!this.bs)
            return true;
        return Bitset.setRange(this.bs, this.data.length, pos, len, false);
    }
    markBytes(len) {
        const pos = this.pos;
        this.pos += len;
        const res = this.markBytesBS(pos, len);
        if (!this.opts.allowMultipleReads && !res)
            throw this.err(`multiple read pos=${this.pos} len=${len}`);
        return res;
    }
    pushObj(obj, objFn) {
        return Path.pushObj(this.stack, obj, objFn);
    }
    readView(n, fn) {
        if (!Number.isFinite(n))
            throw this.err(`readView: wrong length=${n}`);
        if (this.pos + n > this.data.length)
            throw this.err('readView: Unexpected end of buffer');
        const res = fn(this.view, this.pos);
        this.markBytes(n);
        return res;
    }
    // read bytes by absolute offset
    absBytes(n) {
        if (n > this.data.length)
            throw new Error('Unexpected end of buffer');
        return this.data.subarray(n);
    }
    finish() {
        if (this.opts.allowUnreadBytes)
            return;
        if (this.bitPos) {
            throw this.err(`${this.bitPos} bits left after unpack: ${baseHex.encode(this.data.slice(this.pos))}`);
        }
        if (this.bs && !this.parent) {
            const notRead = Bitset.indices(this.bs, this.data.length, true);
            if (notRead.length) {
                const formatted = Bitset.range(notRead)
                    .map(({ pos, length }) => `(${pos}/${length})[${baseHex.encode(this.data.subarray(pos, pos + length))}]`)
                    .join(', ');
                throw this.err(`unread byte ranges: ${formatted} (total=${this.data.length})`);
            }
            else
                return; // all bytes read, everything is ok
        }
        // Default: no pointers enabled
        if (!this.isEnd()) {
            throw this.err(`${this.leftBytes} bytes ${this.bitPos} bits left after unpack: ${baseHex.encode(this.data.slice(this.pos))}`);
        }
    }
    // User methods
    err(msg) {
        return Path.err('Reader', this.stack, msg);
    }
    offsetReader(n) {
        if (n > this.data.length)
            throw this.err('offsetReader: Unexpected end of buffer');
        return new _Reader(this.absBytes(n), this.opts, this.stack, this, n);
    }
    bytes(n, peek = false) {
        if (this.bitPos)
            throw this.err('readBytes: bitPos not empty');
        if (!Number.isFinite(n))
            throw this.err(`readBytes: wrong length=${n}`);
        if (this.pos + n > this.data.length)
            throw this.err('readBytes: Unexpected end of buffer');
        const slice = this.data.subarray(this.pos, this.pos + n);
        if (!peek)
            this.markBytes(n);
        return slice;
    }
    byte(peek = false) {
        if (this.bitPos)
            throw this.err('readByte: bitPos not empty');
        if (this.pos + 1 > this.data.length)
            throw this.err('readBytes: Unexpected end of buffer');
        const data = this.data[this.pos];
        if (!peek)
            this.markBytes(1);
        return data;
    }
    get leftBytes() {
        return this.data.length - this.pos;
    }
    get totalBytes() {
        return this.data.length;
    }
    isEnd() {
        return this.pos >= this.data.length && !this.bitPos;
    }
    // bits are read in BE mode (left to right): (0b1000_0000).readBits(1) == 1
    bits(bits) {
        if (bits > 32)
            throw this.err('BitReader: cannot read more than 32 bits in single call');
        let out = 0;
        while (bits) {
            if (!this.bitPos) {
                this.bitBuf = this.byte();
                this.bitPos = 8;
            }
            const take = Math.min(bits, this.bitPos);
            this.bitPos -= take;
            out = (out << take) | ((this.bitBuf >> this.bitPos) & (2 ** take - 1));
            this.bitBuf &= 2 ** this.bitPos - 1;
            bits -= take;
        }
        // Fix signed integers
        return out >>> 0;
    }
    find(needle, pos = this.pos) {
        if (!isBytes(needle))
            throw this.err(`find: needle is not bytes! ${needle}`);
        if (this.bitPos)
            throw this.err('findByte: bitPos not empty');
        if (!needle.length)
            throw this.err(`find: needle is empty`);
        // indexOf should be faster than full equalBytes check
        for (let idx = pos; (idx = this.data.indexOf(needle[0], idx)) !== -1; idx++) {
            if (idx === -1)
                return;
            const leftBytes = this.data.length - idx;
            if (leftBytes < needle.length)
                return;
            if (equalBytes(needle, this.data.subarray(idx, idx + needle.length)))
                return idx;
        }
        return;
    }
}
/**
 * Internal structure. Writer class for writing to a byte array.
 * The `stack` argument of constructor is internal, for debugging and logs.
 * @class Writer
 */
class _Writer {
    constructor(stack = []) {
        this.pos = 0;
        // We could have a single buffer here and re-alloc it with
        // x1.5-2 size each time it full, but it will be slower:
        // basic/encode bench: 395ns -> 560ns
        this.buffers = [];
        this.ptrs = [];
        this.bitBuf = 0;
        this.bitPos = 0;
        this.viewBuf = new Uint8Array(8);
        this.finished = false;
        this.stack = stack;
        this.view = createView(this.viewBuf);
    }
    pushObj(obj, objFn) {
        return Path.pushObj(this.stack, obj, objFn);
    }
    writeView(len, fn) {
        if (this.finished)
            throw this.err('buffer: finished');
        if (!isNum(len) || len > 8)
            throw new Error(`wrong writeView length=${len}`);
        fn(this.view);
        this.bytes(this.viewBuf.slice(0, len));
        this.viewBuf.fill(0);
    }
    // User methods
    err(msg) {
        if (this.finished)
            throw this.err('buffer: finished');
        return Path.err('Reader', this.stack, msg);
    }
    bytes(b) {
        if (this.finished)
            throw this.err('buffer: finished');
        if (this.bitPos)
            throw this.err('writeBytes: ends with non-empty bit buffer');
        this.buffers.push(b);
        this.pos += b.length;
    }
    byte(b) {
        if (this.finished)
            throw this.err('buffer: finished');
        if (this.bitPos)
            throw this.err('writeByte: ends with non-empty bit buffer');
        this.buffers.push(new Uint8Array([b]));
        this.pos++;
    }
    finish(clean = true) {
        if (this.finished)
            throw this.err('buffer: finished');
        if (this.bitPos)
            throw this.err('buffer: ends with non-empty bit buffer');
        // Can't use concatBytes, because it limits amount of arguments (65K).
        const buffers = this.buffers.concat(this.ptrs.map((i) => i.buffer));
        const sum = buffers.map((b) => b.length).reduce((a, b) => a + b, 0);
        const buf = new Uint8Array(sum);
        for (let i = 0, pad = 0; i < buffers.length; i++) {
            const a = buffers[i];
            buf.set(a, pad);
            pad += a.length;
        }
        for (let pos = this.pos, i = 0; i < this.ptrs.length; i++) {
            const ptr = this.ptrs[i];
            buf.set(ptr.ptr.encode(pos), ptr.pos);
            pos += ptr.buffer.length;
        }
        // Cleanup
        if (clean) {
            // We cannot cleanup buffers here, since it can be static user provided buffer.
            // Only '.byte' and '.bits' create buffer which we can safely clean.
            // for (const b of this.buffers) b.fill(0);
            this.buffers = [];
            for (const p of this.ptrs)
                p.buffer.fill(0);
            this.ptrs = [];
            this.finished = true;
            this.bitBuf = 0;
        }
        return buf;
    }
    bits(value, bits) {
        if (bits > 32)
            throw this.err('writeBits: cannot write more than 32 bits in single call');
        if (value >= 2 ** bits)
            throw this.err(`writeBits: value (${value}) >= 2**bits (${bits})`);
        while (bits) {
            const take = Math.min(bits, 8 - this.bitPos);
            this.bitBuf = (this.bitBuf << take) | (value >> (bits - take));
            this.bitPos += take;
            bits -= take;
            value &= 2 ** bits - 1;
            if (this.bitPos === 8) {
                this.bitPos = 0;
                this.buffers.push(new Uint8Array([this.bitBuf]));
                this.pos++;
            }
        }
    }
}
// Immutable LE<->BE
const swapEndianness = (b) => Uint8Array.from(b).reverse();
/** Internal function for checking bit bounds of bigint in signed/unsinged form */
function checkBounds(value, bits, signed) {
    if (signed) {
        // [-(2**(32-1)), 2**(32-1)-1]
        const signBit = 2n ** (bits - 1n);
        if (value < -signBit || value >= signBit)
            throw new Error(`value out of signed bounds. Expected ${-signBit} <= ${value} < ${signBit}`);
    }
    else {
        // [0, 2**32-1]
        if (0n > value || value >= 2n ** bits)
            throw new Error(`value out of unsigned bounds. Expected 0 <= ${value} < ${2n ** bits}`);
    }
}
function _wrap(inner) {
    return {
        // NOTE: we cannot export validate here, since it is likely mistake.
        encodeStream: inner.encodeStream,
        decodeStream: inner.decodeStream,
        size: inner.size,
        encode: (value) => {
            const w = new _Writer();
            inner.encodeStream(w, value);
            return w.finish();
        },
        decode: (data, opts = {}) => {
            const r = new _Reader(data, opts);
            const res = inner.decodeStream(r);
            r.finish();
            return res;
        },
    };
}
/**
 * Validates a value before encoding and after decoding using a provided function.
 * @param inner - The inner CoderType.
 * @param fn - The validation function.
 * @returns CoderType which check value with validation function.
 * @example
 * const val = (n: number) => {
 *   if (n > 10) throw new Error(`${n} > 10`);
 *   return n;
 * };
 *
 * const RangedInt = P.validate(P.U32LE, val); // Will check if value is <= 10 during encoding and decoding
 */
export function validate(inner, fn) {
    if (!isCoder(inner))
        throw new Error(`validate: invalid inner value ${inner}`);
    if (typeof fn !== 'function')
        throw new Error('validate: fn should be function');
    return _wrap({
        size: inner.size,
        encodeStream: (w, value) => {
            let res;
            try {
                res = fn(value);
            }
            catch (e) {
                throw w.err(e);
            }
            inner.encodeStream(w, res);
        },
        decodeStream: (r) => {
            const res = inner.decodeStream(r);
            try {
                return fn(res);
            }
            catch (e) {
                throw r.err(e);
            }
        },
    });
}
/**
 * Wraps a stream encoder into a generic encoder and optionally validation function
 * @param {inner} inner BytesCoderStream & { validate?: Validate<T> }.
 * @returns The wrapped CoderType.
 * @example
 * const U8 = P.wrap({
 *   encodeStream: (w: Writer, value: number) => w.byte(value),
 *   decodeStream: (r: Reader): number => r.byte()
 * });
 * const checkedU8 = P.wrap({
 *   encodeStream: (w: Writer, value: number) => w.byte(value),
 *   decodeStream: (r: Reader): number => r.byte()
 *   validate: (n: number) => {
 *    if (n > 10) throw new Error(`${n} > 10`);
 *    return n;
 *   }
 * });
 */
export const wrap = (inner) => {
    const res = _wrap(inner);
    return inner.validate ? validate(res, inner.validate) : res;
};
const isBaseCoder = (elm) => isPlainObject(elm) && typeof elm.decode === 'function' && typeof elm.encode === 'function';
/**
 * Checks if the given value is a CoderType.
 * @param elm - The value to check.
 * @returns True if the value is a CoderType, false otherwise.
 */
export function isCoder(elm) {
    return (isPlainObject(elm) &&
        isBaseCoder(elm) &&
        typeof elm.encodeStream === 'function' &&
        typeof elm.decodeStream === 'function' &&
        (elm.size === undefined || isNum(elm.size)));
}
// Coders (like in @scure/base) for common operations
/**
 * Base coder for working with dictionaries (records, objects, key-value map)
 * Dictionary is dynamic type like: `[key: string, value: any][]`
 * @returns base coder that encodes/decodes between arrays of key-value tuples and dictionaries.
 * @example
 * const dict: P.CoderType<Record<string, number>> = P.apply(
 *  P.array(P.U16BE, P.tuple([P.cstring, P.U32LE] as const)),
 *  P.coders.dict()
 * );
 */
function dict() {
    return {
        encode: (from) => {
            if (!Array.isArray(from))
                throw new Error('array expected');
            const to = {};
            for (const item of from) {
                if (!Array.isArray(item) || item.length !== 2)
                    throw new Error(`array of two elements expected`);
                const name = item[0];
                const value = item[1];
                if (to[name] !== undefined)
                    throw new Error(`key(${name}) appears twice in struct`);
                to[name] = value;
            }
            return to;
        },
        decode: (to) => {
            if (!isPlainObject(to))
                throw new Error(`expected plain object, got ${to}`);
            return Object.entries(to);
        },
    };
}
/**
 * Safely converts bigint to number.
 * Sometimes pointers / tags use u64 or other big numbers which cannot be represented by number,
 * but we still can use them since real value will be smaller than u32
 */
const numberBigint = {
    encode: (from) => {
        if (typeof from !== 'bigint')
            throw new Error(`expected bigint, got ${typeof from}`);
        if (from > BigInt(Number.MAX_SAFE_INTEGER))
            throw new Error(`element bigger than MAX_SAFE_INTEGER=${from}`);
        return Number(from);
    },
    decode: (to) => {
        if (!isNum(to))
            throw new Error('element is not a safe integer');
        return BigInt(to);
    },
};
/**
 * Base coder for working with TypeScript enums.
 * @param e - TypeScript enum.
 * @returns base coder that encodes/decodes between numbers and enum keys.
 * @example
 * enum Color { Red, Green, Blue }
 * const colorCoder = P.coders.tsEnum(Color);
 * colorCoder.encode(Color.Red); // 'Red'
 * colorCoder.decode('Green'); // 1
 */
function tsEnum(e) {
    if (!isPlainObject(e))
        throw new Error('plain object expected');
    return {
        encode: (from) => {
            if (!isNum(from) || !(from in e))
                throw new Error(`wrong value ${from}`);
            return e[from];
        },
        decode: (to) => {
            if (typeof to !== 'string')
                throw new Error(`wrong value ${typeof to}`);
            return e[to];
        },
    };
}
/**
 * Base coder for working with decimal numbers.
 * @param precision - Number of decimal places.
 * @param round - Round fraction part if bigger than precision (throws error by default)
 * @returns base coder that encodes/decodes between bigints and decimal strings.
 * @example
 * const decimal8 = P.coders.decimal(8);
 * decimal8.encode(630880845n); // '6.30880845'
 * decimal8.decode('6.30880845'); // 630880845n
 */
function decimal(precision, round = false) {
    if (!isNum(precision))
        throw new Error(`decimal/precision: wrong value ${precision}`);
    if (typeof round !== 'boolean')
        throw new Error(`decimal/round: expected boolean, got ${typeof round}`);
    const decimalMask = 10n ** BigInt(precision);
    return {
        encode: (from) => {
            if (typeof from !== 'bigint')
                throw new Error(`expected bigint, got ${typeof from}`);
            let s = (from < 0n ? -from : from).toString(10);
            let sep = s.length - precision;
            if (sep < 0) {
                s = s.padStart(s.length - sep, '0');
                sep = 0;
            }
            let i = s.length - 1;
            for (; i >= sep && s[i] === '0'; i--)
                ;
            let int = s.slice(0, sep);
            let frac = s.slice(sep, i + 1);
            if (!int)
                int = '0';
            if (from < 0n)
                int = '-' + int;
            if (!frac)
                return int;
            return `${int}.${frac}`;
        },
        decode: (to) => {
            if (typeof to !== 'string')
                throw new Error(`expected string, got ${typeof to}`);
            if (to === '-0')
                throw new Error(`negative zero is not allowed`);
            let neg = false;
            if (to.startsWith('-')) {
                neg = true;
                to = to.slice(1);
            }
            if (!/^(0|[1-9]\d*)(\.\d+)?$/.test(to))
                throw new Error(`wrong string value=${to}`);
            let sep = to.indexOf('.');
            sep = sep === -1 ? to.length : sep;
            // split by separator and strip trailing zeros from fraction. always returns [string, string] (.split doesn't).
            const intS = to.slice(0, sep);
            const fracS = to.slice(sep + 1).replace(/0+$/, '');
            const int = BigInt(intS) * decimalMask;
            if (!round && fracS.length > precision) {
                throw new Error(`fractional part cannot be represented with this precision (num=${to}, prec=${precision})`);
            }
            const fracLen = Math.min(fracS.length, precision);
            const frac = BigInt(fracS.slice(0, fracLen)) * 10n ** BigInt(precision - fracLen);
            const value = int + frac;
            return neg ? -value : value;
        },
    };
}
/**
 * Combines multiple coders into a single coder, allowing conditional encoding/decoding based on input.
 * Acts as a parser combinator, splitting complex conditional coders into smaller parts.
 *
 *   `encode = [Ae, Be]; decode = [Ad, Bd]`
 *   ->
 *   `match([{encode: Ae, decode: Ad}, {encode: Be; decode: Bd}])`
 *
 * @param lst - Array of coders to match.
 * @returns Combined coder for conditional encoding/decoding.
 */
function match(lst) {
    if (!Array.isArray(lst))
        throw new Error(`expected array, got ${typeof lst}`);
    for (const i of lst)
        if (!isBaseCoder(i))
            throw new Error(`wrong base coder ${i}`);
    return {
        encode: (from) => {
            for (const c of lst) {
                const elm = c.encode(from);
                if (elm !== undefined)
                    return elm;
            }
            throw new Error(`match/encode: cannot find match in ${from}`);
        },
        decode: (to) => {
            for (const c of lst) {
                const elm = c.decode(to);
                if (elm !== undefined)
                    return elm;
            }
            throw new Error(`match/decode: cannot find match in ${to}`);
        },
    };
}
/** Reverses direction of coder */
const reverse = (coder) => {
    if (!isBaseCoder(coder))
        throw new Error('BaseCoder expected');
    return { encode: coder.decode, decode: coder.encode };
};
export const coders = { dict, numberBigint, tsEnum, decimal, match, reverse };
/**
 * CoderType for parsing individual bits.
 * NOTE: Structure should parse whole amount of bytes before it can start parsing byte-level elements.
 * @param len - Number of bits to parse.
 * @returns CoderType representing the parsed bits.
 * @example
 * const s = P.struct({ magic: P.bits(1), version: P.bits(1), tag: P.bits(4), len: P.bits(2) });
 */
export const bits = (len) => {
    if (!isNum(len))
        throw new Error(`bits: wrong length ${len} (${typeof len})`);
    return wrap({
        encodeStream: (w, value) => w.bits(value, len),
        decodeStream: (r) => r.bits(len),
        validate: (value) => {
            if (!isNum(value))
                throw new Error(`bits: wrong value ${value}`);
            return value;
        },
    });
};
/**
 * CoderType for working with bigint values.
 * Unsized bigint values should be wrapped in a container (e.g., bytes or string).
 *
 * `0n = new Uint8Array([])`
 *
 * `1n = new Uint8Array([1n])`
 *
 * Please open issue, if you need different behavior for zero.
 *
 * @param size - Size of the bigint in bytes.
 * @param le - Whether to use little-endian byte order.
 * @param signed - Whether the bigint is signed.
 * @param sized - Whether the bigint should have a fixed size.
 * @returns CoderType representing the bigint value.
 * @example
 * const U512BE = P.bigint(64, false, true, true); // Define a CoderType for a 512-bit unsigned big-endian integer
 */
export const bigint = (size, le = false, signed = false, sized = true) => {
    if (!isNum(size))
        throw new Error(`bigint/size: wrong value ${size}`);
    if (typeof le !== 'boolean')
        throw new Error(`bigint/le: expected boolean, got ${typeof le}`);
    if (typeof signed !== 'boolean')
        throw new Error(`bigint/signed: expected boolean, got ${typeof signed}`);
    if (typeof sized !== 'boolean')
        throw new Error(`bigint/sized: expected boolean, got ${typeof sized}`);
    const bLen = BigInt(size);
    const signBit = 2n ** (8n * bLen - 1n);
    return wrap({
        size: sized ? size : undefined,
        encodeStream: (w, value) => {
            if (signed && value < 0)
                value = value | signBit;
            const b = [];
            for (let i = 0; i < size; i++) {
                b.push(Number(value & 255n));
                value >>= 8n;
            }
            let res = new Uint8Array(b).reverse();
            if (!sized) {
                let pos = 0;
                for (pos = 0; pos < res.length; pos++)
                    if (res[pos] !== 0)
                        break;
                res = res.subarray(pos); // remove leading zeros
            }
            w.bytes(le ? res.reverse() : res);
        },
        decodeStream: (r) => {
            // TODO: for le we can read until first zero?
            const value = r.bytes(sized ? size : Math.min(size, r.leftBytes));
            const b = le ? value : swapEndianness(value);
            let res = 0n;
            for (let i = 0; i < b.length; i++)
                res |= BigInt(b[i]) << (8n * BigInt(i));
            if (signed && res & signBit)
                res = (res ^ signBit) - signBit;
            return res;
        },
        validate: (value) => {
            if (typeof value !== 'bigint')
                throw new Error(`bigint: invalid value: ${value}`);
            checkBounds(value, 8n * bLen, !!signed);
            return value;
        },
    });
};
/** Unsigned 256-bit little-endian integer CoderType. */
export const U256LE = /* @__PURE__ */ bigint(32, true);
/** Unsigned 256-bit big-endian integer CoderType. */
export const U256BE = /* @__PURE__ */ bigint(32, false);
/** Signed 256-bit little-endian integer CoderType. */
export const I256LE = /* @__PURE__ */ bigint(32, true, true);
/** Signed 256-bit big-endian integer CoderType. */
export const I256BE = /* @__PURE__ */ bigint(32, false, true);
/** Unsigned 128-bit little-endian integer CoderType. */
export const U128LE = /* @__PURE__ */ bigint(16, true);
/** Unsigned 128-bit big-endian integer CoderType. */
export const U128BE = /* @__PURE__ */ bigint(16, false);
/** Signed 128-bit little-endian integer CoderType. */
export const I128LE = /* @__PURE__ */ bigint(16, true, true);
/** Signed 128-bit big-endian integer CoderType. */
export const I128BE = /* @__PURE__ */ bigint(16, false, true);
/** Unsigned 64-bit little-endian integer CoderType. */
export const U64LE = /* @__PURE__ */ bigint(8, true);
/** Unsigned 64-bit big-endian integer CoderType. */
export const U64BE = /* @__PURE__ */ bigint(8, false);
/** Signed 64-bit little-endian integer CoderType. */
export const I64LE = /* @__PURE__ */ bigint(8, true, true);
/** Signed 64-bit big-endian integer CoderType. */
export const I64BE = /* @__PURE__ */ bigint(8, false, true);
/**
 * CoderType for working with numbber values (up to 6 bytes/48 bits).
 * Unsized int values should be wrapped in a container (e.g., bytes or string).
 *
 * `0 = new Uint8Array([])`
 *
 * `1 = new Uint8Array([1n])`
 *
 * Please open issue, if you need different behavior for zero.
 *
 * @param size - Size of the number in bytes.
 * @param le - Whether to use little-endian byte order.
 * @param signed - Whether the number is signed.
 * @param sized - Whether the number should have a fixed size.
 * @returns CoderType representing the number value.
 * @example
 * const uint64BE = P.bigint(8, false, true); // Define a CoderType for a 64-bit unsigned big-endian integer
 */
export const int = (size, le = false, signed = false, sized = true) => {
    if (!isNum(size))
        throw new Error(`int/size: wrong value ${size}`);
    if (typeof le !== 'boolean')
        throw new Error(`int/le: expected boolean, got ${typeof le}`);
    if (typeof signed !== 'boolean')
        throw new Error(`int/signed: expected boolean, got ${typeof signed}`);
    if (typeof sized !== 'boolean')
        throw new Error(`int/sized: expected boolean, got ${typeof sized}`);
    if (size > 6)
        throw new Error('int supports size up to 6 bytes (48 bits): use bigints instead');
    return apply(bigint(size, le, signed, sized), coders.numberBigint);
};
const view = (len, opts) => wrap({
    size: len,
    encodeStream: (w, value) => w.writeView(len, (view) => opts.write(view, value)),
    decodeStream: (r) => r.readView(len, opts.read),
    validate: (value) => {
        if (typeof value !== 'number')
            throw new Error(`viewCoder: expected number, got ${typeof value}`);
        if (opts.validate)
            opts.validate(value);
        return value;
    },
});
const intView = (len, signed, opts) => {
    const bits = len * 8;
    const signBit = 2 ** (bits - 1);
    // Inlined checkBounds for integer
    const validateSigned = (value) => {
        if (!isNum(value))
            throw new Error(`sintView: value is not safe integer: ${value}`);
        if (value < -signBit || value >= signBit) {
            throw new Error(`sintView: value out of bounds. Expected ${-signBit} <= ${value} < ${signBit}`);
        }
    };
    const maxVal = 2 ** bits;
    const validateUnsigned = (value) => {
        if (!isNum(value))
            throw new Error(`uintView: value is not safe integer: ${value}`);
        if (0 > value || value >= maxVal) {
            throw new Error(`uintView: value out of bounds. Expected 0 <= ${value} < ${maxVal}`);
        }
    };
    return view(len, {
        write: opts.write,
        read: opts.read,
        validate: signed ? validateSigned : validateUnsigned,
    });
};
/** Unsigned 32-bit little-endian integer CoderType. */
export const U32LE = /* @__PURE__ */ intView(4, false, {
    read: (view, pos) => view.getUint32(pos, true),
    write: (view, value) => view.setUint32(0, value, true),
});
/** Unsigned 32-bit big-endian integer CoderType. */
export const U32BE = /* @__PURE__ */ intView(4, false, {
    read: (view, pos) => view.getUint32(pos, false),
    write: (view, value) => view.setUint32(0, value, false),
});
/** Signed 32-bit little-endian integer CoderType. */
export const I32LE = /* @__PURE__ */ intView(4, true, {
    read: (view, pos) => view.getInt32(pos, true),
    write: (view, value) => view.setInt32(0, value, true),
});
/** Signed 32-bit big-endian integer CoderType. */
export const I32BE = /* @__PURE__ */ intView(4, true, {
    read: (view, pos) => view.getInt32(pos, false),
    write: (view, value) => view.setInt32(0, value, false),
});
/** Unsigned 16-bit little-endian integer CoderType. */
export const U16LE = /* @__PURE__ */ intView(2, false, {
    read: (view, pos) => view.getUint16(pos, true),
    write: (view, value) => view.setUint16(0, value, true),
});
/** Unsigned 16-bit big-endian integer CoderType. */
export const U16BE = /* @__PURE__ */ intView(2, false, {
    read: (view, pos) => view.getUint16(pos, false),
    write: (view, value) => view.setUint16(0, value, false),
});
/** Signed 16-bit little-endian integer CoderType. */
export const I16LE = /* @__PURE__ */ intView(2, true, {
    read: (view, pos) => view.getInt16(pos, true),
    write: (view, value) => view.setInt16(0, value, true),
});
/** Signed 16-bit big-endian integer CoderType. */
export const I16BE = /* @__PURE__ */ intView(2, true, {
    read: (view, pos) => view.getInt16(pos, false),
    write: (view, value) => view.setInt16(0, value, false),
});
/** Unsigned 8-bit integer CoderType. */
export const U8 = /* @__PURE__ */ intView(1, false, {
    read: (view, pos) => view.getUint8(pos),
    write: (view, value) => view.setUint8(0, value),
});
/** Signed 8-bit integer CoderType. */
export const I8 = /* @__PURE__ */ intView(1, true, {
    read: (view, pos) => view.getInt8(pos),
    write: (view, value) => view.setInt8(0, value),
});
// Floats
const f32 = (le) => view(4, {
    read: (view, pos) => view.getFloat32(pos, le),
    write: (view, value) => view.setFloat32(0, value, le),
    validate: (value) => {
        if (Math.fround(value) !== value && !Number.isNaN(value))
            throw new Error(`f32: wrong value=${value}`);
    },
});
const f64 = (le) => view(8, {
    read: (view, pos) => view.getFloat64(pos, le),
    write: (view, value) => view.setFloat64(0, value, le),
});
/** 32-bit big-endian floating point CoderType ("binary32", IEEE 754-2008). */
export const F32BE = /* @__PURE__ */ f32(false);
/** 32-bit little-endian floating point  CoderType ("binary32", IEEE 754-2008). */
export const F32LE = /* @__PURE__ */ f32(true);
/** A 64-bit big-endian floating point type ("binary64", IEEE 754-2008). Any JS number can be encoded. */
export const F64BE = /* @__PURE__ */ f64(false);
/** A 64-bit little-endian floating point type ("binary64", IEEE 754-2008). Any JS number can be encoded. */
export const F64LE = /* @__PURE__ */ f64(true);
/** Boolean CoderType. */
export const bool = /* @__PURE__ */ wrap({
    size: 1,
    encodeStream: (w, value) => w.byte(value ? 1 : 0),
    decodeStream: (r) => {
        const value = r.byte();
        if (value !== 0 && value !== 1)
            throw r.err(`bool: invalid value ${value}`);
        return value === 1;
    },
    validate: (value) => {
        if (typeof value !== 'boolean')
            throw new Error(`bool: invalid value ${value}`);
        return value;
    },
});
/**
 * Bytes CoderType with a specified length and endianness.
 * The bytes can have:
 * - Dynamic size (prefixed with a length CoderType like U16BE)
 * - Fixed size (specified by a number)
 * - Unknown size (null, will parse until end of buffer)
 * - Zero-terminated (terminator can be any Uint8Array)
 * @param len - CoderType, number, Uint8Array (terminator) or null
 * @param le - Whether to use little-endian byte order.
 * @returns CoderType representing the bytes.
 * @example
 * // Dynamic size bytes (prefixed with P.U16BE number of bytes length)
 * const dynamicBytes = P.bytes(P.U16BE, false);
 * const fixedBytes = P.bytes(32, false); // Fixed size bytes
 * const unknownBytes = P.bytes(null, false); // Unknown size bytes, will parse until end of buffer
 * const zeroTerminatedBytes = P.bytes(new Uint8Array([0]), false); // Zero-terminated bytes
 */
const createBytes = (len, le = false) => {
    if (typeof le !== 'boolean')
        throw new Error(`bytes/le: expected boolean, got ${typeof le}`);
    const _length = lengthCoder(len);
    const _isb = isBytes(len);
    return wrap({
        size: typeof len === 'number' ? len : undefined,
        encodeStream: (w, value) => {
            if (!_isb)
                _length.encodeStream(w, value.length);
            w.bytes(le ? swapEndianness(value) : value);
            if (_isb)
                w.bytes(len);
        },
        decodeStream: (r) => {
            let bytes;
            if (_isb) {
                const tPos = r.find(len);
                if (!tPos)
                    throw r.err(`bytes: cannot find terminator`);
                bytes = r.bytes(tPos - r.pos);
                r.bytes(len.length);
            }
            else {
                bytes = r.bytes(len === null ? r.leftBytes : _length.decodeStream(r));
            }
            return le ? swapEndianness(bytes) : bytes;
        },
        validate: (value) => {
            if (!isBytes(value))
                throw new Error(`bytes: invalid value ${value}`);
            return value;
        },
    });
};
export { createBytes as bytes, createHex as hex };
/**
 * Prefix-encoded value using a length prefix and an inner CoderType.
 * The prefix can have:
 * - Dynamic size (prefixed with a length CoderType like U16BE)
 * - Fixed size (specified by a number)
 * - Unknown size (null, will parse until end of buffer)
 * - Zero-terminated (terminator can be any Uint8Array)
 * @param len - Length CoderType (dynamic size), number (fixed size), Uint8Array (for terminator), or null (will parse until end of buffer)
 * @param inner - CoderType for the actual value to be prefix-encoded.
 * @returns CoderType representing the prefix-encoded value.
 * @example
 * const dynamicPrefix = P.prefix(P.U16BE, P.bytes(null)); // Dynamic size prefix (prefixed with P.U16BE number of bytes length)
 * const fixedPrefix = P.prefix(10, P.bytes(null)); // Fixed size prefix (always 10 bytes)
 */
export function prefix(len, inner) {
    if (!isCoder(inner))
        throw new Error(`prefix: invalid inner value ${inner}`);
    return apply(createBytes(len), reverse(inner));
}
/**
 * String CoderType with a specified length and endianness.
 * The string can be:
 * - Dynamic size (prefixed with a length CoderType like U16BE)
 * - Fixed size (specified by a number)
 * - Unknown size (null, will parse until end of buffer)
 * - Zero-terminated (terminator can be any Uint8Array)
 * @param len - Length CoderType (dynamic size), number (fixed size), Uint8Array (for terminator), or null (will parse until end of buffer)
 * @param le - Whether to use little-endian byte order.
 * @returns CoderType representing the string.
 * @example
 * const dynamicString = P.string(P.U16BE, false); // Dynamic size string (prefixed with P.U16BE number of string length)
 * const fixedString = P.string(10, false); // Fixed size string
 * const unknownString = P.string(null, false); // Unknown size string, will parse until end of buffer
 * const nullTerminatedString = P.cstring; // NUL-terminated string
 * const _cstring = P.string(new Uint8Array([0])); // Same thing
 */
export const string = (len, le = false) => validate(apply(createBytes(len, le), utf8), (value) => {
    // TextEncoder/TextDecoder will fail on non-string, but we create more readable errors earlier
    if (typeof value !== 'string')
        throw new Error(`expected string, got ${typeof value}`);
    return value;
});
/** NUL-terminated string CoderType. */
export const cstring = /* @__PURE__ */ string(NULL);
/**
 * Hexadecimal string CoderType with a specified length, endianness, and optional 0x prefix.
 * @param len - Length CoderType (dynamic size), number (fixed size), Uint8Array (for terminator), or null (will parse until end of buffer)
 * @param le - Whether to use little-endian byte order.
 * @param withZero - Whether to include the 0x prefix.
 * @returns CoderType representing the hexadecimal string.
 * @example
 * const dynamicHex = P.hex(P.U16BE, {isLE: false, with0x: true}); // Hex string with 0x prefix and U16BE length
 * const fixedHex = P.hex(32, {isLE: false, with0x: false}); // Fixed-length 32-byte hex string without 0x prefix
 */
const createHex = (len, options = { isLE: false, with0x: false }) => {
    let inner = apply(createBytes(len, options.isLE), baseHex);
    const prefix = options.with0x;
    if (typeof prefix !== 'boolean')
        throw new Error(`hex/with0x: expected boolean, got ${typeof prefix}`);
    if (prefix) {
        inner = apply(inner, {
            encode: (value) => `0x${value}`,
            decode: (value) => {
                if (!value.startsWith('0x'))
                    throw new Error('hex(with0x=true).encode input should start with 0x');
                return value.slice(2);
            },
        });
    }
    return inner;
};
/**
 * Applies a base coder to a CoderType.
 * @param inner - The inner CoderType.
 * @param b - The base coder to apply.
 * @returns CoderType representing the transformed value.
 * @example
 * import { hex } from '@scure/base';
 * const hex = P.apply(P.bytes(32), hex); // will decode bytes into a hex string
 */
export function apply(inner, base) {
    if (!isCoder(inner))
        throw new Error(`apply: invalid inner value ${inner}`);
    if (!isBaseCoder(base))
        throw new Error(`apply: invalid base value ${inner}`);
    return wrap({
        size: inner.size,
        encodeStream: (w, value) => {
            let innerValue;
            try {
                innerValue = base.decode(value);
            }
            catch (e) {
                throw w.err('' + e);
            }
            return inner.encodeStream(w, innerValue);
        },
        decodeStream: (r) => {
            const innerValue = inner.decodeStream(r);
            try {
                return base.encode(innerValue);
            }
            catch (e) {
                throw r.err('' + e);
            }
        },
    });
}
/**
 * Lazy CoderType that is evaluated at runtime.
 * @param fn - A function that returns the CoderType.
 * @returns CoderType representing the lazy value.
 * @example
 * type Tree = { name: string; children: Tree[] };
 * const tree = P.struct({
 *   name: P.cstring,
 *   children: P.array(
 *     P.U16BE,
 *     P.lazy((): P.CoderType<Tree> => tree)
 *   ),
 * });
 */
export function lazy(fn) {
    if (typeof fn !== 'function')
        throw new Error(`lazy: expected function, got ${typeof fn}`);
    return wrap({
        encodeStream: (w, value) => fn().encodeStream(w, value),
        decodeStream: (r) => fn().decodeStream(r),
    });
}
/**
 * Flag CoderType that encodes/decodes a boolean value based on the presence of a marker.
 * @param flagValue - Marker value.
 * @param xor - Whether to invert the flag behavior.
 * @returns CoderType representing the flag value.
 * @example
 * const flag = P.flag(new Uint8Array([0x01, 0x02])); // Encodes true as u8a([0x01, 0x02]), false as u8a([])
 * const flagXor = P.flag(new Uint8Array([0x01, 0x02]), true); // Encodes true as u8a([]), false as u8a([0x01, 0x02])
 * // Conditional encoding with flagged
 * const s = P.struct({ f: P.flag(new Uint8Array([0x0, 0x1])), f2: P.flagged('f', P.U32BE) });
 */
export const flag = (flagValue, xor = false) => {
    if (!isBytes(flagValue))
        throw new Error(`flag/flagValue: expected Uint8Array, got ${typeof flagValue}`);
    if (typeof xor !== 'boolean')
        throw new Error(`flag/xor: expected boolean, got ${typeof xor}`);
    return wrap({
        size: flagValue.length,
        encodeStream: (w, value) => {
            if (!!value !== xor)
                w.bytes(flagValue);
        },
        decodeStream: (r) => {
            let hasFlag = r.leftBytes >= flagValue.length;
            if (hasFlag) {
                hasFlag = equalBytes(r.bytes(flagValue.length, true), flagValue);
                // Found flag, advance cursor position
                if (hasFlag)
                    r.bytes(flagValue.length);
            }
            return hasFlag !== xor; // hasFlag ^ xor
        },
        validate: (value) => {
            if (value !== undefined && typeof value !== 'boolean')
                throw new Error(`flag: expected boolean value or undefined, got ${typeof value}`);
            return value;
        },
    });
};
/**
 * Conditional CoderType that encodes/decodes a value only if a flag is present.
 * @param path - Path to the flag value or a CoderType for the flag.
 * @param inner - Inner CoderType for the value.
 * @param def - Optional default value to use if the flag is not present.
 * @returns CoderType representing the conditional value.
 * @example
 * const s = P.struct({
 *   f: P.flag(new Uint8Array([0x0, 0x1])),
 *   f2: P.flagged('f', P.U32BE)
 * });
 *
 * @example
 * const s2 = P.struct({
 *   f: P.flag(new Uint8Array([0x0, 0x1])),
 *   f2: P.flagged('f', P.U32BE, 123)
 * });
 */
export function flagged(path, inner, def) {
    if (!isCoder(inner))
        throw new Error(`flagged: invalid inner value ${inner}`);
    if (typeof path !== 'string' && !isCoder(inner))
        throw new Error(`flagged: wrong path=${path}`);
    return wrap({
        encodeStream: (w, value) => {
            if (typeof path === 'string') {
                if (Path.resolve(w.stack, path))
                    inner.encodeStream(w, value);
                else if (def)
                    inner.encodeStream(w, def);
            }
            else {
                path.encodeStream(w, !!value);
                if (!!value)
                    inner.encodeStream(w, value);
                else if (def)
                    inner.encodeStream(w, def);
            }
        },
        decodeStream: (r) => {
            let hasFlag = false;
            if (typeof path === 'string')
                hasFlag = !!Path.resolve(r.stack, path);
            else
                hasFlag = path.decodeStream(r);
            // If there is a flag -- decode and return value
            if (hasFlag)
                return inner.decodeStream(r);
            else if (def)
                inner.decodeStream(r);
            return;
        },
    });
}
/**
 * Optional CoderType that encodes/decodes a value based on a flag.
 * @param flag - CoderType for the flag value.
 * @param inner - Inner CoderType for the value.
 * @param def - Optional default value to use if the flag is not present.
 * @returns CoderType representing the optional value.
 * @example
 * // Will decode into P.U32BE only if flag present
 * const optional = P.optional(P.flag(new Uint8Array([0x0, 0x1])), P.U32BE);
 *
 * @example
 * // If no flag present, will decode into default value
 * const optionalWithDefault = P.optional(P.flag(new Uint8Array([0x0, 0x1])), P.U32BE, 123);
 */
export function optional(flag, inner, def) {
    if (!isCoder(flag) || !isCoder(inner))
        throw new Error(`optional: invalid flag or inner value flag=${flag} inner=${inner}`);
    return wrap({
        size: def !== undefined && flag.size && inner.size ? flag.size + inner.size : undefined,
        encodeStream: (w, value) => {
            flag.encodeStream(w, !!value);
            if (value)
                inner.encodeStream(w, value);
            else if (def !== undefined)
                inner.encodeStream(w, def);
        },
        decodeStream: (r) => {
            if (flag.decodeStream(r))
                return inner.decodeStream(r);
            else if (def !== undefined)
                inner.decodeStream(r);
            return;
        },
    });
}
/**
 * Magic value CoderType that encodes/decodes a constant value.
 * This can be used to check for a specific magic value or sequence of bytes at the beginning of a data structure.
 * @param inner - Inner CoderType for the value.
 * @param constant - Constant value.
 * @param check - Whether to check the decoded value against the constant.
 * @returns CoderType representing the magic value.
 * @example
 * // Always encodes constant as bytes using inner CoderType, throws if encoded value is not present
 * const magicU8 = P.magic(P.U8, 0x42);
 */
export function magic(inner, constant, check = true) {
    if (!isCoder(inner))
        throw new Error(`magic: invalid inner value ${inner}`);
    if (typeof check !== 'boolean')
        throw new Error(`magic: expected boolean, got ${typeof check}`);
    return wrap({
        size: inner.size,
        encodeStream: (w, _value) => inner.encodeStream(w, constant),
        decodeStream: (r) => {
            const value = inner.decodeStream(r);
            if ((check && typeof value !== 'object' && value !== constant) ||
                (isBytes(constant) && !equalBytes(constant, value))) {
                throw r.err(`magic: invalid value: ${value} !== ${constant}`);
            }
            return;
        },
        validate: (value) => {
            if (value !== undefined)
                throw new Error(`magic: wrong value=${typeof value}`);
            return value;
        },
    });
}
/**
 * Magic bytes CoderType that encodes/decodes a constant byte array or string.
 * @param constant - Constant byte array or string.
 * @returns CoderType representing the magic bytes.
 * @example
 * // Always encodes undefined into byte representation of string 'MAGIC'
 * const magicBytes = P.magicBytes('MAGIC');
 */
export const magicBytes = (constant) => {
    const c = typeof constant === 'string' ? utf8.decode(constant) : constant;
    return magic(createBytes(c.length), c);
};
/**
 * Creates a CoderType for a constant value. The function enforces this value during encoding,
 * ensuring it matches the provided constant. During decoding, it always returns the constant value.
 * The actual value is not written to or read from any byte stream; it's used only for validation.
 *
 * @param c - Constant value.
 * @returns CoderType representing the constant value.
 * @example
 * // Always return 123 on decode, throws on encoding anything other than 123
 * const constantU8 = P.constant(123);
 */
export function constant(c) {
    return wrap({
        encodeStream: (_w, value) => {
            if (value !== c)
                throw new Error(`constant: invalid value ${value} (exp: ${c})`);
        },
        decodeStream: (_r) => c,
    });
}
function sizeof(fields) {
    let size = 0;
    for (const f of fields) {
        if (f.size === undefined)
            return;
        if (!isNum(f.size))
            throw new Error(`sizeof: wrong element size=${size}`);
        size += f.size;
    }
    return size;
}
/**
 * Structure of composable primitives (C/Rust struct)
 * @param fields - Object mapping field names to CoderTypes.
 * @returns CoderType representing the structure.
 * @example
 * // Define a structure with a 32-bit big-endian unsigned integer, a string, and a nested structure
 * const myStruct = P.struct({
 *   id: P.U32BE,
 *   name: P.string(P.U8),
 *   nested: P.struct({
 *     flag: P.bool,
 *     value: P.I16LE
 *   })
 * });
 */
export function struct(fields) {
    if (!isPlainObject(fields))
        throw new Error(`struct: expected plain object, got ${fields}`);
    for (const name in fields) {
        if (!isCoder(fields[name]))
            throw new Error(`struct: field ${name} is not CoderType`);
    }
    return wrap({
        size: sizeof(Object.values(fields)),
        encodeStream: (w, value) => {
            w.pushObj(value, (fieldFn) => {
                for (const name in fields)
                    fieldFn(name, () => fields[name].encodeStream(w, value[name]));
            });
        },
        decodeStream: (r) => {
            const res = {};
            r.pushObj(res, (fieldFn) => {
                for (const name in fields)
                    fieldFn(name, () => (res[name] = fields[name].decodeStream(r)));
            });
            return res;
        },
        validate: (value) => {
            if (typeof value !== 'object' || value === null)
                throw new Error(`struct: invalid value ${value}`);
            return value;
        },
    });
}
/**
 * Tuple (unnamed structure) of CoderTypes. Same as struct but with unnamed fields.
 * @param fields - Array of CoderTypes.
 * @returns CoderType representing the tuple.
 * @example
 * const myTuple = P.tuple([P.U8, P.U16LE, P.string(P.U8)]);
 */
export function tuple(fields) {
    if (!Array.isArray(fields))
        throw new Error(`Packed.Tuple: got ${typeof fields} instead of array`);
    for (let i = 0; i < fields.length; i++) {
        if (!isCoder(fields[i]))
            throw new Error(`tuple: field ${i} is not CoderType`);
    }
    return wrap({
        size: sizeof(fields),
        encodeStream: (w, value) => {
            // TODO: fix types
            if (!Array.isArray(value))
                throw w.err(`tuple: invalid value ${value}`);
            w.pushObj(value, (fieldFn) => {
                for (let i = 0; i < fields.length; i++)
                    fieldFn(`${i}`, () => fields[i].encodeStream(w, value[i]));
            });
        },
        decodeStream: (r) => {
            const res = [];
            r.pushObj(res, (fieldFn) => {
                for (let i = 0; i < fields.length; i++)
                    fieldFn(`${i}`, () => res.push(fields[i].decodeStream(r)));
            });
            return res;
        },
        validate: (value) => {
            if (!Array.isArray(value))
                throw new Error(`tuple: invalid value ${value}`);
            if (value.length !== fields.length)
                throw new Error(`tuple: wrong length=${value.length}, expected ${fields.length}`);
            return value;
        },
    });
}
/**
 * Array of items (inner type) with a specified length.
 * @param len - Length CoderType (dynamic size), number (fixed size), Uint8Array (for terminator), or null (will parse until end of buffer)
 * @param inner - CoderType for encoding/decoding each array item.
 * @returns CoderType representing the array.
 * @example
 * const a1 = P.array(P.U16BE, child); // Dynamic size array (prefixed with P.U16BE number of array length)
 * const a2 = P.array(4, child); // Fixed size array
 * const a3 = P.array(null, child); // Unknown size array, will parse until end of buffer
 * const a4 = P.array(new Uint8Array([0]), child); // zero-terminated array (NOTE: terminator can be any buffer)
 */
export function array(len, inner) {
    if (!isCoder(inner))
        throw new Error(`array: invalid inner value ${inner}`);
    // By construction length is inside array (otherwise there will be various incorrect stack states)
    // But forcing users always write '..' seems like bad idea. Also, breaking change.
    const _length = lengthCoder(typeof len === 'string' ? `../${len}` : len);
    return wrap({
        size: typeof len === 'number' && inner.size ? len * inner.size : undefined,
        encodeStream: (w, value) => {
            const _w = w;
            _w.pushObj(value, (fieldFn) => {
                if (!isBytes(len))
                    _length.encodeStream(w, value.length);
                for (let i = 0; i < value.length; i++) {
                    fieldFn(`${i}`, () => {
                        const elm = value[i];
                        const startPos = w.pos;
                        inner.encodeStream(w, elm);
                        if (isBytes(len)) {
                            // Terminator is bigger than elm size, so skip
                            if (len.length > _w.pos - startPos)
                                return;
                            const data = _w.finish(false).subarray(startPos, _w.pos);
                            // There is still possible case when multiple elements create terminator,
                            // but it is hard to catch here, will be very slow
                            if (equalBytes(data.subarray(0, len.length), len))
                                throw _w.err(`array: inner element encoding same as separator. elm=${elm} data=${data}`);
                        }
                    });
                }
            });
            if (isBytes(len))
                w.bytes(len);
        },
        decodeStream: (r) => {
            const res = [];
            r.pushObj(res, (fieldFn) => {
                if (len === null) {
                    for (let i = 0; !r.isEnd(); i++) {
                        fieldFn(`${i}`, () => res.push(inner.decodeStream(r)));
                        if (inner.size && r.leftBytes < inner.size)
                            break;
                    }
                }
                else if (isBytes(len)) {
                    for (let i = 0;; i++) {
                        if (equalBytes(r.bytes(len.length, true), len)) {
                            // Advance cursor position if terminator found
                            r.bytes(len.length);
                            break;
                        }
                        fieldFn(`${i}`, () => res.push(inner.decodeStream(r)));
                    }
                }
                else {
                    let length;
                    fieldFn('arrayLen', () => (length = _length.decodeStream(r)));
                    for (let i = 0; i < length; i++)
                        fieldFn(`${i}`, () => res.push(inner.decodeStream(r)));
                }
            });
            return res;
        },
        validate: (value) => {
            if (!Array.isArray(value))
                throw new Error(`array: invalid value ${value}`);
            return value;
        },
    });
}
/**
 * Mapping between encoded values and string representations.
 * @param inner - CoderType for encoded values.
 * @param variants - Object mapping string representations to encoded values.
 * @returns CoderType representing the mapping.
 * @example
 * // Map between numbers and strings
 * const numberMap = P.map(P.U8, {
 *   'one': 1,
 *   'two': 2,
 *   'three': 3
 * });
 *
 * // Map between byte arrays and strings
 * const byteMap = P.map(P.bytes(2, false), {
 *   'ab': Uint8Array.from([0x61, 0x62]),
 *   'cd': Uint8Array.from([0x63, 0x64])
 * });
 */
export function map(inner, variants) {
    if (!isCoder(inner))
        throw new Error(`map: invalid inner value ${inner}`);
    if (!isPlainObject(variants))
        throw new Error(`map: variants should be plain object`);
    const variantNames = new Map();
    for (const k in variants)
        variantNames.set(variants[k], k);
    return wrap({
        size: inner.size,
        encodeStream: (w, value) => inner.encodeStream(w, variants[value]),
        decodeStream: (r) => {
            const variant = inner.decodeStream(r);
            const name = variantNames.get(variant);
            if (name === undefined)
                throw r.err(`Enum: unknown value: ${variant} ${Array.from(variantNames.keys())}`);
            return name;
        },
        validate: (value) => {
            if (typeof value !== 'string')
                throw new Error(`map: invalid value ${value}`);
            if (!(value in variants))
                throw new Error(`Map: unknown variant: ${value}`);
            return value;
        },
    });
}
/**
 * Tagged union of CoderTypes, where the tag value determines which CoderType to use.
 * The decoded value will have the structure `{ TAG: number, data: ... }`.
 * @param tag - CoderType for the tag value.
 * @param variants - Object mapping tag values to CoderTypes.
 * @returns CoderType representing the tagged union.
 * @example
 * // Tagged union of array, string, and number
 * // Depending on the value of the first byte, it will be decoded as an array, string, or number.
 * const taggedUnion = P.tag(P.U8, {
 *   0x01: P.array(P.U16LE, P.U8),
 *   0x02: P.string(P.U8),
 *   0x03: P.U32BE
 * });
 *
 * const encoded = taggedUnion.encode({ TAG: 0x01, data: 'hello' }); // Encodes the string 'hello' with tag 0x01
 * const decoded = taggedUnion.decode(encoded); // Decodes the encoded value back to { TAG: 0x01, data: 'hello' }
 */
export function tag(tag, variants) {
    if (!isCoder(tag))
        throw new Error(`tag: invalid tag value ${tag}`);
    if (!isPlainObject(variants))
        throw new Error(`tag: variants should be plain object`);
    for (const name in variants) {
        if (!isCoder(variants[name]))
            throw new Error(`tag: variant ${name} is not CoderType`);
    }
    return wrap({
        size: tag.size,
        encodeStream: (w, value) => {
            const { TAG, data } = value;
            const dataType = variants[TAG];
            tag.encodeStream(w, TAG);
            dataType.encodeStream(w, data);
        },
        decodeStream: (r) => {
            const TAG = tag.decodeStream(r);
            const dataType = variants[TAG];
            if (!dataType)
                throw r.err(`Tag: invalid tag ${TAG}`);
            return { TAG, data: dataType.decodeStream(r) };
        },
        validate: (value) => {
            const { TAG } = value;
            const dataType = variants[TAG];
            if (!dataType)
                throw new Error(`Tag: invalid tag ${TAG.toString()}`);
            return value;
        },
    });
}
/**
 * Mapping between encoded values, string representations, and CoderTypes using a tag CoderType.
 * @param tagCoder - CoderType for the tag value.
 * @param variants - Object mapping string representations to [tag value, CoderType] pairs.
 *  * @returns CoderType representing the mapping.
 * @example
 * const cborValue: P.CoderType<CborValue> = P.mappedTag(P.bits(3), {
 *   uint: [0, cborUint], // An unsigned integer in the range 0..264-1 inclusive.
 *   negint: [1, cborNegint], // A negative integer in the range -264..-1 inclusive
 *   bytes: [2, P.lazy(() => cborLength(P.bytes, cborValue))], // A byte string.
 *   string: [3, P.lazy(() => cborLength(P.string, cborValue))], // A text string (utf8)
 *   array: [4, cborArrLength(P.lazy(() => cborValue))], // An array of data items
 *   map: [5, P.lazy(() => cborArrLength(P.tuple([cborValue, cborValue])))], // A map of pairs of data items
 *   tag: [6, P.tuple([cborUint, P.lazy(() => cborValue)] as const)], // A tagged data item ("tag") whose tag number
 *   simple: [7, cborSimple], // Floating-point numbers and simple values, as well as the "break" stop code
 * });
 */
export function mappedTag(tagCoder, variants) {
    if (!isCoder(tagCoder))
        throw new Error(`mappedTag: invalid tag value ${tag}`);
    if (!isPlainObject(variants))
        throw new Error(`mappedTag: variants should be plain object`);
    const mapValue = {};
    const tagValue = {};
    for (const key in variants) {
        const v = variants[key];
        mapValue[key] = v[0];
        tagValue[key] = v[1];
    }
    return tag(map(tagCoder, mapValue), tagValue);
}
/**
 * Bitset of boolean values with optional padding.
 * @param names - An array of string names for the bitset values.
 * @param pad - Whether to pad the bitset to a multiple of 8 bits.
 * @returns CoderType representing the bitset.
 * @template Names
 * @example
 * const myBitset = P.bitset(['flag1', 'flag2', 'flag3', 'flag4'], true);
 */
export function bitset(names, pad = false) {
    if (typeof pad !== 'boolean')
        throw new Error(`bitset/pad: expected boolean, got ${typeof pad}`);
    if (!Array.isArray(names))
        throw new Error('bitset/names: expected array');
    for (const name of names) {
        if (typeof name !== 'string')
            throw new Error('bitset/names: expected array of strings');
    }
    return wrap({
        encodeStream: (w, value) => {
            for (let i = 0; i < names.length; i++)
                w.bits(+value[names[i]], 1);
            if (pad && names.length % 8)
                w.bits(0, 8 - (names.length % 8));
        },
        decodeStream: (r) => {
            const out = {};
            for (let i = 0; i < names.length; i++)
                out[names[i]] = !!r.bits(1);
            if (pad && names.length % 8)
                r.bits(8 - (names.length % 8));
            return out;
        },
        validate: (value) => {
            if (!isPlainObject(value))
                throw new Error(`bitset: invalid value ${value}`);
            for (const v of Object.values(value)) {
                if (typeof v !== 'boolean')
                    throw new Error('expected boolean');
            }
            return value;
        },
    });
}
/** Padding function which always returns zero */
export const ZeroPad = (_) => 0;
function padLength(blockSize, len) {
    if (len % blockSize === 0)
        return 0;
    return blockSize - (len % blockSize);
}
/**
 * Pads a CoderType with a specified block size and padding function on the left side.
 * @param blockSize - Block size for padding (positive safe integer).
 * @param inner - Inner CoderType to pad.
 * @param padFn - Padding function to use. If not provided, zero padding is used.
 * @returns CoderType representing the padded value.
 * @example
 * // Pad a U32BE with a block size of 4 and zero padding
 * const paddedU32BE = P.padLeft(4, P.U32BE);
 *
 * // Pad a string with a block size of 16 and custom padding
 * const paddedString = P.padLeft(16, P.string(P.U8), (i) => i + 1);
 */
export function padLeft(blockSize, inner, padFn) {
    if (!isNum(blockSize) || blockSize <= 0)
        throw new Error(`padLeft: wrong blockSize=${blockSize}`);
    if (!isCoder(inner))
        throw new Error(`padLeft: invalid inner value ${inner}`);
    if (padFn !== undefined && typeof padFn !== 'function')
        throw new Error(`padLeft: wrong padFn=${typeof padFn}`);
    const _padFn = padFn || ZeroPad;
    if (!inner.size)
        throw new Error('padLeft cannot have dynamic size');
    return wrap({
        size: inner.size + padLength(blockSize, inner.size),
        encodeStream: (w, value) => {
            const padBytes = padLength(blockSize, inner.size);
            for (let i = 0; i < padBytes; i++)
                w.byte(_padFn(i));
            inner.encodeStream(w, value);
        },
        decodeStream: (r) => {
            r.bytes(padLength(blockSize, inner.size));
            return inner.decodeStream(r);
        },
    });
}
/**
 * Pads a CoderType with a specified block size and padding function on the right side.
 * @param blockSize - Block size for padding (positive safe integer).
 * @param inner - Inner CoderType to pad.
 * @param padFn - Padding function to use. If not provided, zero padding is used.
 * @returns CoderType representing the padded value.
 * @example
 * // Pad a U16BE with a block size of 2 and zero padding
 * const paddedU16BE = P.padRight(2, P.U16BE);
 *
 * // Pad a bytes with a block size of 8 and custom padding
 * const paddedBytes = P.padRight(8, P.bytes(null), (i) => i + 1);
 */
export function padRight(blockSize, inner, padFn) {
    if (!isCoder(inner))
        throw new Error(`padRight: invalid inner value ${inner}`);
    if (!isNum(blockSize) || blockSize <= 0)
        throw new Error(`padLeft: wrong blockSize=${blockSize}`);
    if (padFn !== undefined && typeof padFn !== 'function')
        throw new Error(`padRight: wrong padFn=${typeof padFn}`);
    const _padFn = padFn || ZeroPad;
    return wrap({
        size: inner.size ? inner.size + padLength(blockSize, inner.size) : undefined,
        encodeStream: (w, value) => {
            const _w = w;
            const pos = _w.pos;
            inner.encodeStream(w, value);
            const padBytes = padLength(blockSize, _w.pos - pos);
            for (let i = 0; i < padBytes; i++)
                w.byte(_padFn(i));
        },
        decodeStream: (r) => {
            const start = r.pos;
            const res = inner.decodeStream(r);
            r.bytes(padLength(blockSize, r.pos - start));
            return res;
        },
    });
}
1;
/**
 * Pointer to a value using a pointer CoderType and an inner CoderType.
 * Pointers are scoped, and the next pointer in the dereference chain is offset by the previous one.
 * By default (if no 'allowMultipleReads' in ReaderOpts is set) is safe, since
 * same region of memory cannot be read multiple times.
 * @param ptr - CoderType for the pointer value.
 * @param inner - CoderType for encoding/decoding the pointed value.
 * @param sized - Whether the pointer should have a fixed size.
 * @returns CoderType representing the pointer to the value.
 * @example
 * const pointerToU8 = P.pointer(P.U16BE, P.U8); // Pointer to a single U8 value
 */
export function pointer(ptr, inner, sized = false) {
    if (!isCoder(ptr))
        throw new Error(`pointer: invalid ptr value ${ptr}`);
    if (!isCoder(inner))
        throw new Error(`pointer: invalid inner value ${inner}`);
    if (typeof sized !== 'boolean')
        throw new Error(`pointer/sized: expected boolean, got ${typeof sized}`);
    if (!ptr.size)
        throw new Error('unsized pointer');
    return wrap({
        size: sized ? ptr.size : undefined,
        encodeStream: (w, value) => {
            const _w = w;
            const start = _w.pos;
            ptr.encodeStream(w, 0);
            _w.ptrs.push({ pos: start, ptr, buffer: inner.encode(value) });
        },
        decodeStream: (r) => {
            const ptrVal = ptr.decodeStream(r);
            r._enablePointers();
            return inner.decodeStream(r.offsetReader(ptrVal));
        },
    });
}
// Internal methods for test purposes only
export const _TEST = { _bitset: Bitset, _Reader, _Writer, Path };
//# sourceMappingURL=index.js.map