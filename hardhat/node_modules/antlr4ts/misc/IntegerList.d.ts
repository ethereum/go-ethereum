/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { JavaCollection } from "./Stubs";
/**
 *
 * @author Sam Harwell
 */
export declare class IntegerList {
    private _data;
    private _size;
    constructor(arg?: number | IntegerList | Iterable<number>);
    add(value: number): void;
    addAll(list: number[] | IntegerList | JavaCollection<number>): void;
    get(index: number): number;
    contains(value: number): boolean;
    set(index: number, value: number): number;
    removeAt(index: number): number;
    removeRange(fromIndex: number, toIndex: number): void;
    get isEmpty(): boolean;
    get size(): number;
    trimToSize(): void;
    clear(): void;
    toArray(): number[];
    sort(): void;
    /**
     * Compares the specified object with this list for equality.  Returns
     * `true` if and only if the specified object is also an {@link IntegerList},
     * both lists have the same size, and all corresponding pairs of elements in
     * the two lists are equal.  In other words, two lists are defined to be
     * equal if they contain the same elements in the same order.
     *
     * This implementation first checks if the specified object is this
     * list. If so, it returns `true`; if not, it checks if the
     * specified object is an {@link IntegerList}. If not, it returns `false`;
     * if so, it checks the size of both lists. If the lists are not the same size,
     * it returns `false`; otherwise it iterates over both lists, comparing
     * corresponding pairs of elements.  If any comparison returns `false`,
     * this method returns `false`.
     *
     * @param o the object to be compared for equality with this list
     * @returns `true` if the specified object is equal to this list
     */
    equals(o: any): boolean;
    /**
     * Returns the hash code value for this list.
     *
     * This implementation uses exactly the code that is used to define the
     * list hash function in the documentation for the {@link List#hashCode}
     * method.
     *
     * @returns the hash code value for this list
     */
    hashCode(): number;
    /**
     * Returns a string representation of this list.
     */
    toString(): string;
    binarySearch(key: number, fromIndex?: number, toIndex?: number): number;
    private ensureCapacity;
    /** Convert the list to a UTF-16 encoded char array. If all values are less
     *  than the 0xFFFF 16-bit code point limit then this is just a char array
     *  of 16-bit char as usual. For values in the supplementary range, encode
     * them as two UTF-16 code units.
     */
    toCharArray(): Uint16Array;
    private charArraySize;
}
