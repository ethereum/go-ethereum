/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
export declare class MultiMap<K, V> extends Map<K, V[]> {
    constructor();
    map(key: K, value: V): void;
    getPairs(): Array<[K, V]>;
}
