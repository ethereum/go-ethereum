/// <reference types="node" />
export = Decoder;
/**
 * Decode a stream of CBOR bytes by transforming them into equivalent
 * JavaScript data.  Because of the limitations of Node object streams,
 * special symbols are emitted instead of NULL or UNDEFINED.  Fix those
 * up by calling {@link Decoder.nullcheck}.
 *
 * @extends BinaryParseStream
 */
declare class Decoder extends BinaryParseStream {
    /**
     * Check the given value for a symbol encoding a NULL or UNDEFINED value in
     * the CBOR stream.
     *
     * @static
     * @param {any} val The value to check.
     * @returns {any} The corrected value.
     * @throws {Error} Nothing was found.
     * @example
     * myDecoder.on('data', val => {
     *   val = Decoder.nullcheck(val)
     *   // ...
     * })
     */
    static nullcheck(val: any): any;
    /**
     * Decode the first CBOR item in the input, synchronously.  This will throw
     * an exception if the input is not valid CBOR, or if there are more bytes
     * left over at the end (if options.extendedResults is not true).
     *
     * @static
     * @param {BufferLike} input If a Readable stream, must have
     *   received the `readable` event already, or you will get an error
     *   claiming "Insufficient data".
     * @param {DecoderOptions|string} [options={}] Options or encoding for input.
     * @returns {ExtendedResults|any} The decoded value.
     * @throws {UnexpectedDataError} Data is left over after decoding.
     * @throws {Error} Insufficient data.
     */
    static decodeFirstSync(input: BufferLike, options?: DecoderOptions | string): ExtendedResults | any;
    /**
     * Decode all of the CBOR items in the input into an array.  This will throw
     * an exception if the input is not valid CBOR; a zero-length input will
     * return an empty array.
     *
     * @static
     * @param {BufferLike} input What to parse?
     * @param {DecoderOptions|string} [options={}] Options or encoding
     *   for input.
     * @returns {Array<ExtendedResults>|Array<any>} Array of all found items.
     * @throws {TypeError} No input provided.
     * @throws {Error} Insufficient data provided.
     */
    static decodeAllSync(input: BufferLike, options?: DecoderOptions | string): Array<ExtendedResults> | Array<any>;
    /**
     * Decode the first CBOR item in the input.  This will error if there are
     * more bytes left over at the end (if options.extendedResults is not true),
     * and optionally if there were no valid CBOR bytes in the input.  Emits the
     * {Decoder.NOT_FOUND} Symbol in the callback if no data was found and the
     * `required` option is false.
     *
     * @static
     * @param {BufferLike} input What to parse?
     * @param {DecoderOptions|decodeCallback|string} [options={}] Options, the
     *   callback, or input encoding.
     * @param {decodeCallback} [cb] Callback.
     * @returns {Promise<ExtendedResults|any>} Returned even if callback is
     *   specified.
     * @throws {TypeError} No input provided.
     */
    static decodeFirst(input: BufferLike, options?: DecoderOptions | decodeCallback | string, cb?: decodeCallback): Promise<ExtendedResults | any>;
    /**
     * @callback decodeAllCallback
     * @param {Error} error If one was generated.
     * @param {Array<ExtendedResults>|Array<any>} value All of the decoded
     *   values, wrapped in an Array.
     */
    /**
     * Decode all of the CBOR items in the input.  This will error if there are
     * more bytes left over at the end.
     *
     * @static
     * @param {BufferLike} input What to parse?
     * @param {DecoderOptions|decodeAllCallback|string} [options={}]
     *   Decoding options, the callback, or the input encoding.
     * @param {decodeAllCallback} [cb] Callback.
     * @returns {Promise<Array<ExtendedResults>|Array<any>>} Even if callback
     *   is specified.
     * @throws {TypeError} No input specified.
     */
    static decodeAll(input: BufferLike, options?: string | DecoderOptions | ((error: Error, value: Array<ExtendedResults> | Array<any>) => any), cb?: (error: Error, value: Array<ExtendedResults> | Array<any>) => any): Promise<Array<ExtendedResults> | Array<any>>;
    /**
     * Create a parsing stream.
     *
     * @param {DecoderOptions} [options={}] Options.
     */
    constructor(options?: DecoderOptions);
    running: boolean;
    max_depth: number;
    tags: {
        [x: string]: Tagged.TagFunction;
    };
    preferWeb: boolean;
    extendedResults: boolean;
    required: boolean;
    preventDuplicateKeys: boolean;
    valueBytes: NoFilter;
    /**
     * Stop processing.
     */
    close(): void;
    /**
     * Only called if extendedResults is true.
     *
     * @ignore
     */
    _onRead(data: any): void;
}
declare namespace Decoder {
    export { NOT_FOUND, BufferLike, ExtendedResults, DecoderOptions, decodeCallback };
}
import BinaryParseStream = require("../vendor/binary-parse-stream");
import Tagged = require("./tagged");
import NoFilter = require("nofilter");
/**
 * Things that can act as inputs, from which a NoFilter can be created.
 */
type BufferLike = string | Buffer | ArrayBuffer | Uint8Array | Uint8ClampedArray | DataView | stream.Readable;
type DecoderOptions = {
    /**
     * The maximum depth to parse.
     * Use -1 for "until you run out of memory".  Set this to a finite
     * positive number for un-trusted inputs.  Most standard inputs won't nest
     * more than 100 or so levels; I've tested into the millions before
     * running out of memory.
     */
    max_depth?: number;
    /**
     * Mapping from tag number to function(v),
     * where v is the decoded value that comes after the tag, and where the
     * function returns the correctly-created value for that tag.
     */
    tags?: Tagged.TagMap;
    /**
     * If true, prefer Uint8Arrays to
     * be generated instead of node Buffers.  This might turn on some more
     * changes in the future, so forward-compatibility is not guaranteed yet.
     */
    preferWeb?: boolean;
    /**
     * The encoding of the input.
     * Ignored if input is a Buffer.
     */
    encoding?: BufferEncoding;
    /**
     * Should an error be thrown when no
     * data is in the input?
     */
    required?: boolean;
    /**
     * If true, emit extended
     * results, which will be an object with shape {@link ExtendedResults }.
     * The value will already have been null-checked.
     */
    extendedResults?: boolean;
    /**
     * If true, error is
     * thrown if a map has duplicate keys.
     */
    preventDuplicateKeys?: boolean;
};
type ExtendedResults = {
    /**
     * The value that was found.
     */
    value: any;
    /**
     * The number of bytes of the original input that
     * were read.
     */
    length: number;
    /**
     * The bytes of the original input that were used
     * to produce the value.
     */
    bytes: Buffer;
    /**
     * The bytes that were left over from the original
     * input.  This property only exists if {@link Decoder.decodeFirst } or
     * {@link Decoder.decodeFirstSync } was called.
     */
    unused?: Buffer;
};
type decodeCallback = (error?: Error, value?: any) => void;
declare const NOT_FOUND: unique symbol;
import { Buffer } from "buffer";
import stream = require("stream");
