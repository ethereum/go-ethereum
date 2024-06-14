/**
 * Mnemonist LRUMap Typings
 * =========================
 */
import {IArrayLikeConstructor} from './utils/types';

export default class LRUMap<K, V> implements Iterable<[K, V]> {

  // Members
  capacity: number;
  size: number;

  // Constructor
  constructor(capacity: number);
  constructor(KeyArrayClass: IArrayLikeConstructor, ValueArrayClass: IArrayLikeConstructor, capacity: number);

  // Methods
  clear(): void;
  set(key: K, value: V): this;
  setpop(key: K, value: V): {evicted: boolean, key: K, value: V};
  get(key: K): V | undefined;
  peek(key: K): V | undefined;
  has(key: K): boolean;
  forEach(callback: (value: V, key: K, cache: this) => void, scope?: any): void;
  keys(): IterableIterator<K>;
  values(): IterableIterator<V>;
  entries(): IterableIterator<[K, V]>;
  [Symbol.iterator](): IterableIterator<[K, V]>;
  inspect(): any;

  // Statics
  static from<I, J>(
    iterable: Iterable<[I, J]> | {[key: string]: J},
    KeyArrayClass: IArrayLikeConstructor,
    ValueArrayClass: IArrayLikeConstructor,
    capacity?: number
  ): LRUMap<I, J>;

  static from<I, J>(
    iterable: Iterable<[I, J]> | {[key: string]: J},
    capacity?: number
  ): LRUMap<I, J>;
}
