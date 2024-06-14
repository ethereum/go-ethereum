/// <reference types="node" />
export = NoFilter;
/**
 * @typedef {object} NoFilterOptions
 * @property {string|Buffer} [input=null] Input source data.
 * @property {BufferEncoding} [inputEncoding=null] Encoding name for input,
 *   ignored if input is not a String.
 * @property {number} [highWaterMark=16384] The maximum number of bytes to
 *   store in the internal buffer before ceasing to read from the underlying
 *   resource. Default=16kb, or 16 for objectMode streams.
 * @property {BufferEncoding} [encoding=null] If specified, then buffers
 *   will be decoded to strings using the specified encoding.
 * @property {boolean} [objectMode=false] Whether this stream should behave
 *   as a stream of objects. Meaning that stream.read(n) returns a single
 *   value instead of a Buffer of size n.
 * @property {boolean} [decodeStrings=true] Whether or not to decode
 *   strings into Buffers before passing them to _write().
 * @property {boolean} [watchPipe=true] Whether to watch for 'pipe' events,
 *   setting this stream's objectMode based on the objectMode of the input
 *   stream.
 * @property {boolean} [readError=false] If true, when a read() underflows,
 *   throw an error.
 * @property {boolean} [allowHalfOpen=true] If set to false, then the
 *   stream will automatically end the writable side when the readable side
 *   ends.
 * @property {boolean} [autoDestroy=true] Whether this stream should
 *   automatically call .destroy() on itself after ending.
 * @property {BufferEncoding} [defaultEncoding='utf8'] The default encoding
 *   that is used when no encoding is specified as an argument to
 *   stream.write().
 * @property {boolean} [emitClose=true] Whether or not the stream should
 *   emit 'close' after it has been destroyed.
 * @property {number} [readableHighWaterMark] Sets highWaterMark for the
 *   readable side of the stream. Has no effect if highWaterMark is provided.
 * @property {boolean} [readableObjectMode=false] Sets objectMode for
 *   readable side of the stream. Has no effect if objectMode is true.
 * @property {number} [writableHighWaterMark] Sets highWaterMark for the
 *   writable side of the stream. Has no effect if highWaterMark is provided.
 * @property {boolean} [writableObjectMode=false] Sets objectMode for
 *   writable side of the stream. Has no effect if objectMode is true.
 */
/**
 * NoFilter stream.  Can be used to sink or source data to and from
 * other node streams.  Implemented as the "identity" Transform stream
 * (hence the name), but allows for inspecting data that is in-flight.
 *
 * Allows passing in source data (input, inputEncoding) at creation
 * time.  Source data can also be passed in the options object.
 *
 * @example <caption>source and sink</caption>
 * const source = new NoFilter('Zm9v', 'base64')
 * source.pipe(process.stdout)
 * const sink = new Nofilter()
 * // NOTE: 'finish' fires when the input is done writing
 * sink.on('finish', () => console.log(n.toString('base64')))
 * process.stdin.pipe(sink)
 */
