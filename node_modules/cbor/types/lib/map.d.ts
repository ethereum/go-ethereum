export = CborMap;
/**
 * Wrapper around a JavaScript Map object that allows the keys to be
 * any complex type.  The base Map object allows this, but will only
 * compare the keys by identity, not by value.  CborMap translates keys
 * to CBOR first (and base64's them to ensure by-value comparison).
 *
 * This is not a subclass of Object, because it would be tough to get
 * the semantics to be an exact match.
 *
 * @extends Map
 */
declare class CborMap extends Map<any, any> {
    /**
     * @ignore
     */
    static _encode(key: any): string;
    /**
     * @ignore
     */
    static _decode(key: any): any;
    /**
     * Creates an instance of CborMap.
     *
     * @param {Iterable<any>} [iterable] An Array or other iterable
     *   object whose elements are key-value pairs (arrays with two elements, e.g.
     *   <code>[[ 1, 'one' ],[ 2, 'two' ]]</code>). Each key-value pair is added
     *   to the new CborMap; null values are treated as undefined.
     */
    constructor(iterable?: Iterable<any>);
    /**
     * Push the simple value onto the CBOR stream.
     *
     * @param {object} gen The generator to push onto.
     * @returns {boolean} True on success.
     */
    encodeCBOR(gen: object): boolean;
}
