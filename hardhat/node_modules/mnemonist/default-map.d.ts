/**
 * Mnemonist DefaultMap Typings
 * =============================
 */
export default class DefaultMap<K, V> implements Iterable<[K, V]> {

  // Members
  size: number;

  // Constructor
  constructor(factory: (key: K, index: number) => V);

  // Methods
  clear(): void;
  set(key: K, value: V): this;
  delete(key: K): boolean;
  has(key: K): boolean;
  get(key: K): V;
  peek(key: K): V | undefined;
  forEach(callback: (value: V, key: K, map: this) => void, scope?: any): void;
  keys(): IterableIterator<K>;
  values(): IterableIterator<V>;
  entries(): IterableIterator<[K, V]>;
  [Symbol.iterator](): IterableIterator<[K, V]>;
  inspect(): any;

  // Statics
  static autoIncrement(): number;
}
