import { hex as baseHex, utf8, type Coder as BaseCoder } from '@scure/base';

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
export const EMPTY: Uint8Array = /* @__PURE__ */ new Uint8Array();
/** Shortcut to one-element (element is 0) byte array */
export const NULL: Uint8Array = /* @__PURE__ */ new Uint8Array([0]);

/** Checks if two Uint8Arrays are equal. Not constant-time. */
function equalBytes(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) if (a[i] !== b[i]) return false;
  return true;
}
/** Checks if the given value is a Uint8Array. */
function isBytes(a: unknown): a is Bytes {
  return a instanceof Uint8Array || (ArrayBuffer.isView(a) && a.constructor.name === 'Uint8Array');
}

/**
 * Concatenates multiple Uint8Arrays.
 * Engines limit functions to 65K+ arguments.
 * @param arrays Array of Uint8Array elements
 * @returns Concatenated Uint8Array
 */
function concatBytes(...arrays: Uint8Array[]): Uint8Array {
  let sum = 0;
  for (let i = 0; i < arrays.length; i++) {
    const a = arrays[i];
    if (!isBytes(a)) throw new Error('Uint8Array expected');
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
const createView = (arr: Uint8Array) => new DataView(arr.buffer, arr.byteOffset, arr.byteLength);

/**
 * Checks if the provided value is a plain object, not created from any class or special constructor.
 * Array, Uint8Array and others are not plain objects.
 * @param obj - The value to be checked.
 */
function isPlainObject(obj: any): boolean {
  return Object.prototype.toString.call(obj) === '[object Object]';
}

function isNum(num: unknown): num is number {
  return Number.isSafeInteger(num);
}

export const utils: {
  equalBytes: typeof equalBytes;
  isBytes: typeof isBytes;
  isCoder: typeof isCoder;
  checkBounds: typeof checkBounds;
  concatBytes: typeof concatBytes;
  createView: (arr: Uint8Array) => DataView;
  isPlainObject: typeof isPlainObject;
} = {
  equalBytes,
  isBytes,
  isCoder,
  checkBounds,
  concatBytes,
  createView,
  isPlainObject,
};

// Types
export type Bytes = Uint8Array;
export type Option<T> = T | undefined;
/**
 * Coder encodes and decodes between two types.
 * @property {(from: F) => T} encode - Encodes (converts) F to T
 * @property {(to: T) => F} decode - Decodes (converts) T to F
 */
export interface Coder<F, T> {
  encode(from: F): T;
  decode(to: T): F;
}
/**
 * BytesCoder converts value between a type and a byte array
 * @property {number} [size] - Size hint for the element.
 * @property {(data: T) => Bytes} encode - Encodes a value of type T to a byte array
 * @property {(data: Bytes, opts?: ReaderOpts) => T} decode - Decodes a byte array to a value of type T
 */
export interface BytesCoder<T> extends Coder<T, Bytes> {
  size?: number; // Size hint element
  encode: (data: T) => Bytes;
  decode: (data: Bytes, opts?: ReaderOpts) => T;
}
/**
 * BytesCoderStream converts value between a type and a byte array, using streams.
 * @property {number} [size] - Size hint for the element.
 * @property {(w: Writer, value: T) => void} encodeStream - Encodes a value of type T to a byte array using a Writer stream.
 * @property {(r: Reader) => T} decodeStream - Decodes a byte array to a value of type T using a Reader stream.
 */
export interface BytesCoderStream<T> {
  size?: number;
  encodeStream: (w: Writer, value: T) => void;
  decodeStream: (r: Reader) => T;
}
export type CoderType<T> = BytesCoderStream<T> & BytesCoder<T>;
export type Sized<T> = CoderType<T> & { size: number };
export type UnwrapCoder<T> = T extends CoderType<infer U> ? U : T;
/**
 * Validation function. Should return value after validation.
 * Can be used to narrow types
 */
export type Validate<T> = (elm: T) => T;

export type Length = CoderType<number> | CoderType<bigint> | number | Bytes | string | null;

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
const lengthCoder = (len: Length) => {
  if (len !== null && typeof len !== 'string' && !isCoder(len) && !isBytes(len) && !isNum(len)) {
    throw new Error(
      `lengthCoder: expected null | number | Uint8Array | CoderType, got ${len} (${typeof len})`
    );
  }
  return {
    encodeStream(w: Writer, value: number | null) {
      if (len === null) return;
      if (isCoder(len)) return len.encodeStream(w, value);
      let byteLen;
      if (typeof len === 'number') byteLen = len;
      else if (typeof len === 'string') byteLen = Path.resolve((w as _Writer).stack, len);
      if (typeof byteLen === 'bigint') byteLen = Number(byteLen);
      if (byteLen === undefined || byteLen !== value)
        throw w.err(`Wrong length: ${byteLen} len=${len} exp=${value} (${typeof value})`);
    },
    decodeStream(r: Reader) {
      let byteLen;
      if (isCoder(len)) byteLen = Number(len.decodeStream(r));
      else if (typeof len === 'number') byteLen = len;
      else if (typeof len === 'string') byteLen = Path.resolve((r as _Reader).stack, len);
      if (typeof byteLen === 'bigint') byteLen = Number(byteLen);
      if (typeof byteLen !== 'number') throw r.err(`Wrong length: ${byteLen}`);
      return byteLen;
    },
  };
};

type ArrLike<T> = Array<T> | ReadonlyArray<T>;
// prettier-ignore
export type TypedArray =
  | Uint8Array  | Int8Array | Uint8ClampedArray
  | Uint16Array | Int16Array
  | Uint32Array | Int32Array;

/** Writable version of a type, where readonly properties are made writable. */
export type Writable<T> = T extends {}
  ? T extends TypedArray
    ? T
    : {
        -readonly [P in keyof T]: Writable<T[P]>;
      }
  : T;
export type Values<T> = T[keyof T];
export type NonUndefinedKey<T, K extends keyof T> = T[K] extends undefined ? never : K;
export type NullableKey<T, K extends keyof T> = T[K] extends NonNullable<T[K]> ? never : K;
// Opt: value !== undefined, but value === T|undefined
export type OptKey<T, K extends keyof T> = NullableKey<T, K> & NonUndefinedKey<T, K>;
export type ReqKey<T, K extends keyof T> = T[K] extends NonNullable<T[K]> ? K : never;

export type OptKeys<T> = Pick<T, { [K in keyof T]: OptKey<T, K> }[keyof T]>;
export type ReqKeys<T> = Pick<T, { [K in keyof T]: ReqKey<T, K> }[keyof T]>;
export type StructInput<T extends Record<string, any>> = { [P in keyof ReqKeys<T>]: T[P] } & {
  [P in keyof OptKeys<T>]?: T[P];
};
export type StructRecord<T extends Record<string, any>> = {
  [P in keyof T]: CoderType<T[P]>;
};

export type StructOut = Record<string, any>;
/** Padding function that takes an index and returns a padding value. */
export type PadFn = (i: number) => number;

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
  len: (len: number) => Math.ceil(len / 32),
  create: (len: number) => new Uint32Array(Bitset.len(len)),
  clean: (bs: Uint32Array) => bs.fill(0),
  debug: (bs: Uint32Array) => Array.from(bs).map((i) => (i >>> 0).toString(2).padStart(32, '0')),
  checkLen: (bs: Uint32Array, len: number) => {
    if (Bitset.len(len) === bs.length) return;
    throw new Error(`wrong length=${bs.length}. Expected: ${Bitset.len(len)}`);
  },
  chunkLen: (bsLen: number, pos: number, len: number) => {
    if (pos < 0) throw new Error(`wrong pos=${pos}`);
    if (pos + len > bsLen) throw new Error(`wrong range=${pos}/${len} of ${bsLen}`);
  },
  set: (bs: Uint32Array, chunk: number, value: number, allowRewrite = true) => {
    if (!allowRewrite && (bs[chunk] & value) !== 0) return false;
    bs[chunk] |= value;
    return true;
  },
  pos: (pos: number, i: number) => ({
    chunk: Math.floor((pos + i) / 32),
    mask: 1 << (32 - ((pos + i) % 32) - 1),
  }),
  indices: (bs: Uint32Array, len: number, invert = false) => {
    Bitset.checkLen(bs, len);
    const { FULL_MASK, BITS } = Bitset;
    const left = BITS - (len % BITS);
    const lastMask = left ? (FULL_MASK >>> left) << left : FULL_MASK;
    const res = [];
    for (let i = 0; i < bs.length; i++) {
      let c = bs[i];
      if (invert) c = ~c; // allows to gen unset elements
      // apply mask to last element, so we won't iterate non-existent items
      if (i === bs.length - 1) c &= lastMask;
      if (c === 0) continue; // fast-path
      for (let j = 0; j < BITS; j++) {
        const m = 1 << (BITS - j - 1);
        if (c & m) res.push(i * BITS + j);
      }
    }
    return res;
  },
  range: (arr: number[]) => {
    const res = [];
    let cur;
    for (const i of arr) {
      if (cur === undefined || i !== cur.pos + cur.length) res.push((cur = { pos: i, length: 1 }));
      else cur.length += 1;
    }
    return res;
  },
  rangeDebug: (bs: Uint32Array, len: number, invert = false) =>
    `[${Bitset.range(Bitset.indices(bs, len, invert))
      .map((i) => `(${i.pos}/${i.length})`)
      .join(', ')}]`,
  setRange: (bs: Uint32Array, bsLen: number, pos: number, len: number, allowRewrite = true) => {
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
      return Bitset.set(
        bs,
        first,
        (FULL_MASK >>> (BITS - len)) << (BITS - len - pos),
        allowRewrite
      );
    if (first !== undefined) {
      if (!Bitset.set(bs, first, FULL_MASK >>> pos % BITS, allowRewrite)) return false; // first chunk
    }
    // middle chunks
    const start = first !== undefined ? first + 1 : pos / BITS;
    const end = last !== undefined ? last : lastPos / BITS;
    for (let i = start; i < end; i++) if (!Bitset.set(bs, i, FULL_MASK, allowRewrite)) return false;
    if (last !== undefined && first !== last)
      if (!Bitset.set(bs, last, FULL_MASK << (BITS - (lastPos % BITS)), allowRewrite)) return false; // last chunk
    return true;
  },
};

/** Path related utils (internal) */
type Path = { obj: StructOut; field?: string };
type PathStack = Path[];
export type _PathObjFn = (cb: (field: string, fieldFn: Function) => void) => void;
const Path = {
  /**
   * Internal method for handling stack of paths (debug, errors, dynamic fields via path)
   * This is looks ugly (callback), but allows us to force stack cleaning by construction (.pop always after function).
   * Also, this makes impossible:
   * - pushing field when stack is empty
   * - pushing field inside of field (real bug)
   * NOTE: we don't want to do '.pop' on error!
   */
  pushObj: (stack: PathStack, obj: StructOut, objFn: _PathObjFn): void => {
    const last: Path = { obj };
    stack.push(last);
    objFn((field: string, fieldFn: Function) => {
      last.field = field;
      fieldFn();
      last.field = undefined;
    });
    stack.pop();
  },
  path: (stack: PathStack): string => {
    const res = [];
    for (const i of stack) if (i.field !== undefined) res.push(i.field);
    return res.join('/');
  },
  err: (name: string, stack: PathStack, msg: string | Error): Error => {
    const err = new Error(
      `${name}(${Path.path(stack)}): ${typeof msg === 'string' ? msg : msg.message}`
    );
    if (msg instanceof Error && msg.stack) err.stack = msg.stack;
    return err;
  },
  resolve: (stack: PathStack, path: string): StructOut | undefined => {
    const parts = path.split('/');
    const objPath = stack.map((i) => i.obj);
    let i = 0;
    for (; i < parts.length; i++) {
      if (parts[i] === '..') objPath.pop();
      else break;
    }
    let cur = objPath.pop();
    for (; i < parts.length; i++) {
      if (!cur || cur[parts[i]] === undefined) return undefined;
      cur = cur[parts[i]];
    }
    return cur;
  },
};

/**
 * Options for the Reader class.
 * @property {boolean} [allowUnreadBytes: false] - If there are remaining unparsed bytes, the decoding is probably wrong.
 * @property {boolean} [allowMultipleReads: false] - The check enforces parser termination. If pointers can read the same region of memory multiple times, you can cause combinatorial explosion by creating an array of pointers to the same address and cause DoS.
 */
export type ReaderOpts = {
  allowUnreadBytes?: boolean;
  allowMultipleReads?: boolean;
};
// These are safe API for external usage
export type Reader = {
  // Utils
  /** Current position in the buffer. */
  readonly pos: number;
  /** Number of bytes left in the buffer. */
  readonly leftBytes: number;
  /** Total number of bytes in the buffer. */
  readonly totalBytes: number;
  /** Checks if the end of the buffer has been reached. */
  isEnd(): boolean;
  /**
   * Creates an error with the given message. Adds information about current field path.
   * If Error object provided, saves original stack trace.
   * @param msg - The error message or an Error object.
   * @returns The created Error object.
   */
  err(msg: string | Error): Error;
  /**
   * Reads a specified number of bytes from the buffer.
   *
   * WARNING: Uint8Array is subarray of original buffer. Do not modify.
   * @param n - The number of bytes to read.
   * @param peek - If `true`, the bytes are read without advancing the position.
   * @returns The read bytes as a Uint8Array.
   */
  bytes(n: number, peek?: boolean): Uint8Array;
  /**
   * Reads a single byte from the buffer.
   * @param peek - If `true`, the byte is read without advancing the position.
   * @returns The read byte as a number.
   */
  byte(peek?: boolean): number;
  /**
   * Reads a specified number of bits from the buffer.
   * @param bits - The number of bits to read.
   * @returns The read bits as a number.
   */
  bits(bits: number): number;
  /**
   * Finds the first occurrence of a needle in the buffer.
   * @param needle - The needle to search for.
   * @param pos - The starting position for the search.
   * @returns The position of the first occurrence of the needle, or `undefined` if not found.
   */
  find(needle: Bytes, pos?: number): number | undefined;
  /**
   * Creates a new Reader instance at the specified offset.
   * Complex and unsafe API: currently only used in eth ABI parsing of pointers.
   * Required to break pointer boundaries inside arrays for complex structure.
   * Please use only if absolutely necessary!
   * @param n - The offset to create the new Reader at.
   * @returns A new Reader instance at the specified offset.
   */
  offsetReader(n: number): Reader;
};

export type Writer = {
  /**
   * Creates an error with the given message. Adds information about current field path.
   * If Error object provided, saves original stack trace.
   * @param msg - The error message or an Error object.
   * @returns The created Error object.
   */
  err(msg: string | Error): Error;
  /**
   * Writes a byte array to the buffer.
   * @param b - The byte array to write.
   */
  bytes(b: Bytes): void;
  /**
   * Writes a single byte to the buffer.
   * @param b - The byte to write.
   */
  byte(b: number): void;
  /**
   * Writes a specified number of bits to the buffer.
   * @param value - The value to write.
   * @param bits - The number of bits to write.
   */
  bits(value: number, bits: number): void;
};

/**
 * Internal structure. Reader class for reading from a byte array.
 * `stack` is internal: for debugger and logging
 * @class Reader
 */
class _Reader implements Reader {
  pos = 0;
  readonly data: Bytes;
  readonly opts: ReaderOpts;
  readonly stack: PathStack;
  private parent: _Reader | undefined;
  private parentOffset: number;
  private bitBuf = 0;
  private bitPos = 0;
  private bs: Uint32Array | undefined; // bitset
  private view: DataView;
  constructor(
    data: Bytes,
    opts: ReaderOpts = {},
    stack: PathStack = [],
    parent: _Reader | undefined = undefined,
    parentOffset: number = 0
  ) {
    this.data = data;
    this.opts = opts;
    this.stack = stack;
    this.parent = parent;
    this.parentOffset = parentOffset;
    this.view = createView(data);
  }
  /** Internal method for pointers. */
  _enablePointers(): void {
    if (this.parent) return this.parent._enablePointers();
    if (this.bs) return;
    this.bs = Bitset.create(this.data.length);
    Bitset.setRange(this.bs, this.data.length, 0, this.pos, this.opts.allowMultipleReads);
  }
  private markBytesBS(pos: number, len: number): boolean {
    if (this.parent) return this.parent.markBytesBS(this.parentOffset + pos, len);
    if (!len) return true;
    if (!this.bs) return true;
    return Bitset.setRange(this.bs, this.data.length, pos, len, false);
  }
  private markBytes(len: number): boolean {
    const pos = this.pos;
    this.pos += len;
    const res = this.markBytesBS(pos, len);
    if (!this.opts.allowMultipleReads && !res)
      throw this.err(`multiple read pos=${this.pos} len=${len}`);
    return res;
  }

  pushObj(obj: StructOut, objFn: _PathObjFn): void {
    return Path.pushObj(this.stack, obj, objFn);
  }
  readView(n: number, fn: (view: DataView, pos: number) => number): number {
    if (!Number.isFinite(n)) throw this.err(`readView: wrong length=${n}`);
    if (this.pos + n > this.data.length) throw this.err('readView: Unexpected end of buffer');
    const res = fn(this.view, this.pos);
    this.markBytes(n);
    return res;
  }
  // read bytes by absolute offset
  absBytes(n: number): Uint8Array {
    if (n > this.data.length) throw new Error('Unexpected end of buffer');
    return this.data.subarray(n);
  }
  finish(): void {
    if (this.opts.allowUnreadBytes) return;
    if (this.bitPos) {
      throw this.err(
        `${this.bitPos} bits left after unpack: ${baseHex.encode(this.data.slice(this.pos))}`
      );
    }
    if (this.bs && !this.parent) {
      const notRead = Bitset.indices(this.bs, this.data.length, true);
      if (notRead.length) {
        const formatted = Bitset.range(notRead)
          .map(
            ({ pos, length }) =>
              `(${pos}/${length})[${baseHex.encode(this.data.subarray(pos, pos + length))}]`
          )
          .join(', ');
        throw this.err(`unread byte ranges: ${formatted} (total=${this.data.length})`);
      } else return; // all bytes read, everything is ok
    }
    // Default: no pointers enabled
    if (!this.isEnd()) {
      throw this.err(
        `${this.leftBytes} bytes ${this.bitPos} bits left after unpack: ${baseHex.encode(
          this.data.slice(this.pos)
        )}`
      );
    }
  }
  // User methods
  err(msg: string | Error): Error {
    return Path.err('Reader', this.stack, msg);
  }
  offsetReader(n: number): _Reader {
    if (n > this.data.length) throw this.err('offsetReader: Unexpected end of buffer');
    return new _Reader(this.absBytes(n), this.opts, this.stack, this, n);
  }
  bytes(n: number, peek = false): Uint8Array {
    if (this.bitPos) throw this.err('readBytes: bitPos not empty');
    if (!Number.isFinite(n)) throw this.err(`readBytes: wrong length=${n}`);
    if (this.pos + n > this.data.length) throw this.err('readBytes: Unexpected end of buffer');
    const slice = this.data.subarray(this.pos, this.pos + n);
    if (!peek) this.markBytes(n);
    return slice;
  }
  byte(peek = false): number {
    if (this.bitPos) throw this.err('readByte: bitPos not empty');
    if (this.pos + 1 > this.data.length) throw this.err('readBytes: Unexpected end of buffer');
    const data = this.data[this.pos];
    if (!peek) this.markBytes(1);
    return data;
  }
  get leftBytes(): number {
    return this.data.length - this.pos;
  }
  get totalBytes(): number {
    return this.data.length;
  }
  isEnd(): boolean {
    return this.pos >= this.data.length && !this.bitPos;
  }
  // bits are read in BE mode (left to right): (0b1000_0000).readBits(1) == 1
  bits(bits: number): number {
    if (bits > 32) throw this.err('BitReader: cannot read more than 32 bits in single call');
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
  find(needle: Bytes, pos: number = this.pos): number | undefined {
    if (!isBytes(needle)) throw this.err(`find: needle is not bytes! ${needle}`);
    if (this.bitPos) throw this.err('findByte: bitPos not empty');
    if (!needle.length) throw this.err(`find: needle is empty`);
    // indexOf should be faster than full equalBytes check
    for (let idx = pos; (idx = this.data.indexOf(needle[0], idx)) !== -1; idx++) {
      if (idx === -1) return;
      const leftBytes = this.data.length - idx;
      if (leftBytes < needle.length) return;
      if (equalBytes(needle, this.data.subarray(idx, idx + needle.length))) return idx;
    }
    return;
  }
}

/**
 * Internal structure. Writer class for writing to a byte array.
 * The `stack` argument of constructor is internal, for debugging and logs.
 * @class Writer
 */
class _Writer implements Writer {
  pos: number = 0;
  readonly stack: PathStack;
  // We could have a single buffer here and re-alloc it with
  // x1.5-2 size each time it full, but it will be slower:
  // basic/encode bench: 395ns -> 560ns
  private buffers: Bytes[] = [];
  ptrs: { pos: number; ptr: CoderType<number>; buffer: Bytes }[] = [];
  private bitBuf = 0;
  private bitPos = 0;
  private viewBuf = new Uint8Array(8);
  private view: DataView;
  private finished = false;
  constructor(stack: PathStack = []) {
    this.stack = stack;
    this.view = createView(this.viewBuf);
  }
  pushObj(obj: StructOut, objFn: _PathObjFn): void {
    return Path.pushObj(this.stack, obj, objFn);
  }
  writeView(len: number, fn: (view: DataView) => void): void {
    if (this.finished) throw this.err('buffer: finished');
    if (!isNum(len) || len > 8) throw new Error(`wrong writeView length=${len}`);
    fn(this.view);
    this.bytes(this.viewBuf.slice(0, len));
    this.viewBuf.fill(0);
  }
  // User methods
  err(msg: string | Error): Error {
    if (this.finished) throw this.err('buffer: finished');
    return Path.err('Reader', this.stack, msg);
  }
  bytes(b: Bytes): void {
    if (this.finished) throw this.err('buffer: finished');
    if (this.bitPos) throw this.err('writeBytes: ends with non-empty bit buffer');
    this.buffers.push(b);
    this.pos += b.length;
  }
  byte(b: number): void {
    if (this.finished) throw this.err('buffer: finished');
    if (this.bitPos) throw this.err('writeByte: ends with non-empty bit buffer');
    this.buffers.push(new Uint8Array([b]));
    this.pos++;
  }
  finish(clean = true): Bytes {
    if (this.finished) throw this.err('buffer: finished');
    if (this.bitPos) throw this.err('buffer: ends with non-empty bit buffer');
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
      for (const p of this.ptrs) p.buffer.fill(0);
      this.ptrs = [];
      this.finished = true;
      this.bitBuf = 0;
    }
    return buf;
  }
  bits(value: number, bits: number): void {
    if (bits > 32) throw this.err('writeBits: cannot write more than 32 bits in single call');
    if (value >= 2 ** bits) throw this.err(`writeBits: value (${value}) >= 2**bits (${bits})`);
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
const swapEndianness = (b: Bytes): Bytes => Uint8Array.from(b).reverse();
/** Internal function for checking bit bounds of bigint in signed/unsinged form */
function checkBounds(value: bigint, bits: bigint, signed: boolean): void {
  if (signed) {
    // [-(2**(32-1)), 2**(32-1)-1]
    const signBit = 2n ** (bits - 1n);
    if (value < -signBit || value >= signBit)
      throw new Error(`value out of signed bounds. Expected ${-signBit} <= ${value} < ${signBit}`);
  } else {
    // [0, 2**32-1]
    if (0n > value || value >= 2n ** bits)
      throw new Error(`value out of unsigned bounds. Expected 0 <= ${value} < ${2n ** bits}`);
  }
}

function _wrap<T>(inner: BytesCoderStream<T>): CoderType<T> {
  return {
    // NOTE: we cannot export validate here, since it is likely mistake.
    encodeStream: inner.encodeStream,
    decodeStream: inner.decodeStream,
    size: inner.size,
    encode: (value: T): Bytes => {
      const w = new _Writer();
      inner.encodeStream(w, value);
      return w.finish();
    },
    decode: (data: Bytes, opts: ReaderOpts = {}): T => {
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
export function validate<T>(inner: CoderType<T>, fn: Validate<T>): CoderType<T> {
  if (!isCoder(inner)) throw new Error(`validate: invalid inner value ${inner}`);
  if (typeof fn !== 'function') throw new Error('validate: fn should be function');
  return _wrap({
    size: inner.size,
    encodeStream: (w: Writer, value: T) => {
      let res;
      try {
        res = fn(value);
      } catch (e) {
        throw w.err(e as Error);
      }
      inner.encodeStream(w, res);
    },
    decodeStream: (r: Reader): T => {
      const res = inner.decodeStream(r);
      try {
        return fn(res);
      } catch (e) {
        throw r.err(e as Error);
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
export const wrap = <T>(inner: BytesCoderStream<T> & { validate?: Validate<T> }): CoderType<T> => {
  const res = _wrap(inner);
  return inner.validate ? validate(res, inner.validate) : res;
};

const isBaseCoder = (elm: any) =>
  isPlainObject(elm) && typeof elm.decode === 'function' && typeof elm.encode === 'function';

/**
 * Checks if the given value is a CoderType.
 * @param elm - The value to check.
 * @returns True if the value is a CoderType, false otherwise.
 */
export function isCoder<T>(elm: any): elm is CoderType<T> {
  return (
    isPlainObject(elm) &&
    isBaseCoder(elm) &&
    typeof elm.encodeStream === 'function' &&
    typeof elm.decodeStream === 'function' &&
    (elm.size === undefined || isNum(elm.size))
  );
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
function dict<T>(): BaseCoder<[string, T][], Record<string, T>> {
  return {
    encode: (from: [string, T][]): Record<string, T> => {
      if (!Array.isArray(from)) throw new Error('array expected');
      const to: Record<string, T> = {};
      for (const item of from) {
        if (!Array.isArray(item) || item.length !== 2)
          throw new Error(`array of two elements expected`);
        const name = item[0];
        const value = item[1];
        if (to[name] !== undefined) throw new Error(`key(${name}) appears twice in struct`);
        to[name] = value;
      }
      return to;
    },
    decode: (to: Record<string, T>): [string, T][] => {
      if (!isPlainObject(to)) throw new Error(`expected plain object, got ${to}`);
      return Object.entries(to);
    },
  };
}
/**
 * Safely converts bigint to number.
 * Sometimes pointers / tags use u64 or other big numbers which cannot be represented by number,
 * but we still can use them since real value will be smaller than u32
 */
const numberBigint: BaseCoder<bigint, number> = {
  encode: (from: bigint): number => {
    if (typeof from !== 'bigint') throw new Error(`expected bigint, got ${typeof from}`);
    if (from > BigInt(Number.MAX_SAFE_INTEGER))
      throw new Error(`element bigger than MAX_SAFE_INTEGER=${from}`);
    return Number(from);
  },
  decode: (to: number): bigint => {
    if (!isNum(to)) throw new Error('element is not a safe integer');
    return BigInt(to);
  },
};
// TODO: replace map with this?
type Enum = { [k: string]: number | string } & { [k: number]: string };
// Doesn't return numeric keys, so it's fine
type EnumKeys<T extends Enum> = keyof T;
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
function tsEnum<T extends Enum>(e: T): BaseCoder<number, EnumKeys<T>> {
  if (!isPlainObject(e)) throw new Error('plain object expected');
  return {
    encode: (from: number): string => {
      if (!isNum(from) || !(from in e)) throw new Error(`wrong value ${from}`);
      return e[from];
    },
    decode: (to: string): number => {
      if (typeof to !== 'string') throw new Error(`wrong value ${typeof to}`);
      return e[to] as number;
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
function decimal(precision: number, round = false): Coder<bigint, string> {
  if (!isNum(precision)) throw new Error(`decimal/precision: wrong value ${precision}`);
  if (typeof round !== 'boolean')
    throw new Error(`decimal/round: expected boolean, got ${typeof round}`);
  const decimalMask = 10n ** BigInt(precision);
  return {
    encode: (from: bigint): string => {
      if (typeof from !== 'bigint') throw new Error(`expected bigint, got ${typeof from}`);
      let s = (from < 0n ? -from : from).toString(10);
      let sep = s.length - precision;
      if (sep < 0) {
        s = s.padStart(s.length - sep, '0');
        sep = 0;
      }
      let i = s.length - 1;
      for (; i >= sep && s[i] === '0'; i--);
      let int = s.slice(0, sep);
      let frac = s.slice(sep, i + 1);
      if (!int) int = '0';
      if (from < 0n) int = '-' + int;
      if (!frac) return int;
      return `${int}.${frac}`;
    },
    decode: (to: string): bigint => {
      if (typeof to !== 'string') throw new Error(`expected string, got ${typeof to}`);
      if (to === '-0') throw new Error(`negative zero is not allowed`);
      let neg = false;
      if (to.startsWith('-')) {
        neg = true;
        to = to.slice(1);
      }
      if (!/^(0|[1-9]\d*)(\.\d+)?$/.test(to)) throw new Error(`wrong string value=${to}`);
      let sep = to.indexOf('.');
      sep = sep === -1 ? to.length : sep;
      // split by separator and strip trailing zeros from fraction. always returns [string, string] (.split doesn't).
      const intS = to.slice(0, sep);
      const fracS = to.slice(sep + 1).replace(/0+$/, '');
      const int = BigInt(intS) * decimalMask;
      if (!round && fracS.length > precision) {
        throw new Error(
          `fractional part cannot be represented with this precision (num=${to}, prec=${precision})`
        );
      }
      const fracLen = Math.min(fracS.length, precision);
      const frac = BigInt(fracS.slice(0, fracLen)) * 10n ** BigInt(precision - fracLen);
      const value = int + frac;
      return neg ? -value : value;
    },
  };
}

// TODO: export from @scure/base?
type BaseInput<F> = F extends BaseCoder<infer T, any> ? T : never;
type BaseOutput<F> = F extends BaseCoder<any, infer T> ? T : never;

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
function match<
  L extends BaseCoder<unknown | undefined, unknown | undefined>[],
  I = { [K in keyof L]: NonNullable<BaseInput<L[K]>> }[number],
  O = { [K in keyof L]: NonNullable<BaseOutput<L[K]>> }[number],
>(lst: L): BaseCoder<I, O> {
  if (!Array.isArray(lst)) throw new Error(`expected array, got ${typeof lst}`);
  for (const i of lst) if (!isBaseCoder(i)) throw new Error(`wrong base coder ${i}`);
  return {
    encode: (from: I): O => {
      for (const c of lst) {
        const elm = c.encode(from);
        if (elm !== undefined) return elm as O;
      }
      throw new Error(`match/encode: cannot find match in ${from}`);
    },
    decode: (to: O): I => {
      for (const c of lst) {
        const elm = c.decode(to);
        if (elm !== undefined) return elm as I;
      }
      throw new Error(`match/decode: cannot find match in ${to}`);
    },
  };
}
/** Reverses direction of coder */
const reverse = <F, T>(coder: Coder<F, T>): Coder<T, F> => {
  if (!isBaseCoder(coder)) throw new Error('BaseCoder expected');
  return { encode: coder.decode, decode: coder.encode };
};

export const coders: {
  dict: typeof dict;
  numberBigint: BaseCoder<bigint, number>;
  tsEnum: typeof tsEnum;
  decimal: typeof decimal;
  match: typeof match;
  reverse: <F, T>(coder: Coder<F, T>) => Coder<T, F>;
} = { dict, numberBigint, tsEnum, decimal, match, reverse };

/**
 * CoderType for parsing individual bits.
 * NOTE: Structure should parse whole amount of bytes before it can start parsing byte-level elements.
 * @param len - Number of bits to parse.
 * @returns CoderType representing the parsed bits.
 * @example
 * const s = P.struct({ magic: P.bits(1), version: P.bits(1), tag: P.bits(4), len: P.bits(2) });
 */
export const bits = (len: number): CoderType<number> => {
  if (!isNum(len)) throw new Error(`bits: wrong length ${len} (${typeof len})`);
  return wrap({
    encodeStream: (w: Writer, value: number) => w.bits(value, len),
    decodeStream: (r: Reader): number => r.bits(len),
    validate: (value) => {
      if (!isNum(value)) throw new Error(`bits: wrong value ${value}`);
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
export const bigint = (
  size: number,
  le = false,
  signed = false,
  sized = true
): CoderType<bigint> => {
  if (!isNum(size)) throw new Error(`bigint/size: wrong value ${size}`);
  if (typeof le !== 'boolean') throw new Error(`bigint/le: expected boolean, got ${typeof le}`);
  if (typeof signed !== 'boolean')
    throw new Error(`bigint/signed: expected boolean, got ${typeof signed}`);
  if (typeof sized !== 'boolean')
    throw new Error(`bigint/sized: expected boolean, got ${typeof sized}`);
  const bLen = BigInt(size);
  const signBit = 2n ** (8n * bLen - 1n);
  return wrap({
    size: sized ? size : undefined,
    encodeStream: (w: Writer, value: bigint) => {
      if (signed && value < 0) value = value | signBit;
      const b = [];
      for (let i = 0; i < size; i++) {
        b.push(Number(value & 255n));
        value >>= 8n;
      }
      let res = new Uint8Array(b).reverse();
      if (!sized) {
        let pos = 0;
        for (pos = 0; pos < res.length; pos++) if (res[pos] !== 0) break;
        res = res.subarray(pos); // remove leading zeros
      }
      w.bytes(le ? res.reverse() : res);
    },
    decodeStream: (r: Reader): bigint => {
      // TODO: for le we can read until first zero?
      const value = r.bytes(sized ? size : Math.min(size, r.leftBytes));
      const b = le ? value : swapEndianness(value);
      let res = 0n;
      for (let i = 0; i < b.length; i++) res |= BigInt(b[i]) << (8n * BigInt(i));
      if (signed && res & signBit) res = (res ^ signBit) - signBit;
      return res;
    },
    validate: (value) => {
      if (typeof value !== 'bigint') throw new Error(`bigint: invalid value: ${value}`);
      checkBounds(value, 8n * bLen, !!signed);
      return value;
    },
  });
};
/** Unsigned 256-bit little-endian integer CoderType. */
export const U256LE: CoderType<bigint> = /* @__PURE__ */ bigint(32, true);
/** Unsigned 256-bit big-endian integer CoderType. */
export const U256BE: CoderType<bigint> = /* @__PURE__ */ bigint(32, false);
/** Signed 256-bit little-endian integer CoderType. */
export const I256LE: CoderType<bigint> = /* @__PURE__ */ bigint(32, true, true);
/** Signed 256-bit big-endian integer CoderType. */
export const I256BE: CoderType<bigint> = /* @__PURE__ */ bigint(32, false, true);
/** Unsigned 128-bit little-endian integer CoderType. */
export const U128LE: CoderType<bigint> = /* @__PURE__ */ bigint(16, true);
/** Unsigned 128-bit big-endian integer CoderType. */
export const U128BE: CoderType<bigint> = /* @__PURE__ */ bigint(16, false);
/** Signed 128-bit little-endian integer CoderType. */
export const I128LE: CoderType<bigint> = /* @__PURE__ */ bigint(16, true, true);
/** Signed 128-bit big-endian integer CoderType. */
export const I128BE: CoderType<bigint> = /* @__PURE__ */ bigint(16, false, true);
/** Unsigned 64-bit little-endian integer CoderType. */
export const U64LE: CoderType<bigint> = /* @__PURE__ */ bigint(8, true);
/** Unsigned 64-bit big-endian integer CoderType. */
export const U64BE: CoderType<bigint> = /* @__PURE__ */ bigint(8, false);
/** Signed 64-bit little-endian integer CoderType. */
export const I64LE: CoderType<bigint> = /* @__PURE__ */ bigint(8, true, true);
/** Signed 64-bit big-endian integer CoderType. */
export const I64BE: CoderType<bigint> = /* @__PURE__ */ bigint(8, false, true);

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
export const int = (size: number, le = false, signed = false, sized = true): CoderType<number> => {
  if (!isNum(size)) throw new Error(`int/size: wrong value ${size}`);
  if (typeof le !== 'boolean') throw new Error(`int/le: expected boolean, got ${typeof le}`);
  if (typeof signed !== 'boolean')
    throw new Error(`int/signed: expected boolean, got ${typeof signed}`);
  if (typeof sized !== 'boolean')
    throw new Error(`int/sized: expected boolean, got ${typeof sized}`);
  if (size > 6) throw new Error('int supports size up to 6 bytes (48 bits): use bigints instead');
  return apply(bigint(size, le, signed, sized), coders.numberBigint);
};

type ViewCoder = {
  read: (view: DataView, pos: number) => number;
  write: (view: DataView, value: number) => void;
  validate?: (value: number) => void;
};

const view = (len: number, opts: ViewCoder) =>
  wrap({
    size: len,
    encodeStream: (w, value: number) =>
      (w as _Writer).writeView(len, (view) => opts.write(view, value)),
    decodeStream: (r) => (r as _Reader).readView(len, opts.read),
    validate: (value: number) => {
      if (typeof value !== 'number')
        throw new Error(`viewCoder: expected number, got ${typeof value}`);
      if (opts.validate) opts.validate(value);
      return value;
    },
  });

const intView = (len: number, signed: boolean, opts: ViewCoder) => {
  const bits = len * 8;
  const signBit = 2 ** (bits - 1);
  // Inlined checkBounds for integer
  const validateSigned = (value: number) => {
    if (!isNum(value)) throw new Error(`sintView: value is not safe integer: ${value}`);
    if (value < -signBit || value >= signBit) {
      throw new Error(
        `sintView: value out of bounds. Expected ${-signBit} <= ${value} < ${signBit}`
      );
    }
  };
  const maxVal = 2 ** bits;
  const validateUnsigned = (value: number) => {
    if (!isNum(value)) throw new Error(`uintView: value is not safe integer: ${value}`);
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
export const U32LE: CoderType<number> = /* @__PURE__ */ intView(4, false, {
  read: (view, pos) => view.getUint32(pos, true),
  write: (view, value) => view.setUint32(0, value, true),
});
/** Unsigned 32-bit big-endian integer CoderType. */
export const U32BE: CoderType<number> = /* @__PURE__ */ intView(4, false, {
  read: (view, pos) => view.getUint32(pos, false),
  write: (view, value) => view.setUint32(0, value, false),
});
/** Signed 32-bit little-endian integer CoderType. */
export const I32LE: CoderType<number> = /* @__PURE__ */ intView(4, true, {
  read: (view, pos) => view.getInt32(pos, true),
  write: (view, value) => view.setInt32(0, value, true),
});
/** Signed 32-bit big-endian integer CoderType. */
export const I32BE: CoderType<number> = /* @__PURE__ */ intView(4, true, {
  read: (view, pos) => view.getInt32(pos, false),
  write: (view, value) => view.setInt32(0, value, false),
});
/** Unsigned 16-bit little-endian integer CoderType. */
export const U16LE: CoderType<number> = /* @__PURE__ */ intView(2, false, {
  read: (view, pos) => view.getUint16(pos, true),
  write: (view, value) => view.setUint16(0, value, true),
});
/** Unsigned 16-bit big-endian integer CoderType. */
export const U16BE: CoderType<number> = /* @__PURE__ */ intView(2, false, {
  read: (view, pos) => view.getUint16(pos, false),
  write: (view, value) => view.setUint16(0, value, false),
});
/** Signed 16-bit little-endian integer CoderType. */
export const I16LE: CoderType<number> = /* @__PURE__ */ intView(2, true, {
  read: (view, pos) => view.getInt16(pos, true),
  write: (view, value) => view.setInt16(0, value, true),
});
/** Signed 16-bit big-endian integer CoderType. */
export const I16BE: CoderType<number> = /* @__PURE__ */ intView(2, true, {
  read: (view, pos) => view.getInt16(pos, false),
  write: (view, value) => view.setInt16(0, value, false),
});
/** Unsigned 8-bit integer CoderType. */
export const U8: CoderType<number> = /* @__PURE__ */ intView(1, false, {
  read: (view, pos) => view.getUint8(pos),
  write: (view, value) => view.setUint8(0, value),
});
/** Signed 8-bit integer CoderType. */
export const I8: CoderType<number> = /* @__PURE__ */ intView(1, true, {
  read: (view, pos) => view.getInt8(pos),
  write: (view, value) => view.setInt8(0, value),
});

// Floats
const f32 = (le?: boolean) =>
  view(4, {
    read: (view, pos) => view.getFloat32(pos, le),
    write: (view, value) => view.setFloat32(0, value, le),
    validate: (value) => {
      if (Math.fround(value) !== value && !Number.isNaN(value))
        throw new Error(`f32: wrong value=${value}`);
    },
  });
const f64 = (le?: boolean) =>
  view(8, {
    read: (view, pos) => view.getFloat64(pos, le),
    write: (view, value) => view.setFloat64(0, value, le),
  });

/** 32-bit big-endian floating point CoderType ("binary32", IEEE 754-2008). */
export const F32BE: CoderType<number> = /* @__PURE__ */ f32(false);
/** 32-bit little-endian floating point  CoderType ("binary32", IEEE 754-2008). */
export const F32LE: CoderType<number> = /* @__PURE__ */ f32(true);
/** A 64-bit big-endian floating point type ("binary64", IEEE 754-2008). Any JS number can be encoded. */
export const F64BE: CoderType<number> = /* @__PURE__ */ f64(false);
/** A 64-bit little-endian floating point type ("binary64", IEEE 754-2008). Any JS number can be encoded. */
export const F64LE: CoderType<number> = /* @__PURE__ */ f64(true);

/** Boolean CoderType. */
export const bool: CoderType<boolean> = /* @__PURE__ */ wrap({
  size: 1,
  encodeStream: (w: Writer, value: boolean) => w.byte(value ? 1 : 0),
  decodeStream: (r: Reader): boolean => {
    const value = r.byte();
    if (value !== 0 && value !== 1) throw r.err(`bool: invalid value ${value}`);
    return value === 1;
  },
  validate: (value) => {
    if (typeof value !== 'boolean') throw new Error(`bool: invalid value ${value}`);
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
const createBytes = (len: Length, le = false): CoderType<Bytes> => {
  if (typeof le !== 'boolean') throw new Error(`bytes/le: expected boolean, got ${typeof le}`);
  const _length = lengthCoder(len);
  const _isb = isBytes(len);
  return wrap({
    size: typeof len === 'number' ? len : undefined,
    encodeStream: (w: Writer, value: Bytes) => {
      if (!_isb) _length.encodeStream(w, value.length);
      w.bytes(le ? swapEndianness(value) : value);
      if (_isb) w.bytes(len);
    },
    decodeStream: (r: Reader): Bytes => {
      let bytes: Bytes;
      if (_isb) {
        const tPos = r.find(len);
        if (!tPos) throw r.err(`bytes: cannot find terminator`);
        bytes = r.bytes(tPos - r.pos);
        r.bytes(len.length);
      } else {
        bytes = r.bytes(len === null ? r.leftBytes : _length.decodeStream(r));
      }
      return le ? swapEndianness(bytes) : bytes;
    },
    validate: (value) => {
      if (!isBytes(value)) throw new Error(`bytes: invalid value ${value}`);
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
export function prefix<T>(len: Length, inner: CoderType<T>): CoderType<T> {
  if (!isCoder(inner)) throw new Error(`prefix: invalid inner value ${inner}`);
  return apply(createBytes(len), reverse(inner)) as CoderType<T>;
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
export const string = (len: Length, le = false): CoderType<string> =>
  validate(apply(createBytes(len, le), utf8), (value) => {
    // TextEncoder/TextDecoder will fail on non-string, but we create more readable errors earlier
    if (typeof value !== 'string') throw new Error(`expected string, got ${typeof value}`);
    return value;
  });

/** NUL-terminated string CoderType. */
export const cstring: CoderType<string> = /* @__PURE__ */ string(NULL);

type HexOpts = { isLE?: boolean; with0x?: boolean };
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
const createHex = (
  len: Length,
  options: HexOpts = { isLE: false, with0x: false }
): CoderType<string> => {
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
export function apply<T, F>(inner: CoderType<T>, base: BaseCoder<T, F>): CoderType<F> {
  if (!isCoder(inner)) throw new Error(`apply: invalid inner value ${inner}`);
  if (!isBaseCoder(base)) throw new Error(`apply: invalid base value ${inner}`);
  return wrap({
    size: inner.size,
    encodeStream: (w: Writer, value: F) => {
      let innerValue;
      try {
        innerValue = base.decode(value);
      } catch (e) {
        throw w.err('' + e);
      }
      return inner.encodeStream(w, innerValue);
    },
    decodeStream: (r: Reader): F => {
      const innerValue = inner.decodeStream(r);
      try {
        return base.encode(innerValue);
      } catch (e) {
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
export function lazy<T>(fn: () => CoderType<T>): CoderType<T> {
  if (typeof fn !== 'function') throw new Error(`lazy: expected function, got ${typeof fn}`);
  return wrap({
    encodeStream: (w: Writer, value: T) => fn().encodeStream(w, value),
    decodeStream: (r: Reader): T => fn().decodeStream(r),
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
export const flag = (flagValue: Bytes, xor = false): CoderType<boolean | undefined> => {
  if (!isBytes(flagValue))
    throw new Error(`flag/flagValue: expected Uint8Array, got ${typeof flagValue}`);
  if (typeof xor !== 'boolean') throw new Error(`flag/xor: expected boolean, got ${typeof xor}`);
  return wrap({
    size: flagValue.length,
    encodeStream: (w: Writer, value: boolean | undefined) => {
      if (!!value !== xor) w.bytes(flagValue);
    },
    decodeStream: (r: Reader): boolean | undefined => {
      let hasFlag = r.leftBytes >= flagValue.length;
      if (hasFlag) {
        hasFlag = equalBytes(r.bytes(flagValue.length, true), flagValue);
        // Found flag, advance cursor position
        if (hasFlag) r.bytes(flagValue.length);
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
export function flagged<T>(
  path: string | CoderType<boolean>,
  inner: CoderType<T>,
  def?: T
): CoderType<Option<T>> {
  if (!isCoder(inner)) throw new Error(`flagged: invalid inner value ${inner}`);
  if (typeof path !== 'string' && !isCoder(inner)) throw new Error(`flagged: wrong path=${path}`);
  return wrap({
    encodeStream: (w: Writer, value: Option<T>) => {
      if (typeof path === 'string') {
        if (Path.resolve((w as _Writer).stack, path)) inner.encodeStream(w, value);
        else if (def) inner.encodeStream(w, def);
      } else {
        path.encodeStream(w, !!value);
        if (!!value) inner.encodeStream(w, value);
        else if (def) inner.encodeStream(w, def);
      }
    },
    decodeStream: (r: Reader): Option<T> => {
      let hasFlag = false;
      if (typeof path === 'string') hasFlag = !!Path.resolve((r as _Reader).stack, path);
      else hasFlag = path.decodeStream(r);
      // If there is a flag -- decode and return value
      if (hasFlag) return inner.decodeStream(r);
      else if (def) inner.decodeStream(r);
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
export function optional<T>(
  flag: CoderType<boolean>,
  inner: CoderType<T>,
  def?: T
): CoderType<Option<T>> {
  if (!isCoder(flag) || !isCoder(inner))
    throw new Error(`optional: invalid flag or inner value flag=${flag} inner=${inner}`);
  return wrap({
    size: def !== undefined && flag.size && inner.size ? flag.size + inner.size : undefined,
    encodeStream: (w: Writer, value: Option<T>) => {
      flag.encodeStream(w, !!value);
      if (value) inner.encodeStream(w, value);
      else if (def !== undefined) inner.encodeStream(w, def);
    },
    decodeStream: (r: Reader): Option<T> => {
      if (flag.decodeStream(r)) return inner.decodeStream(r);
      else if (def !== undefined) inner.decodeStream(r);
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
export function magic<T>(inner: CoderType<T>, constant: T, check = true): CoderType<undefined> {
  if (!isCoder(inner)) throw new Error(`magic: invalid inner value ${inner}`);
  if (typeof check !== 'boolean') throw new Error(`magic: expected boolean, got ${typeof check}`);
  return wrap({
    size: inner.size,
    encodeStream: (w: Writer, _value: undefined) => inner.encodeStream(w, constant),
    decodeStream: (r: Reader): undefined => {
      const value = inner.decodeStream(r);
      if (
        (check && typeof value !== 'object' && value !== constant) ||
        (isBytes(constant) && !equalBytes(constant, value as any))
      ) {
        throw r.err(`magic: invalid value: ${value} !== ${constant}`);
      }
      return;
    },
    validate: (value) => {
      if (value !== undefined) throw new Error(`magic: wrong value=${typeof value}`);
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
export const magicBytes = (constant: Bytes | string): CoderType<undefined> => {
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
export function constant<T>(c: T): CoderType<T> {
  return wrap({
    encodeStream: (_w: Writer, value: T) => {
      if (value !== c) throw new Error(`constant: invalid value ${value} (exp: ${c})`);
    },
    decodeStream: (_r: Reader): T => c,
  });
}

function sizeof(fields: CoderType<any>[]): Option<number> {
  let size: Option<number> = 0;
  for (const f of fields) {
    if (f.size === undefined) return;
    if (!isNum(f.size)) throw new Error(`sizeof: wrong element size=${size}`);
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
export function struct<T extends Record<string, any>>(
  fields: StructRecord<T>
): CoderType<StructInput<T>> {
  if (!isPlainObject(fields)) throw new Error(`struct: expected plain object, got ${fields}`);
  for (const name in fields) {
    if (!isCoder(fields[name])) throw new Error(`struct: field ${name} is not CoderType`);
  }
  return wrap({
    size: sizeof(Object.values(fields)),
    encodeStream: (w: Writer, value: StructInput<T>) => {
      (w as _Writer).pushObj(value, (fieldFn) => {
        for (const name in fields)
          fieldFn(name, () => fields[name].encodeStream(w, (value as T)[name]));
      });
    },
    decodeStream: (r: Reader): StructInput<T> => {
      const res: Partial<T> = {};
      (r as _Reader).pushObj(res, (fieldFn) => {
        for (const name in fields) fieldFn(name, () => (res[name] = fields[name].decodeStream(r)));
      });
      return res as T;
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
export function tuple<
  T extends ArrLike<CoderType<any>>,
  O = Writable<{ [K in keyof T]: UnwrapCoder<T[K]> }>,
>(fields: T): CoderType<O> {
  if (!Array.isArray(fields))
    throw new Error(`Packed.Tuple: got ${typeof fields} instead of array`);
  for (let i = 0; i < fields.length; i++) {
    if (!isCoder(fields[i])) throw new Error(`tuple: field ${i} is not CoderType`);
  }
  return wrap({
    size: sizeof(fields),
    encodeStream: (w: Writer, value: O) => {
      // TODO: fix types
      if (!Array.isArray(value)) throw w.err(`tuple: invalid value ${value}`);
      (w as _Writer).pushObj(value, (fieldFn) => {
        for (let i = 0; i < fields.length; i++)
          fieldFn(`${i}`, () => fields[i].encodeStream(w, value[i]));
      });
    },
    decodeStream: (r: Reader): O => {
      const res: any = [];
      (r as _Reader).pushObj(res, (fieldFn) => {
        for (let i = 0; i < fields.length; i++)
          fieldFn(`${i}`, () => res.push(fields[i].decodeStream(r)));
      });
      return res;
    },
    validate: (value) => {
      if (!Array.isArray(value)) throw new Error(`tuple: invalid value ${value}`);
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
export function array<T>(len: Length, inner: CoderType<T>): CoderType<T[]> {
  if (!isCoder(inner)) throw new Error(`array: invalid inner value ${inner}`);
  // By construction length is inside array (otherwise there will be various incorrect stack states)
  // But forcing users always write '..' seems like bad idea. Also, breaking change.
  const _length = lengthCoder(typeof len === 'string' ? `../${len}` : len);
  return wrap({
    size: typeof len === 'number' && inner.size ? len * inner.size : undefined,
    encodeStream: (w: Writer, value: T[]) => {
      const _w = w as _Writer;
      _w.pushObj(value, (fieldFn) => {
        if (!isBytes(len)) _length.encodeStream(w, value.length);
        for (let i = 0; i < value.length; i++) {
          fieldFn(`${i}`, () => {
            const elm = value[i];
            const startPos = (w as _Writer).pos;
            inner.encodeStream(w, elm);
            if (isBytes(len)) {
              // Terminator is bigger than elm size, so skip
              if (len.length > _w.pos - startPos) return;
              const data = _w.finish(false).subarray(startPos, _w.pos);
              // There is still possible case when multiple elements create terminator,
              // but it is hard to catch here, will be very slow
              if (equalBytes(data.subarray(0, len.length), len))
                throw _w.err(
                  `array: inner element encoding same as separator. elm=${elm} data=${data}`
                );
            }
          });
        }
      });
      if (isBytes(len)) w.bytes(len);
    },
    decodeStream: (r: Reader): T[] => {
      const res: T[] = [];
      (r as _Reader).pushObj(res, (fieldFn) => {
        if (len === null) {
          for (let i = 0; !r.isEnd(); i++) {
            fieldFn(`${i}`, () => res.push(inner.decodeStream(r)));
            if (inner.size && r.leftBytes < inner.size) break;
          }
        } else if (isBytes(len)) {
          for (let i = 0; ; i++) {
            if (equalBytes(r.bytes(len.length, true), len)) {
              // Advance cursor position if terminator found
              r.bytes(len.length);
              break;
            }
            fieldFn(`${i}`, () => res.push(inner.decodeStream(r)));
          }
        } else {
          let length: number;
          fieldFn('arrayLen', () => (length = _length.decodeStream(r)));
          for (let i = 0; i < length!; i++) fieldFn(`${i}`, () => res.push(inner.decodeStream(r)));
        }
      });
      return res;
    },
    validate: (value) => {
      if (!Array.isArray(value)) throw new Error(`array: invalid value ${value}`);
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
export function map<T>(inner: CoderType<T>, variants: Record<string, T>): CoderType<string> {
  if (!isCoder(inner)) throw new Error(`map: invalid inner value ${inner}`);
  if (!isPlainObject(variants)) throw new Error(`map: variants should be plain object`);
  const variantNames: Map<T, string> = new Map();
  for (const k in variants) variantNames.set(variants[k], k);
  return wrap({
    size: inner.size,
    encodeStream: (w: Writer, value: string) => inner.encodeStream(w, variants[value]),
    decodeStream: (r: Reader): string => {
      const variant = inner.decodeStream(r);
      const name = variantNames.get(variant);
      if (name === undefined)
        throw r.err(`Enum: unknown value: ${variant} ${Array.from(variantNames.keys())}`);
      return name;
    },
    validate: (value) => {
      if (typeof value !== 'string') throw new Error(`map: invalid value ${value}`);
      if (!(value in variants)) throw new Error(`Map: unknown variant: ${value}`);
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
export function tag<
  T extends Values<{
    [P in keyof Variants]: { TAG: P; data: UnwrapCoder<Variants[P]> };
  }>,
  TagValue extends string | number,
  Variants extends Record<TagValue, CoderType<any>>,
>(tag: CoderType<TagValue>, variants: Variants): CoderType<T> {
  if (!isCoder(tag)) throw new Error(`tag: invalid tag value ${tag}`);
  if (!isPlainObject(variants)) throw new Error(`tag: variants should be plain object`);
  for (const name in variants) {
    if (!isCoder(variants[name])) throw new Error(`tag: variant ${name} is not CoderType`);
  }
  return wrap({
    size: tag.size,
    encodeStream: (w: Writer, value: T) => {
      const { TAG, data } = value;
      const dataType = variants[TAG];
      tag.encodeStream(w, TAG as any);
      dataType.encodeStream(w, data);
    },
    decodeStream: (r: Reader): T => {
      const TAG = tag.decodeStream(r);
      const dataType = variants[TAG];
      if (!dataType) throw r.err(`Tag: invalid tag ${TAG}`);
      return { TAG, data: dataType.decodeStream(r) } as any;
    },
    validate: (value) => {
      const { TAG } = value;
      const dataType = variants[TAG];
      if (!dataType) throw new Error(`Tag: invalid tag ${TAG.toString()}`);
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
export function mappedTag<
  T extends Values<{
    [P in keyof Variants]: { TAG: P; data: UnwrapCoder<Variants[P][1]> };
  }>,
  TagValue extends string | number,
  Variants extends Record<string, [TagValue, CoderType<any>]>,
>(tagCoder: CoderType<TagValue>, variants: Variants): CoderType<T> {
  if (!isCoder(tagCoder)) throw new Error(`mappedTag: invalid tag value ${tag}`);
  if (!isPlainObject(variants)) throw new Error(`mappedTag: variants should be plain object`);
  const mapValue: Record<string, TagValue> = {};
  const tagValue: Record<string, CoderType<any>> = {};
  for (const key in variants) {
    const v = variants[key];
    mapValue[key] = v[0];
    tagValue[key] = v[1];
  }
  return tag(map(tagCoder, mapValue), tagValue) as any as CoderType<T>;
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
export function bitset<Names extends readonly string[]>(
  names: Names,
  pad = false
): CoderType<Record<Names[number], boolean>> {
  if (typeof pad !== 'boolean') throw new Error(`bitset/pad: expected boolean, got ${typeof pad}`);
  if (!Array.isArray(names)) throw new Error('bitset/names: expected array');
  for (const name of names) {
    if (typeof name !== 'string') throw new Error('bitset/names: expected array of strings');
  }
  return wrap({
    encodeStream: (w: Writer, value: Record<Names[number], boolean>) => {
      for (let i = 0; i < names.length; i++) w.bits(+(value as any)[names[i]], 1);
      if (pad && names.length % 8) w.bits(0, 8 - (names.length % 8));
    },
    decodeStream: (r: Reader): Record<Names[number], boolean> => {
      const out: Record<string, boolean> = {};
      for (let i = 0; i < names.length; i++) out[names[i]] = !!r.bits(1);
      if (pad && names.length % 8) r.bits(8 - (names.length % 8));
      return out;
    },
    validate: (value) => {
      if (!isPlainObject(value)) throw new Error(`bitset: invalid value ${value}`);
      for (const v of Object.values(value)) {
        if (typeof v !== 'boolean') throw new Error('expected boolean');
      }
      return value;
    },
  });
}
/** Padding function which always returns zero */
export const ZeroPad: PadFn = (_) => 0;

function padLength(blockSize: number, len: number): number {
  if (len % blockSize === 0) return 0;
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
export function padLeft<T>(
  blockSize: number,
  inner: CoderType<T>,
  padFn: Option<PadFn>
): CoderType<T> {
  if (!isNum(blockSize) || blockSize <= 0) throw new Error(`padLeft: wrong blockSize=${blockSize}`);
  if (!isCoder(inner)) throw new Error(`padLeft: invalid inner value ${inner}`);
  if (padFn !== undefined && typeof padFn !== 'function')
    throw new Error(`padLeft: wrong padFn=${typeof padFn}`);
  const _padFn = padFn || ZeroPad;
  if (!inner.size) throw new Error('padLeft cannot have dynamic size');
  return wrap({
    size: inner.size + padLength(blockSize, inner.size),
    encodeStream: (w: Writer, value: T) => {
      const padBytes = padLength(blockSize, inner.size!);
      for (let i = 0; i < padBytes; i++) w.byte(_padFn(i));
      inner.encodeStream(w, value);
    },
    decodeStream: (r: Reader): T => {
      r.bytes(padLength(blockSize, inner.size!));
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
export function padRight<T>(
  blockSize: number,
  inner: CoderType<T>,
  padFn: Option<PadFn>
): CoderType<T> {
  if (!isCoder(inner)) throw new Error(`padRight: invalid inner value ${inner}`);
  if (!isNum(blockSize) || blockSize <= 0) throw new Error(`padLeft: wrong blockSize=${blockSize}`);
  if (padFn !== undefined && typeof padFn !== 'function')
    throw new Error(`padRight: wrong padFn=${typeof padFn}`);
  const _padFn = padFn || ZeroPad;
  return wrap({
    size: inner.size ? inner.size + padLength(blockSize, inner.size) : undefined,
    encodeStream: (w: Writer, value: T) => {
      const _w = w as _Writer;
      const pos = _w.pos;
      inner.encodeStream(w, value);
      const padBytes = padLength(blockSize, _w.pos - pos);
      for (let i = 0; i < padBytes; i++) w.byte(_padFn(i));
    },
    decodeStream: (r: Reader): T => {
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
export function pointer<T>(
  ptr: CoderType<number>,
  inner: CoderType<T>,
  sized = false
): CoderType<T> {
  if (!isCoder(ptr)) throw new Error(`pointer: invalid ptr value ${ptr}`);
  if (!isCoder(inner)) throw new Error(`pointer: invalid inner value ${inner}`);
  if (typeof sized !== 'boolean')
    throw new Error(`pointer/sized: expected boolean, got ${typeof sized}`);
  if (!ptr.size) throw new Error('unsized pointer');
  return wrap({
    size: sized ? ptr.size : undefined,
    encodeStream: (w: Writer, value: T) => {
      const _w = w as _Writer;
      const start = _w.pos;
      ptr.encodeStream(w, 0);
      _w.ptrs.push({ pos: start, ptr, buffer: inner.encode(value) });
    },
    decodeStream: (r: Reader): T => {
      const ptrVal = ptr.decodeStream(r);
      (r as _Reader)._enablePointers();
      return inner.decodeStream(r.offsetReader(ptrVal));
    },
  });
}

// Internal methods for test purposes only
export const _TEST: {
  _bitset: {
    BITS: number;
    FULL_MASK: number;
    len: (len: number) => number;
    create: (len: number) => Uint32Array;
    clean: (bs: Uint32Array) => Uint32Array;
    debug: (bs: Uint32Array) => string[];
    checkLen: (bs: Uint32Array, len: number) => void;
    chunkLen: (bsLen: number, pos: number, len: number) => void;
    set: (bs: Uint32Array, chunk: number, value: number, allowRewrite?: boolean) => boolean;
    pos: (
      pos: number,
      i: number
    ) => {
      chunk: number;
      mask: number;
    };
    indices: (bs: Uint32Array, len: number, invert?: boolean) => number[];
    range: (arr: number[]) => {
      pos: number;
      length: number;
    }[];
    rangeDebug: (bs: Uint32Array, len: number, invert?: boolean) => string;
    setRange: (
      bs: Uint32Array,
      bsLen: number,
      pos: number,
      len: number,
      allowRewrite?: boolean
    ) => boolean;
  };
  _Reader: typeof _Reader;
  _Writer: typeof _Writer;
  Path: {
    /**
     * Internal method for handling stack of paths (debug, errors, dynamic fields via path)
     * This is looks ugly (callback), but allows us to force stack cleaning by construction (.pop always after function).
     * Also, this makes impossible:
     * - pushing field when stack is empty
     * - pushing field inside of field (real bug)
     * NOTE: we don't want to do '.pop' on error!
     */
    pushObj: (stack: PathStack, obj: StructOut, objFn: _PathObjFn) => void;
    path: (stack: PathStack) => string;
    err(name: string, stack: PathStack, msg: string | Error): Error;
    resolve: (stack: PathStack, path: string) => StructOut | undefined;
  };
} = { _bitset: Bitset, _Reader, _Writer, Path };
