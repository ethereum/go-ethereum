export = Diagnose;
/**
 * Output the diagnostic format from a stream of CBOR bytes.
 *
 * @extends stream.Transform
 */
declare class Diagnose extends stream.Transform {
    /**
     * Convenience function to return a string in diagnostic format.
     *
     * @param {BufferLike} input The CBOR bytes to format.
     * @param {DiagnoseOptions |diagnoseCallback|string} [options={}]
     *   Options, the callback, or the input encoding.
     * @param {diagnoseCallback} [cb] Callback.
     * @returns {Promise} If callback not specified.
     * @throws {TypeError} Input not provided.
     */
    static diagnose(input: BufferLike, options?: DiagnoseOptions | diagnoseCallback | string, cb?: diagnoseCallback): Promise<any>;
    /**
     * Creates an instance of Diagnose.
     *
     * @param {DiagnoseOptions} [options={}] Options for creation.
     */
    constructor(options?: DiagnoseOptions);
    float_bytes: number;
    separator: string;
    stream_errors: boolean;
    parser: Decoder;
    /**
     * @ignore
     */
    _on_error(er: any): void;
    /** @private */
    private _on_more;
    /** @private */
    private _fore;
    /** @private */
    private _on_value;
    /** @private */
    private _on_start;
    /** @private */
    private _on_stop;
    /** @private */
    private _on_data;
}
declare namespace Diagnose {
    export { BufferLike, DiagnoseOptions, diagnoseCallback };
}
import stream = require("stream");
import Decoder = require("./decoder");
/**
 * Things that can act as inputs, from which a NoFilter can be created.
 */
type BufferLike = string | Buffer | ArrayBuffer | Uint8Array | Uint8ClampedArray | DataView | stream.Readable;
type DiagnoseOptions = {
    /**
     * Output between detected objects.
     */
    separator?: string;
    /**
     * Put error info into the
     * output stream.
     */
    stream_errors?: boolean;
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
    tags?: object;
    /**
     * If true, prefer Uint8Arrays to
     * be generated instead of node Buffers.  This might turn on some more
     * changes in the future, so forward-compatibility is not guaranteed yet.
     */
    preferWeb?: boolean;
    /**
     * The encoding of input, ignored if
     * input is not string.
     */
    encoding?: BufferEncoding;
};
type diagnoseCallback = (error?: Error, value?: string) => void;
