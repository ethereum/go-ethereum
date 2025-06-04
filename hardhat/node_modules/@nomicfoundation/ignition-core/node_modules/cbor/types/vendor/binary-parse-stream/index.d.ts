export = BinaryParseStream;
/**
 * BinaryParseStream is a TransformStream that consumes buffers and outputs
 * objects on the other end.  It expects your subclass to implement a `_parse`
 * method that is a generator.  When your generator yields a number, it'll be
 * fed a buffer of that length from the input.  When your generator returns,
 * the return value will be pushed to the output side.
 *
 * @extends stream.Transform
 */
declare class BinaryParseStream extends stream.Transform {
    /**
     * Creates an instance of BinaryParseStream.
     *
     * @memberof BinaryParseStream
     * @param {stream.TransformOptions} options Stream options.
     */
    constructor(options: stream.TransformOptions);
    bs: NoFilter;
    __fresh: boolean;
    __needed: any;
    /**
     * Subclasses must override this to set their parsing behavior.  Yield a
     * number to receive a Buffer of that many bytes.
     *
     * @abstract
     * @returns {Generator<number, any, Buffer>}
     */
    _parse(): Generator<number, any, Buffer>;
    __restart(): void;
    __parser: Generator<number, any, Buffer>;
}
import stream = require("stream");
import NoFilter = require("nofilter");
