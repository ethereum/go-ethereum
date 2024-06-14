export = Simple;
/**
 * A CBOR Simple Value that does not map onto a known constant.
 */
declare class Simple {
    /**
     * Is the given object a Simple?
     *
     * @param {any} obj Object to test.
     * @returns {boolean} Is it Simple?
     */
    static isSimple(obj: any): boolean;
    /**
     * Decode from the CBOR additional information into a JavaScript value.
     * If the CBOR item has no parent, return a "safe" symbol instead of
     * `null` or `undefined`, so that the value can be passed through a
     * stream in object mode.
     *
     * @param {number} val The CBOR additional info to convert.
     * @param {boolean} [has_parent=true] Does the CBOR item have a parent?
     * @param {boolean} [parent_indefinite=false] Is the parent element
     *   indefinitely encoded?
     * @returns {(null|undefined|boolean|symbol|Simple)} The decoded value.
     * @throws {Error} Invalid BREAK.
     */
    static decode(val: number, has_parent?: boolean, parent_indefinite?: boolean): (null | undefined | boolean | symbol | Simple);
    /**
     * Creates an instance of Simple.
     *
     * @param {number} value The simple value's integer value.
     */
    constructor(value: number);
    value: number;
    /**
     * Debug string for simple value.
     *
     * @returns {string} Formated string of `simple(value)`.
     */
    toString(): string;
    /**
     * Push the simple value onto the CBOR stream.
     *
     * @param {object} gen The generator to push onto.
     * @returns {boolean} True on success.
     */
    encodeCBOR(gen: object): boolean;
}
