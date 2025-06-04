/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { EqualityComparator } from "./EqualityComparator";
import { JavaCollection, JavaSet } from "./Stubs";
export declare class Array2DHashSet<T extends {
    toString(): string;
}> implements JavaSet<T> {
    protected comparator: EqualityComparator<T>;
    protected buckets: Array<T[] | undefined>;
    /** How many elements in set */
    protected n: number;
    protected threshold: number;
    constructor(comparator?: EqualityComparator<T>, initialCapacity?: number);
    constructor(set: Array2DHashSet<T>);
    /**
     * Add `o` to set if not there; return existing value if already
     * there. This method performs the same operation as {@link #add} aside from
     * the return value.
     */
    getOrAdd(o: T): T;
    protected getOrAddImpl(o: T): T;
    get(o: T): T | undefined;
    protected getBucket(o: T): number;
    hashCode(): number;
    equals(o: any): boolean;
    protected expand(): void;
    add(t: T): boolean;
    get size(): number;
    get isEmpty(): boolean;
    contains(o: any): boolean;
    containsFast(obj: T): boolean;
    [Symbol.iterator](): IterableIterator<T>;
    toArray(): T[];
    containsAll(collection: JavaCollection<T>): boolean;
    addAll(c: Iterable<T>): boolean;
    clear(): void;
    toString(): string;
    toTableString(): string;
    /**
     * Return `o` as an instance of the element type `T`. If
     * `o` is non-undefined but known to not be an instance of `T`, this
     * method returns `undefined`. The base implementation does not perform any
     * type checks; override this method to provide strong type checks for the
     * {@link #contains} and {@link #remove} methods to ensure the arguments to
     * the {@link EqualityComparator} for the set always have the expected
     * types.
     *
     * @param o the object to try and cast to the element type of the set
     * @returns `o` if it could be an instance of `T`, otherwise
     * `undefined`.
     */
    protected asElementType(o: any): T;
    /**
     * Return an array of `T[]` with length `capacity`.
     *
     * @param capacity the length of the array to return
     * @returns the newly constructed array
     */
    protected createBuckets(capacity: number): Array<T[] | undefined>;
}
