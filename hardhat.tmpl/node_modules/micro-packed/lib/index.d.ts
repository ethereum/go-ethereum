import { type Coder as BaseCoder } from '@scure/base';
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
/** Shortcut to zero-length (empty) byte array */
export declare const EMPTY: Uint8Array;
/** Shortcut to one-element (element is 0) byte array */
export declare const NULL: Uint8Array;
/** Checks if two Uint8Arrays are equal. Not constant-time. */
declare function equalBytes(a: Uint8Array, b: Uint8Array): boolean;
/** Checks if the given value is a Uint8Array. */
declare function isBytes(a: unknown): a is Bytes;
/**
 * Concatenates multiple Uint8Arrays.
 * Engines limit functions to 65K+ arguments.
 * @param arrays Array of Uint8Array elements
 * @returns Concatenated Uint8Array
 */
declare function concatBytes(...arrays: Uint8Array[]): Uint8Array;
/**
 * Checks if the provided value is a plain object, not created from any class or special constructor.
 * Array, Uint8Array and others are not plain objects.
 * @param obj - The value to be checked.
 */
declare function isPlainObject(obj: any): boolean;
export declare const utils: {
    equalBytes: typeof equalBytes;
    isBytes: typeof isBytes;
    isCoder: typeof isCoder;
    checkBounds: typeof checkBounds;
    concatBytes: typeof concatBytes;
    createView: (arr: Uint8Array) => DataView;
    isPlainObject: typeof isPlainObject;
};
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
    size?: number;
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
export type Sized<T> = CoderType<T> & {
    size: number;
};
export type UnwrapCoder<T> = T extends CoderType<infer U> ? U : T;
/**
 * Validation function. Should return value after validation.
 * Can be used to narrow types
 */
