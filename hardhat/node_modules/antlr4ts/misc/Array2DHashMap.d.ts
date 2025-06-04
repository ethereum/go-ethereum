/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { EqualityComparator } from "./EqualityComparator";
import { JavaMap } from "./Stubs";
export declare class Array2DHashMap<K, V> implements JavaMap<K, V> {
    private backingStore;
    constructor(keyComparer: EqualityComparator<K>);
    constructor(map: Array2DHashMap<K, V>);
    clear(): void;
    containsKey(key: K): boolean;
    get(key: K): V | undefined;
    get isEmpty(): boolean;
    put(key: K, value: V): V | undefined;
    putIfAbsent(key: K, value: V): V | undefined;
    get size(): number;
    hashCode(): number;
    equals(o: any): boolean;
}
