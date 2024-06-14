/**
 * Mnemonist SparseMap Typings
 * ============================
 */
export default class SparseMap<V> implements Iterable<[number, V]> {

  // Members
  length: number;
  size: number;

  // Constructor
  constructor(length: number);

  // Methods
  clear(): void;
  has(key: number): boolean;
  get(key: number): V | undefined;
  set(key: number, value: V): this;
  delete(key: number): boolean;
  forEach(callback: (value: V, key: number, set: this) => void, scope?: any): void;
  keys(): IterableIterator<number>;
  values(): IterableIterator<V>;
  entries(): IterableIterator<[number, V]>;
  [Symbol.iterator](): IterableIterator<[number, V]>;
  inspect(): any;
}
