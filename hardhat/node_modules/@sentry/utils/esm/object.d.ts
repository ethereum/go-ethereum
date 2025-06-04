import { Memo } from './memo';
/**
 * Wrap a given object method with a higher-order function
 *
 * @param source An object that contains a method to be wrapped.
 * @param name A name of method to be wrapped.
 * @param replacementFactory A function that should be used to wrap a given method, returning the wrapped method which
 * will be substituted in for `source[name]`.
 * @returns void
 */
export declare function fill(source: {
    [key: string]: any;
}, name: string, replacementFactory: (...args: any[]) => any): void;
/**
 * Encodes given object into url-friendly format
 *
 * @param object An object that contains serializable values
 * @returns string Encoded
 */
export declare function urlEncode(object: {
    [key: string]: any;
}): string;
/** JSDoc */
export declare function normalizeToSize<T>(object: {
    [key: string]: any;
}, depth?: number, maxSize?: number): T;
/**
 * Walks an object to perform a normalization on it
 *
 * @param key of object that's walked in current iteration
 * @param value object to be walked
 * @param depth Optional number indicating how deep should walking be performed
 * @param memo Optional Memo class handling decycling
 */
export declare function walk(key: string, value: any, depth?: number, memo?: Memo): any;
/**
 * normalize()
 *
 * - Creates a copy to prevent original input mutation
 * - Skip non-enumerablers
 * - Calls `toJSON` if implemented
 * - Removes circular references
 * - Translates non-serializeable values (undefined/NaN/Functions) to serializable format
 * - Translates known global objects/Classes to a string representations
 * - Takes care of Error objects serialization
 * - Optionally limit depth of final output
 */
export declare function normalize(input: any, depth?: number): any;
/**
 * Given any captured exception, extract its keys and create a sorted
 * and truncated list that will be used inside the event message.
 * eg. `Non-error exception captured with keys: foo, bar, baz`
 */
export declare function extractExceptionKeysForMessage(exception: any, maxLength?: number): string;
/**
 * Given any object, return the new object with removed keys that value was `undefined`.
 * Works recursively on objects and arrays.
 */
export declare function dropUndefinedKeys<T>(val: T): T;
//# sourceMappingURL=object.d.ts.map