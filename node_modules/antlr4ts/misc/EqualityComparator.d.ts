/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
/**
 * This interface provides an abstract concept of object equality independent of
 * {@link Object#equals} (object equality) and the `==` operator
 * (reference equality). It can be used to provide algorithm-specific unordered
 * comparisons without requiring changes to the object itself.
 *
 * @author Sam Harwell
 */
export interface EqualityComparator<T> {
    /**
     * This method returns a hash code for the specified object.
     *
     * @param obj The object.
     * @returns The hash code for `obj`.
     */
    hashCode(obj: T): number;
    /**
     * This method tests if two objects are equal.
     *
     * @param a The first object to compare.
     * @param b The second object to compare.
     * @returns `true` if `a` equals `b`, otherwise `false`.
     */
    equals(a: T, b: T): boolean;
}