export type Validate<T> = (elm: T) => T;
export type Length = CoderType<number> | CoderType<bigint> | number | Bytes | string | null;
type ArrLike<T> = Array<T> | ReadonlyArray<T>;
export type TypedArray = Uint8Array | Int8Array | Uint8ClampedArray | Uint16Array | Int16Array | Uint32Array | Int32Array;
/** Writable version of a type, where readonly properties are made writable. */
export type Writable<T> = T extends {} ? T extends TypedArray ? T : {
    -readonly [P in keyof T]: Writable<T[P]>;
} : T;
export type Values<T> = T[keyof T];
export type NonUndefinedKey<T, K extends keyof T> = T[K] extends undefined ? never : K;
export type NullableKey<T, K extends keyof T> = T[K] extends NonNullable<T[K]> ? never : K;
export type OptKey<T, K extends keyof T> = NullableKey<T, K> & NonUndefinedKey<T, K>;
export type ReqKey<T, K extends keyof T> = T[K] extends NonNullable<T[K]> ? K : never;
export type OptKeys<T> = Pick<T, {
    [K in keyof T]: OptKey<T, K>;
}[keyof T]>;
export type ReqKeys<T> = Pick<T, {
    [K in keyof T]: ReqKey<T, K>;
}[keyof T]>;
export type StructInput<T extends Record<string, any>> = {
    [P in keyof ReqKeys<T>]: T[P];
} & {
    [P in keyof OptKeys<T>]?: T[P];
};
export type StructRecord<T extends Record<string, any>> = {
    [P in keyof T]: CoderType<T[P]>;
};
export type StructOut = Record<string, any>;
/** Padding function that takes an index and returns a padding value. */
export type PadFn = (i: number) => number;
/** Path related utils (internal) */
type Path = {
    obj: StructOut;
    field?: string;
};
type PathStack = Path[];
export type _PathObjFn = (cb: (field: string, fieldFn: Function) => void) => void;
declare const Path: {
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
    err: (name: string, stack: PathStack, msg: string | Error) => Error;
    resolve: (stack: PathStack, path: string) => StructOut | undefined;
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
export type Reader = {
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
declare class _Reader implements Reader {
    pos: number;
    readonly data: Bytes;
    readonly opts: ReaderOpts;
    readonly stack: PathStack;
    private parent;
    private parentOffset;
    private bitBuf;
    private bitPos;
    private bs;
    private view;
    constructor(data: Bytes, opts?: ReaderOpts, stack?: PathStack, parent?: _Reader | undefined, parentOffset?: number);
    /** Internal method for pointers. */
    _enablePointers(): void;
    private markBytesBS;
    private markBytes;
    pushObj(obj: StructOut, objFn: _PathObjFn): void;
    readView(n: number, fn: (view: DataView, pos: number) => number): number;
    absBytes(n: number): Uint8Array;
    finish(): void;
    err(msg: string | Error): Error;
    offsetReader(n: number): _Reader;
    bytes(n: number, peek?: boolean): Uint8Array;
    byte(peek?: boolean): number;
    get leftBytes(): number;
    get totalBytes(): number;
    isEnd(): boolean;
    bits(bits: number): number;
    find(needle: Bytes, pos?: number): number | undefined;
}
/**
 * Internal structure. Writer class for writing to a byte array.
 * The `stack` argument of constructor is internal, for debugging and logs.
 * @class Writer
 */
declare class _Writer implements Writer {
    pos: number;
    readonly stack: PathStack;
    private buffers;
    ptrs: {
        pos: number;
        ptr: CoderType<number>;
        buffer: Bytes;
    }[];
    private bitBuf;
    private bitPos;
    private viewBuf;
    private view;
    private finished;
    constructor(stack?: PathStack);
    pushObj(obj: StructOut, objFn: _PathObjFn): void;
    writeView(len: number, fn: (view: DataView) => void): void;
    err(msg: string | Error): Error;
    bytes(b: Bytes): void;
    byte(b: number): void;
    finish(clean?: boolean): Bytes;
    bits(value: number, bits: number): void;
}
/** Internal function for checking bit bounds of bigint in signed/unsinged form */
declare function checkBounds(value: bigint, bits: bigint, signed: boolean): void;
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
export declare function validate<T>(inner: CoderType<T>, fn: Validate<T>): CoderType<T>;
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
export declare const wrap: <T>(inner: BytesCoderStream<T> & {
    validate?: Validate<T>;
}) => CoderType<T>;
/**
 * Checks if the given value is a CoderType.
 * @param elm - The value to check.
 * @returns True if the value is a CoderType, false otherwise.
 */
export declare function isCoder<T>(elm: any): elm is CoderType<T>;
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
declare function dict<T>(): BaseCoder<[string, T][], Record<string, T>>;
type Enum = {
    [k: string]: number | string;
} & {
    [k: number]: string;
};
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
declare function tsEnum<T extends Enum>(e: T): BaseCoder<number, EnumKeys<T>>;
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
declare function decimal(precision: number, round?: boolean): Coder<bigint, string>;
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
declare function match<L extends BaseCoder<unknown | undefined, unknown | undefined>[], I = {
    [K in keyof L]: NonNullable<BaseInput<L[K]>>;
}[number], O = {
    [K in keyof L]: NonNullable<BaseOutput<L[K]>>;
}[number]>(lst: L): BaseCoder<I, O>;
export declare const coders: {
    dict: typeof dict;
    numberBigint: BaseCoder<bigint, number>;
    tsEnum: typeof tsEnum;
    decimal: typeof decimal;
    match: typeof match;
    reverse: <F, T>(coder: Coder<F, T>) => Coder<T, F>;
};
/**
 * CoderType for parsing individual bits.
 * NOTE: Structure should parse whole amount of bytes before it can start parsing byte-level elements.
 * @param len - Number of bits to parse.
 * @returns CoderType representing the parsed bits.
 * @example
 * const s = P.struct({ magic: P.bits(1), version: P.bits(1), tag: P.bits(4), len: P.bits(2) });
 */
export declare const bits: (len: number) => CoderType<number>;
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
export declare const bigint: (size: number, le?: boolean, signed?: boolean, sized?: boolean) => CoderType<bigint>;
/** Unsigned 256-bit little-endian integer CoderType. */
export declare const U256LE: CoderType<bigint>;
/** Unsigned 256-bit big-endian integer CoderType. */
export declare const U256BE: CoderType<bigint>;
/** Signed 256-bit little-endian integer CoderType. */
export declare const I256LE: CoderType<bigint>;
/** Signed 256-bit big-endian integer CoderType. */
export declare const I256BE: CoderType<bigint>;
/** Unsigned 128-bit little-endian integer CoderType. */
export declare const U128LE: CoderType<bigint>;
/** Unsigned 128-bit big-endian integer CoderType. */
export declare const U128BE: CoderType<bigint>;
/** Signed 128-bit little-endian integer CoderType. */
export declare const I128LE: CoderType<bigint>;
/** Signed 128-bit big-endian integer CoderType. */
export declare const I128BE: CoderType<bigint>;
/** Unsigned 64-bit little-endian integer CoderType. */
export declare const U64LE: CoderType<bigint>;
/** Unsigned 64-bit big-endian integer CoderType. */
export declare const U64BE: CoderType<bigint>;
/** Signed 64-bit little-endian integer CoderType. */
export declare const I64LE: CoderType<bigint>;
/** Signed 64-bit big-endian integer CoderType. */
export declare const I64BE: CoderType<bigint>;
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
export declare const int: (size: number, le?: boolean, signed?: boolean, sized?: boolean) => CoderType<number>;
/** Unsigned 32-bit little-endian integer CoderType. */
export declare const U32LE: CoderType<number>;
/** Unsigned 32-bit big-endian integer CoderType. */
export declare const U32BE: CoderType<number>;
/** Signed 32-bit little-endian integer CoderType. */
export declare const I32LE: CoderType<number>;
/** Signed 32-bit big-endian integer CoderType. */
export declare const I32BE: CoderType<number>;
/** Unsigned 16-bit little-endian integer CoderType. */
export declare const U16LE: CoderType<number>;
/** Unsigned 16-bit big-endian integer CoderType. */
export declare const U16BE: CoderType<number>;
/** Signed 16-bit little-endian integer CoderType. */
export declare const I16LE: CoderType<number>;
/** Signed 16-bit big-endian integer CoderType. */
export declare const I16BE: CoderType<number>;
/** Unsigned 8-bit integer CoderType. */
export declare const U8: CoderType<number>;
/** Signed 8-bit integer CoderType. */
export declare const I8: CoderType<number>;
/** 32-bit big-endian floating point CoderType ("binary32", IEEE 754-2008). */
export declare const F32BE: CoderType<number>;
/** 32-bit little-endian floating point  CoderType ("binary32", IEEE 754-2008). */
export declare const F32LE: CoderType<number>;
/** A 64-bit big-endian floating point type ("binary64", IEEE 754-2008). Any JS number can be encoded. */
export declare const F64BE: CoderType<number>;
/** A 64-bit little-endian floating point type ("binary64", IEEE 754-2008). Any JS number can be encoded. */
export declare const F64LE: CoderType<number>;
/** Boolean CoderType. */
export declare const bool: CoderType<boolean>;
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
declare const createBytes: (len: Length, le?: boolean) => CoderType<Bytes>;
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
export declare function prefix<T>(len: Length, inner: CoderType<T>): CoderType<T>;
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
export declare const string: (len: Length, le?: boolean) => CoderType<string>;
/** NUL-terminated string CoderType. */
export declare const cstring: CoderType<string>;
type HexOpts = {
    isLE?: boolean;
    with0x?: boolean;
};
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
declare const createHex: (len: Length, options?: HexOpts) => CoderType<string>;
/**
 * Applies a base coder to a CoderType.
 * @param inner - The inner CoderType.
 * @param b - The base coder to apply.
 * @returns CoderType representing the transformed value.
 * @example
 * import { hex } from '@scure/base';
 * const hex = P.apply(P.bytes(32), hex); // will decode bytes into a hex string
 */
export declare function apply<T, F>(inner: CoderType<T>, base: BaseCoder<T, F>): CoderType<F>;
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
export declare function lazy<T>(fn: () => CoderType<T>): CoderType<T>;
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
export declare const flag: (flagValue: Bytes, xor?: boolean) => CoderType<boolean | undefined>;
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
export declare function flagged<T>(path: string | CoderType<boolean>, inner: CoderType<T>, def?: T): CoderType<Option<T>>;
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
export declare function optional<T>(flag: CoderType<boolean>, inner: CoderType<T>, def?: T): CoderType<Option<T>>;
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
export declare function magic<T>(inner: CoderType<T>, constant: T, check?: boolean): CoderType<undefined>;
/**
 * Magic bytes CoderType that encodes/decodes a constant byte array or string.
 * @param constant - Constant byte array or string.
 * @returns CoderType representing the magic bytes.
 * @example
 * // Always encodes undefined into byte representation of string 'MAGIC'
 * const magicBytes = P.magicBytes('MAGIC');
 */
export declare const magicBytes: (constant: Bytes | string) => CoderType<undefined>;
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
export declare function constant<T>(c: T): CoderType<T>;
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
export declare function struct<T extends Record<string, any>>(fields: StructRecord<T>): CoderType<StructInput<T>>;
/**
 * Tuple (unnamed structure) of CoderTypes. Same as struct but with unnamed fields.
 * @param fields - Array of CoderTypes.
 * @returns CoderType representing the tuple.
 * @example
 * const myTuple = P.tuple([P.U8, P.U16LE, P.string(P.U8)]);
 */
export declare function tuple<T extends ArrLike<CoderType<any>>, O = Writable<{
    [K in keyof T]: UnwrapCoder<T[K]>;
}>>(fields: T): CoderType<O>;
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
export declare function array<T>(len: Length, inner: CoderType<T>): CoderType<T[]>;
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
export declare function map<T>(inner: CoderType<T>, variants: Record<string, T>): CoderType<string>;
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
export declare function tag<T extends Values<{
    [P in keyof Variants]: {
        TAG: P;
        data: UnwrapCoder<Variants[P]>;
    };
}>, TagValue extends string | number, Variants extends Record<TagValue, CoderType<any>>>(tag: CoderType<TagValue>, variants: Variants): CoderType<T>;
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
export declare function mappedTag<T extends Values<{
    [P in keyof Variants]: {
        TAG: P;
        data: UnwrapCoder<Variants[P][1]>;
    };
}>, TagValue extends string | number, Variants extends Record<string, [TagValue, CoderType<any>]>>(tagCoder: CoderType<TagValue>, variants: Variants): CoderType<T>;
/**
 * Bitset of boolean values with optional padding.
 * @param names - An array of string names for the bitset values.
 * @param pad - Whether to pad the bitset to a multiple of 8 bits.
 * @returns CoderType representing the bitset.
 * @template Names
 * @example
 * const myBitset = P.bitset(['flag1', 'flag2', 'flag3', 'flag4'], true);
 */
export declare function bitset<Names extends readonly string[]>(names: Names, pad?: boolean): CoderType<Record<Names[number], boolean>>;
/** Padding function which always returns zero */
export declare const ZeroPad: PadFn;
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
export declare function padLeft<T>(blockSize: number, inner: CoderType<T>, padFn: Option<PadFn>): CoderType<T>;
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
export declare function padRight<T>(blockSize: number, inner: CoderType<T>, padFn: Option<PadFn>): CoderType<T>;
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
export declare function pointer<T>(ptr: CoderType<number>, inner: CoderType<T>, sized?: boolean): CoderType<T>;
export declare const _TEST: {
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
        pos: (pos: number, i: number) => {
            chunk: number;
            mask: number;
        };
        indices: (bs: Uint32Array, len: number, invert?: boolean) => number[];
        range: (arr: number[]) => {
            pos: number;
            length: number;
        }[];
        rangeDebug: (bs: Uint32Array, len: number, invert?: boolean) => string;
        setRange: (bs: Uint32Array, bsLen: number, pos: number, len: number, allowRewrite?: boolean) => boolean;
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
};
//# sourceMappingURL=index.d.ts.map