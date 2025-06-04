/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Equatable } from "./Stubs";
/**
 *
 * @author Sam Harwell
 */
export declare namespace MurmurHash {
    /**
     * Initialize the hash using the specified `seed`.
     *
     * @param seed the seed (optional)
     * @returns the intermediate hash value
     */
    function initialize(seed?: number): number;
    /**
     * Update the intermediate hash value for the next input `value`.
     *
     * @param hash the intermediate hash value
     * @param value the value to add to the current hash
     * @returns the updated intermediate hash value
     */
    function update(hash: number, value: number | string | Equatable | null | undefined): number;
    /**
     * Apply the final computation steps to the intermediate value `hash`
     * to form the final result of the MurmurHash 3 hash function.
     *
     * @param hash the intermediate hash value
     * @param numberOfWords the number of integer values added to the hash
     * @returns the final hash result
     */
    function finish(hash: number, numberOfWords: number): number;
    /**
     * Utility function to compute the hash code of an array using the
     * MurmurHash algorithm.
     *
     * @param <T> the array element type
     * @param data the array data
     * @param seed the seed for the MurmurHash algorithm
     * @returns the hash code of the data
     */
    function hashCode<T extends number | string | Equatable>(data: Iterable<T>, seed?: number): number;
}
