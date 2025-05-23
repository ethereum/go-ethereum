"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.Arrays = void 0;
var Arrays;
(function (Arrays) {
    /**
     * Searches the specified array of numbers for the specified value using the binary search algorithm. The array must
     * be sorted prior to making this call. If it is not sorted, the results are unspecified. If the array contains
     * multiple elements with the specified value, there is no guarantee which one will be found.
     *
     * @returns index of the search key, if it is contained in the array; otherwise, (-(insertion point) - 1). The
     * insertion point is defined as the point at which the key would be inserted into the array: the index of the first
     * element greater than the key, or array.length if all elements in the array are less than the specified key. Note
     * that this guarantees that the return value will be >= 0 if and only if the key is found.
     */
    function binarySearch(array, key, fromIndex, toIndex) {
        return binarySearch0(array, fromIndex !== undefined ? fromIndex : 0, toIndex !== undefined ? toIndex : array.length, key);
    }
    Arrays.binarySearch = binarySearch;
    function binarySearch0(array, fromIndex, toIndex, key) {
        let low = fromIndex;
        let high = toIndex - 1;
        while (low <= high) {
            let mid = (low + high) >>> 1;
            let midVal = array[mid];
            if (midVal < key) {
                low = mid + 1;
            }
            else if (midVal > key) {
                high = mid - 1;
            }
            else {
                // key found
                return mid;
            }
        }
        // key not found.
        return -(low + 1);
    }
    function toString(array) {
        let result = "[";
        let first = true;
        for (let element of array) {
            if (first) {
                first = false;
            }
            else {
                result += ", ";
            }
            if (element === null) {
                result += "null";
            }
            else if (element === undefined) {
                result += "undefined";
            }
            else {
                result += element;
            }
        }
        result += "]";
        return result;
    }
    Arrays.toString = toString;
})(Arrays = exports.Arrays || (exports.Arrays = {}));
//# sourceMappingURL=Arrays.js.map