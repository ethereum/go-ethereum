/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
export interface Equatable {
    equals(other: any): boolean;
    hashCode(): number;
}
export interface Comparable<T> {
    compareTo(o: T): number;
}
export interface JavaCollection<E> extends Iterable<E>, Equatable {
    add(e: E): boolean;
    addAll(collection: Iterable<E>): boolean;
    clear(): void;
    contains(o: any): boolean;
    containsAll(collection: Iterable<any>): boolean;
    readonly isEmpty: boolean;
    readonly size: number;
    toArray(): E[];
}
export interface JavaSet<E> extends JavaCollection<E> {
}
export interface JavaMap<K, V> extends Equatable {
    clear(): void;
    containsKey(key: K): boolean;
    get(key: K): V | undefined;
    readonly isEmpty: boolean;
    put(key: K, value: V): V | undefined;
    readonly size: number;
}
