/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { IntegerList } from "./IntegerList";
import { Interval } from "./Interval";
import { IntSet } from "./IntSet";
import { Vocabulary } from "../Vocabulary";
/**
 * This class implements the {@link IntSet} backed by a sorted array of
 * non-overlapping intervals. It is particularly efficient for representing
 * large collections of numbers, where the majority of elements appear as part
 * of a sequential range of numbers that are all part of the set. For example,
 * the set { 1, 2, 3, 4, 7, 8 } may be represented as { [1, 4], [7, 8] }.
 *
 * This class is able to represent sets containing any combination of values in
 * the range {@link Integer#MIN_VALUE} to {@link Integer#MAX_VALUE}
 * (inclusive).
 */
export declare class IntervalSet implements IntSet {
    private static _COMPLETE_CHAR_SET;
    static get COMPLETE_CHAR_SET(): IntervalSet;
    private static _EMPTY_SET;
    static get EMPTY_SET(): IntervalSet;
    /** The list of sorted, disjoint intervals. */
    private _intervals;
    private readonly;
    constructor(intervals?: Interval[]);
    /**
     * Create a set with all ints within range [a..b] (inclusive). If b is omitted, the set contains the single element
     * a.
     */
    static of(a: number, b?: number): IntervalSet;
    clear(): void;
    /** Add interval; i.e., add all integers from a to b to set.
     *  If b&lt;a, do nothing.
     *  Keep list in sorted order (by left range value).
     *  If overlap, combine ranges.  For example,
     *  If this is {1..5, 10..20}, adding 6..7 yields
     *  {1..5, 6..7, 10..20}.  Adding 4..8 yields {1..8, 10..20}.
     */
    add(a: number, b?: number): void;
    protected addRange(addition: Interval): void;
    /** combine all sets in the array returned the or'd value */
    static or(sets: IntervalSet[]): IntervalSet;
    addAll(set: IntSet): IntervalSet;
    complementRange(minElement: number, maxElement: number): IntervalSet;
    /** {@inheritDoc} */
    complement(vocabulary: IntSet): IntervalSet;
    subtract(a: IntSet): IntervalSet;
    /**
     * Compute the set difference between two interval sets. The specific
     * operation is `left - right`.
     */
    static subtract(left: IntervalSet, right: IntervalSet): IntervalSet;
    or(a: IntSet): IntervalSet;
    /** {@inheritDoc} */
    and(other: IntSet): IntervalSet;
    /** {@inheritDoc} */
    contains(el: number): boolean;
    /** {@inheritDoc} */
    get isNil(): boolean;
    /**
     * Returns the maximum value contained in the set if not isNil.
     *
     * @return the maximum value contained in the set.
     * @throws RangeError if set is empty
     */
    get maxElement(): number;
    /**
     * Returns the minimum value contained in the set if not isNil.
     *
     * @return the minimum value contained in the set.
     * @throws RangeError if set is empty
     */
    get minElement(): number;
    /** Return a list of Interval objects. */
    get intervals(): Interval[];
    hashCode(): number;
    /** Are two IntervalSets equal?  Because all intervals are sorted
     *  and disjoint, equals is a simple linear walk over both lists
     *  to make sure they are the same.  Interval.equals() is used
     *  by the List.equals() method to check the ranges.
     */
    equals(o: any): boolean;
    toString(elemAreChar?: boolean): string;
    toStringVocabulary(vocabulary: Vocabulary): string;
    protected elementName(vocabulary: Vocabulary, a: number): string;
    get size(): number;
    toIntegerList(): IntegerList;
    toSet(): Set<number>;
    toArray(): number[];
    remove(el: number): void;
    get isReadonly(): boolean;
    setReadonly(readonly: boolean): void;
}
