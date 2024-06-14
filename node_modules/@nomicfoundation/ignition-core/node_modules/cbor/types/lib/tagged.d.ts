export = Tagged;
/**
 * A CBOR tagged item, where the tag does not have semantics specified at the
 * moment, or those semantics threw an error during parsing. Typically this will
 * be an extension point you're not yet expecting.
 */
declare class Tagged {
    static set TAGS(arg: {
        [x: string]: TagFunction;
    });
    /**
     * The current set of supported tags.  May be modified by plugins.
     *
     * @type {TagMap}
     * @static
     */
    static get TAGS(): {
        [x: string]: TagFunction;
    };
    /**
     * Reset the supported tags to the original set, before any plugins modified
     * the list.
     */
    static reset(): void;
    /**
     * Creates an instance of Tagged.
     *
     * @param {number} tag The number of the tag.
     * @param {any} value The value inside the tag.
     * @param {Error} [err] The error that was thrown parsing the tag, or null.
     */
    constructor(tag: number, value: any, err?: Error);
    tag: number;
    value: any;
    err: Error;
    toJSON(): any;
    /**
     * Convert to a String.
     *
     * @returns {string} String of the form '1(2)'.
     */
    toString(): string;
    /**
     * Push the simple value onto the CBOR stream.
     *
     * @param {object} gen The generator to push onto.
     * @returns {boolean} True on success.
     */
    encodeCBOR(gen: object): boolean;
    /**
     * If we have a converter for this type, do the conversion.  Some converters
     * are built-in.  Additional ones can be passed in.  If you want to remove
     * a built-in converter, pass a converter in whose value is 'null' instead
     * of a function.
     *
     * @param {object} converters Keys in the object are a tag number, the value
     *   is a function that takes the decoded CBOR and returns a JavaScript value
     *   of the appropriate type.  Throw an exception in the function on errors.
     * @returns {any} The converted item.
     */
    convert(converters: object): any;
}
declare namespace Tagged {
    export { INTERNAL_JSON, TagFunction, TagMap };
}
/**
 * Convert a tagged value to a more interesting JavaScript type.  Errors
 * thrown in this function will be captured into the "err" property of the
 * original Tagged instance.
 */
type TagFunction = (value: any, tag: Tagged) => any;
declare const INTERNAL_JSON: unique symbol;
/**
 * A mapping from tag number to a tag decoding function.
 */
type TagMap = {
    [x: string]: TagFunction;
};
