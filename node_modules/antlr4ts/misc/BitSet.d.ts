/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
export declare class BitSet implements Iterable<number> {
    private data;
    /**
     * Creates a new bit set. All bits are initially `false`.
     */
    constructor();
    /**
     * Creates a bit set whose initial size is large enough to explicitly represent bits with indices in the range `0`
     * through `nbits-1`. All bits are initially `false`.
     */
    constructor(nbits: number);
    /**
     * Creates a bit set from a iterable list of numbers (including another BitSet);
     */
    constructor(numbers: Iterable<number>);
    /**
     * Performs a logical **AND** of this target bit set with the argument bit set. This bit set is modified so that
     * each bit in it has the value `true` if and only if it both initially had the value `true` and the corresponding
     * bit in the bit set argument also had the value `true`.
     */
    and(set: BitSet): void;
    /**
     * Clears all of the bits in this `BitSet` whose corresponding bit is set in the specified `BitSet`.
     */
    andNot(set: BitSet): void;
    /**
     * Returns the number of bits set to `true` in this `BitSet`.
     */
    cardinality(): number;
    /**
     * Sets all of the bits in this `BitSet` to `false`.
     */
    clear(): void;
    /**
     * Sets the bit specified by the index to `false`.
     *
     * @param bitIndex the index of the bit to be cleared
     *
     * @throws RangeError if the specified index is negative
     */
    clear(bitIndex: number): void;
    /**
     * Sets the bits from the specified `fromIndex` (inclusive) to the specified `toIndex` (exclusive) to `false`.
     *
     * @param fromIndex index of the first bit to be cleared
     * @param toIndex index after the last bit to be cleared
     *
     * @throws RangeError if `fromIndex` is negative, or `toIndex` is negative, or `fromIndex` is larger than `toIndex`
     */
    clear(fromIndex: number, toIndex: number): void;
    /**
     * Sets the bit at the specified index to the complement of its current value.
     *
     * @param bitIndex the index of the bit to flip
     *
     * @throws RangeError if the specified index is negative
     */
    flip(bitIndex: number): void;
    /**
     * Sets each bit from the specified `fromIndex` (inclusive) to the specified `toIndex` (exclusive) to the complement
     * of its current value.
     *
     * @param fromIndex index of the first bit to flip
     * @param toIndex index after the last bit to flip
     *
     * @throws RangeError if `fromIndex` is negative, or `toIndex` is negative, or `fromIndex` is larger than `toIndex`
     */
    flip(fromIndex: number, toIndex: number): void;
    /**
     * Returns the value of the bit with the specified index. The value is `true` if the bit with the index `bitIndex`
     * is currently set in this `BitSet`; otherwise, the result is `false`.
     *
     * @param bitIndex the bit index
     *
     * @throws RangeError if the specified index is negative
     */
    get(bitIndex: number): boolean;
    /**
     * Returns a new `BitSet` composed of bits from this `BitSet` from `fromIndex` (inclusive) to `toIndex` (exclusive).
     *
     * @param fromIndex index of the first bit to include
     * @param toIndex index after the last bit to include
     *
     * @throws RangeError if `fromIndex` is negative, or `toIndex` is negative, or `fromIndex` is larger than `toIndex`
     */
    get(fromIndex: number, toIndex: number): BitSet;
    /**
     * Returns true if the specified `BitSet` has any bits set to `true` that are also set to `true` in this `BitSet`.
     *
     * @param set `BitSet` to intersect with
     */
    intersects(set: BitSet): boolean;
    /**
     * Returns true if this `BitSet` contains no bits that are set to `true`.
     */
    get isEmpty(): boolean;
    /**
     * Returns the "logical size" of this `BitSet`: the index of the highest set bit in the `BitSet` plus one. Returns
     * zero if the `BitSet` contains no set bits.
     */
    length(): number;
    /**
     * Returns the index of the first bit that is set to `false` that occurs on or after the specified starting index,
     * If no such bit exists then `-1` is returned.
     *
     * @param fromIndex the index to start checking from (inclusive)
     *
     * @throws RangeError if the specified index is negative
     */
    nextClearBit(fromIndex: number): number;
    /**
     * Returns the index of the first bit that is set to `true` that occurs on or after the specified starting index.
     * If no such bit exists then `-1` is returned.
     *
     * To iterate over the `true` bits in a `BitSet`, use the following loop:
     *
     * ```
     * for (let i = bs.nextSetBit(0); i >= 0; i = bs.nextSetBit(i + 1)) {
     *   // operate on index i here
     * }
     * ```
     *
     * @param fromIndex the index to start checking from (inclusive)
     *
     * @throws RangeError if the specified index is negative
     */
    nextSetBit(fromIndex: number): number;
    /**
     * Performs a logical **OR** of this bit set with the bit set argument. This bit set is modified so that a bit in it
     * has the value `true` if and only if it either already had the value `true` or the corresponding bit in the bit
     * set argument has the value `true`.
     */
    or(set: BitSet): void;
    /**
     * Returns the index of the nearest bit that is set to `false` that occurs on or before the specified starting
     * index. If no such bit exists, or if `-1` is given as the starting index, then `-1` is returned.
     *
     * @param fromIndex the index to start checking from (inclusive)
     *
     * @throws RangeError if the specified index is less than `-1`
     */
    previousClearBit(fromIndex: number): number;
    /**
     * Returns the index of the nearest bit that is set to `true` that occurs on or before the specified starting index.
     * If no such bit exists, or if `-1` is given as the starting index, then `-1` is returned.
     *
     * To iterate over the `true` bits in a `BitSet`, use the following loop:
     *
     * ```
     * for (let i = bs.length(); (i = bs.previousSetBit(i-1)) >= 0; ) {
     *   // operate on index i here
     * }
     * ```
     *
     * @param fromIndex the index to start checking from (inclusive)
     *
     * @throws RangeError if the specified index is less than `-1`
     */
    previousSetBit(fromIndex: number): number;
    /**
     * Sets the bit at the specified index to `true`.
     *
     * @param bitIndex a bit index
     *
     * @throws RangeError if the specified index is negative
     */
    set(bitIndex: number): void;
    /**
     * Sets the bit at the specified index to the specified value.
     *
     * @param bitIndex a bit index
     * @param value a boolean value to set
     *
     * @throws RangeError if the specified index is negative
     */
    set(bitIndex: number, value: boolean): void;
    /**
     * Sets the bits from the specified `fromIndex` (inclusive) to the specified `toIndex` (exclusive) to `true`.
     *
     * @param fromIndex index of the first bit to be set
     * @param toIndex index after the last bit to be set
     *
     * @throws RangeError if `fromIndex` is negative, or `toIndex` is negative, or `fromIndex` is larger than `toIndex`
     */
    set(fromIndex: number, toIndex: number): void;
    /**
     * Sets the bits from the specified `fromIndex` (inclusive) to the specified `toIndex` (exclusive) to the specified
     * value.
     *
     * @param fromIndex index of the first bit to be set
     * @param toIndex index after the last bit to be set
     * @param value value to set the selected bits to
     *
     * @throws RangeError if `fromIndex` is negative, or `toIndex` is negative, or `fromIndex` is larger than `toIndex`
     */
    set(fromIndex: number, toIndex: number, value: boolean): void;
    private _setBits;
    /**
     * Returns the number of bits of space actually in use by this `BitSet` to represent bit values. The maximum element
     * in the set is the size - 1st element.
     */
    get size(): number;
    /**
     * Returns a new byte array containing all the bits in this bit set.
     *
     * More precisely, if
     * `let bytes = s.toByteArray();`
     * then `bytes.length === (s.length()+7)/8` and `s.get(n) === ((bytes[n/8] & (1<<(n%8))) != 0)` for all
     * `n < 8 * bytes.length`.
     */
    /**
     * Returns a new integer array containing all the bits in this bit set.
     *
     * More precisely, if
     * `let integers = s.toIntegerArray();`
     * then `integers.length === (s.length()+31)/32` and `s.get(n) === ((integers[n/32] & (1<<(n%32))) != 0)` for all
     * `n < 32 * integers.length`.
     */
    hashCode(): number;
    /**
     * Compares this object against the specified object. The result is `true` if and only if the argument is not
     * `undefined` and is a `Bitset` object that has exactly the same set of bits set to `true` as this bit set. That
     * is, for every nonnegative index `k`,
     *
     * ```
     * ((BitSet)obj).get(k) == this.get(k)
     * ```
     *
     * must be true. The current sizes of the two bit sets are not compared.
     */
    equals(obj: any): boolean;
    /**
     * Returns a string representation of this bit set. For every index for which this `BitSet` contains a bit in the
     * set state, the decimal representation of that index is included in the result. Such indices are listed in order
     * from lowest to highest, separated by ", " (a comma and a space) and surrounded by braces, resulting in the usual
     * mathematical notation for a set of integers.
     *
     * Example:
     *
     *     BitSet drPepper = new BitSet();
     *
     * Now `drPepper.toString()` returns `"{}"`.
     *
     *     drPepper.set(2);
     *
     * Now `drPepper.toString()` returns `"{2}"`.
     *
     *     drPepper.set(4);
     *     drPepper.set(10);
     *
     * Now `drPepper.toString()` returns `"{2, 4, 10}"`.
     */
    toString(): string;
    /**
     * Performs a logical **XOR** of this bit set with the bit set argument. This bit set is modified so that a bit in
     * it has the value `true` if and only if one of the following statements holds:
     *
     * * The bit initially has the value `true`, and the corresponding bit in the argument has the value `false`.
     * * The bit initially has the value `false`, and the corresponding bit in the argument has the value `true`.
     */
    xor(set: BitSet): void;
    clone(): BitSet;
    [Symbol.iterator](): IterableIterator<number>;
}