declare class NoFilter extends stream.Transform {
    /**
     * Is the given object a {NoFilter}?
     *
     * @param {object} obj The object to test.
     * @returns {boolean} True if obj is a NoFilter.
     */
    static isNoFilter(obj: object): boolean;
    /**
     * The same as nf1.compare(nf2). Useful for sorting an Array of NoFilters.
     *
     * @param {NoFilter} nf1 The first object to compare.
     * @param {NoFilter} nf2 The second object to compare.
     * @returns {number} -1, 0, 1 for less, equal, greater.
     * @throws {TypeError} Arguments not NoFilter instances.
     * @example
     * const arr = [new NoFilter('1234'), new NoFilter('0123')]
     * arr.sort(NoFilter.compare)
     */
    static compare(nf1: NoFilter, nf2: NoFilter): number;
    /**
     * Returns a buffer which is the result of concatenating all the
     * NoFilters in the list together. If the list has no items, or if
     * the totalLength is 0, then it returns a zero-length buffer.
     *
     * If length is not provided, it is read from the buffers in the
     * list. However, this adds an additional loop to the function, so
     * it is faster to provide the length explicitly if you already know it.
     *
     * @param {Array<NoFilter>} list Inputs.  Must not be all either in object
     *   mode, or all not in object mode.
     * @param {number} [length=null] Number of bytes or objects to read.
     * @returns {Buffer|Array} The concatenated values as an array if in object
     *   mode, otherwise a Buffer.
     * @throws {TypeError} List not array of NoFilters.
     */
    static concat(list: Array<NoFilter>, length?: number): Buffer | any[];
    /**
     * Create an instance of NoFilter.
     *
     * @param {string|Buffer|BufferEncoding|NoFilterOptions} [input] Source data.
     * @param {BufferEncoding|NoFilterOptions} [inputEncoding] Encoding
     *   name for input, ignored if input is not a String.
     * @param {NoFilterOptions} [options] Other options.
     */
    constructor(input?: string | Buffer | BufferEncoding | NoFilterOptions, inputEncoding?: BufferEncoding | NoFilterOptions, options?: NoFilterOptions);
    readError: boolean;
    /**
     * @returns {Buffer[]} The current internal buffers.  They are layed out
     *   end to end.
     * @ignore
     */
    _bufArray(): Buffer[];
    /**
     * Read the full number of bytes asked for, no matter how long it takes.
     * Fail if an error occurs in the meantime, or if the stream finishes before
     * enough data is available.
     *
     * Note: This function won't work fully correctly if you are using
     * stream-browserify (for example, on the Web).
     *
     * @param {number} size The number of bytes to read.
     * @returns {Promise<string|Buffer>} A promise for the data read.
     */
    readFull(size: number): Promise<string | Buffer>;
    /**
     * Return a promise fulfilled with the full contents, after the 'finish'
     * event fires.  Errors on the stream cause the promise to be rejected.
     *
     * @param {Function} [cb=null] Finished/error callback used in *addition*
     *   to the promise.
     * @returns {Promise<Buffer|string>} Fulfilled when complete.
     */
    promise(cb?: Function): Promise<Buffer | string>;
    /**
     * Returns a number indicating whether this comes before or after or is the
     * same as the other NoFilter in sort order.
     *
     * @param {NoFilter} other The other object to compare.
     * @returns {number} -1, 0, 1 for less, equal, greater.
     * @throws {TypeError} Arguments must be NoFilters.
     */
    compare(other: NoFilter): number;
    /**
     * Do these NoFilter's contain the same bytes?  Doesn't work if either is
     * in object mode.
     *
     * @param {NoFilter} other Other NoFilter to compare against.
     * @returns {boolean} Equal?
     */
    equals(other: NoFilter): boolean;
    /**
     * Read bytes or objects without consuming them.  Useful for diagnostics.
     * Note: as a side-effect, concatenates multiple writes together into what
     * looks like a single write, so that this concat doesn't have to happen
     * multiple times when you're futzing with the same NoFilter.
     *
     * @param {number} [start=0] Beginning offset.
     * @param {number} [end=length] Ending offset.
     * @returns {Buffer|Array} If in object mode, an array of objects.  Otherwise,
     *   concatenated array of contents.
     */
    slice(start?: number, end?: number): Buffer | any[];
    /**
     * Get a byte by offset.  I didn't want to get into metaprogramming
     * to give you the `NoFilter[0]` syntax.
     *
     * @param {number} index The byte to retrieve.
     * @returns {number} 0-255.
     */
    get(index: number): number;
    /**
     * Return an object compatible with Buffer's toJSON implementation, so that
     * round-tripping will produce a Buffer.
     *
     * @returns {string|Array|{type: 'Buffer',data: number[]}} If in object mode,
     *   the objects.  Otherwise, JSON text.
     * @example <caption>output for 'foo', not in object mode</caption>
     * ({
     *   type: 'Buffer',
     *   data: [102, 111, 111],
     * })
     */
    toJSON(): string | any[] | {
        type: 'Buffer';
        data: number[];
    };
    /**
     * Decodes and returns a string from buffer data encoded using the specified
     * character set encoding. If encoding is undefined or null, then encoding
     * defaults to 'utf8'. The start and end parameters default to 0 and
     * NoFilter.length when undefined.
     *
     * @param {BufferEncoding} [encoding='utf8'] Which to use for decoding?
     * @param {number} [start=0] Start offset.
     * @param {number} [end=length] End offset.
     * @returns {string} String version of the contents.
     */
    toString(encoding?: BufferEncoding, start?: number, end?: number): string;
    /**
     * Current readable length, in bytes.
     *
     * @returns {number} Length of the contents.
     */
    get length(): number;
    /**
     * Write a JavaScript BigInt to the stream.  Negative numbers will be
     * written as their 2's complement version.
     *
     * @param {bigint} val The value to write.
     * @returns {boolean} True on success.
     */
    writeBigInt(val: bigint): boolean;
    /**
     * Read a variable-sized JavaScript unsigned BigInt from the stream.
     *
     * @param {number} [len=null] Number of bytes to read or all remaining
     *   if null.
     * @returns {bigint} A BigInt.
     */
    readUBigInt(len?: number): bigint;
    /**
     * Read a variable-sized JavaScript signed BigInt from the stream in 2's
     * complement format.
     *
     * @param {number} [len=null] Number of bytes to read or all remaining
     *   if null.
     * @returns {bigint} A BigInt.
     */
    readBigInt(len?: number): bigint;
    /**
     * Write an 8-bit unsigned integer to the stream.  Adds 1 byte.
     *
     * @param {number} value 0..255.
     * @returns {boolean} True on success.
     */
    writeUInt8(value: number): boolean;
    /**
     * Write a little-endian 16-bit unsigned integer to the stream.  Adds
     * 2 bytes.
     *
     * @param {number} value 0..65535.
     * @returns {boolean} True on success.
     */
    writeUInt16LE(value: number): boolean;
    /**
     * Write a big-endian 16-bit unsigned integer to the stream.  Adds
     * 2 bytes.
     *
     * @param {number} value 0..65535.
     * @returns {boolean} True on success.
     */
    writeUInt16BE(value: number): boolean;
    /**
     * Write a little-endian 32-bit unsigned integer to the stream.  Adds
     * 4 bytes.
     *
     * @param {number} value 0..2**32-1.
     * @returns {boolean} True on success.
     */
    writeUInt32LE(value: number): boolean;
    /**
     * Write a big-endian 32-bit unsigned integer to the stream.  Adds
     * 4 bytes.
     *
     * @param {number} value 0..2**32-1.
     * @returns {boolean} True on success.
     */
    writeUInt32BE(value: number): boolean;
    /**
     * Write a signed 8-bit integer to the stream.  Adds 1 byte.
     *
     * @param {number} value (-128)..127.
     * @returns {boolean} True on success.
     */
    writeInt8(value: number): boolean;
    /**
     * Write a signed little-endian 16-bit integer to the stream.  Adds 2 bytes.
     *
     * @param {number} value (-32768)..32767.
     * @returns {boolean} True on success.
     */
    writeInt16LE(value: number): boolean;
    /**
     * Write a signed big-endian 16-bit integer to the stream.  Adds 2 bytes.
     *
     * @param {number} value (-32768)..32767.
     * @returns {boolean} True on success.
     */
    writeInt16BE(value: number): boolean;
    /**
     * Write a signed little-endian 32-bit integer to the stream.  Adds 4 bytes.
     *
     * @param {number} value (-2**31)..(2**31-1).
     * @returns {boolean} True on success.
     */
    writeInt32LE(value: number): boolean;
    /**
     * Write a signed big-endian 32-bit integer to the stream.  Adds 4 bytes.
     *
     * @param {number} value (-2**31)..(2**31-1).
     * @returns {boolean} True on success.
     */
    writeInt32BE(value: number): boolean;
    /**
     * Write a little-endian 32-bit float to the stream.  Adds 4 bytes.
     *
     * @param {number} value 32-bit float.
     * @returns {boolean} True on success.
     */
    writeFloatLE(value: number): boolean;
    /**
     * Write a big-endian 32-bit float to the stream.  Adds 4 bytes.
     *
     * @param {number} value 32-bit float.
     * @returns {boolean} True on success.
     */
    writeFloatBE(value: number): boolean;
    /**
     * Write a little-endian 64-bit double to the stream.  Adds 8 bytes.
     *
     * @param {number} value 64-bit float.
     * @returns {boolean} True on success.
     */
    writeDoubleLE(value: number): boolean;
    /**
     * Write a big-endian 64-bit float to the stream.  Adds 8 bytes.
     *
     * @param {number} value 64-bit float.
     * @returns {boolean} True on success.
     */
    writeDoubleBE(value: number): boolean;
    /**
     * Write a signed little-endian 64-bit BigInt to the stream.  Adds 8 bytes.
     *
     * @param {bigint} value BigInt.
     * @returns {boolean} True on success.
     */
    writeBigInt64LE(value: bigint): boolean;
    /**
     * Write a signed big-endian 64-bit BigInt to the stream.  Adds 8 bytes.
     *
     * @param {bigint} value BigInt.
     * @returns {boolean} True on success.
     */
    writeBigInt64BE(value: bigint): boolean;
    /**
     * Write an unsigned little-endian 64-bit BigInt to the stream.  Adds 8 bytes.
     *
     * @param {bigint} value Non-negative BigInt.
     * @returns {boolean} True on success.
     */
    writeBigUInt64LE(value: bigint): boolean;
    /**
     * Write an unsigned big-endian 64-bit BigInt to the stream.  Adds 8 bytes.
     *
     * @param {bigint} value Non-negative BigInt.
     * @returns {boolean} True on success.
     */
    writeBigUInt64BE(value: bigint): boolean;
    /**
     * Read an unsigned 8-bit integer from the stream.  Consumes 1 byte.
     *
     * @returns {number} Value read.
     */
    readUInt8(): number;
    /**
     * Read a little-endian unsigned 16-bit integer from the stream.
     * Consumes 2 bytes.
     *
     * @returns {number} Value read.
     */
    readUInt16LE(): number;
    /**
     * Read a little-endian unsigned 16-bit integer from the stream.
     * Consumes 2 bytes.
     *
     * @returns {number} Value read.
     */
    readUInt16BE(): number;
    /**
     * Read a little-endian unsigned 32-bit integer from the stream.
     * Consumes 4 bytes.
     *
     * @returns {number} Value read.
     */
    readUInt32LE(): number;
    /**
     * Read a little-endian unsigned 16-bit integer from the stream.
     * Consumes 4 bytes.
     *
     * @returns {number} Value read.
     */
    readUInt32BE(): number;
    /**
     * Read a signed 8-bit integer from the stream.  Consumes 1 byte.
     *
     * @returns {number} Value read.
     */
    readInt8(): number;
    /**
     * Read a little-endian signed 16-bit integer from the stream.
     * Consumes 2 bytes.
     *
     * @returns {number} Value read.
     */
    readInt16LE(): number;
    /**
     * Read a little-endian signed 16-bit integer from the stream.
     * Consumes 2 bytes.
     *
     * @returns {number} Value read.
     */
    readInt16BE(): number;
    /**
     * Read a little-endian signed 32-bit integer from the stream.
     * Consumes 4 bytes.
     *
     * @returns {number} Value read.
     */
    readInt32LE(): number;
    /**
     * Read a little-endian signed 16-bit integer from the stream.
     * Consumes 4 bytes.
     *
     * @returns {number} Value read.
     */
    readInt32BE(): number;
    /**
     * Read a 32-bit little-endian float from the stream.
     * Consumes 4 bytes.
     *
     * @returns {number} Value read.
     */
    readFloatLE(): number;
    /**
     * Read a 32-bit big-endian float from the stream.
     * Consumes 4 bytes.
     *
     * @returns {number} Value read.
     */
    readFloatBE(): number;
    /**
     * Read a 64-bit little-endian float from the stream.
     * Consumes 8 bytes.
     *
     * @returns {number} Value read.
     */
    readDoubleLE(): number;
    /**
     * Read a 64-bit big-endian float from the stream.
     * Consumes 8 bytes.
     *
     * @returns {number} Value read.
     */
    readDoubleBE(): number;
    /**
     * Read a signed 64-bit little-endian BigInt from the stream.
     * Consumes 8 bytes.
     *
     * @returns {bigint} Value read.
     */
    readBigInt64LE(): bigint;
    /**
     * Read a signed 64-bit big-endian BigInt from the stream.
     * Consumes 8 bytes.
     *
     * @returns {bigint} Value read.
     */
    readBigInt64BE(): bigint;
    /**
     * Read an unsigned 64-bit little-endian BigInt from the stream.
     * Consumes 8 bytes.
     *
     * @returns {bigint} Value read.
     */
    readBigUInt64LE(): bigint;
    /**
     * Read an unsigned 64-bit big-endian BigInt from the stream.
     * Consumes 8 bytes.
     *
     * @returns {bigint} Value read.
     */
    readBigUInt64BE(): bigint;
}
declare namespace NoFilter {
    export { NoFilterOptions };
}
import stream = require("stream");
import { Buffer } from "buffer";
type NoFilterOptions = {
    /**
     * Input source data.
     */
    input?: string | Buffer;
    /**
     * Encoding name for input,
     * ignored if input is not a String.
     */
    inputEncoding?: BufferEncoding;
    /**
     * The maximum number of bytes to
     * store in the internal buffer before ceasing to read from the underlying
     * resource. Default=16kb, or 16 for objectMode streams.
     */
    highWaterMark?: number;
    /**
     * If specified, then buffers
     * will be decoded to strings using the specified encoding.
     */
    encoding?: BufferEncoding;
    /**
     * Whether this stream should behave
     * as a stream of objects. Meaning that stream.read(n) returns a single
     * value instead of a Buffer of size n.
     */
    objectMode?: boolean;
    /**
     * Whether or not to decode
     * strings into Buffers before passing them to _write().
     */
    decodeStrings?: boolean;
    /**
     * Whether to watch for 'pipe' events,
     * setting this stream's objectMode based on the objectMode of the input
     * stream.
     */
    watchPipe?: boolean;
    /**
     * If true, when a read() underflows,
     * throw an error.
     */
    readError?: boolean;
    /**
     * If set to false, then the
     * stream will automatically end the writable side when the readable side
     * ends.
     */
    allowHalfOpen?: boolean;
    /**
     * Whether this stream should
     * automatically call .destroy() on itself after ending.
     */
    autoDestroy?: boolean;
    /**
     * The default encoding
     * that is used when no encoding is specified as an argument to
     * stream.write().
     */
    defaultEncoding?: BufferEncoding;
    /**
     * Whether or not the stream should
     * emit 'close' after it has been destroyed.
     */
    emitClose?: boolean;
    /**
     * Sets highWaterMark for the
     * readable side of the stream. Has no effect if highWaterMark is provided.
     */
    readableHighWaterMark?: number;
    /**
     * Sets objectMode for
     * readable side of the stream. Has no effect if objectMode is true.
     */
    readableObjectMode?: boolean;
    /**
     * Sets highWaterMark for the
     * writable side of the stream. Has no effect if highWaterMark is provided.
     */
    writableHighWaterMark?: number;
    /**
     * Sets objectMode for
     * writable side of the stream. Has no effect if objectMode is true.
     */
    writableObjectMode?: boolean;
};
