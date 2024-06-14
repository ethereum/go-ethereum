/**
 * Mnemonist TrieMap Typings
 * ==========================
 */
export default class TrieMap<K, V> implements Iterable<[K, V]> {

  // Members
  size: number;

  // Constructor
  constructor(Token?: new () => K);

  // Methods
  clear(): void;
  set(prefix: K, value: V): this;
  update(prefix: K, updateFunction: (oldValue: V | undefined) => V): this
  get(prefix: K): V;
  delete(prefix: K): boolean;
  has(prefix: K): boolean;
  find(prefix: K): Array<[K, V]>;
  values(): IterableIterator<V>;
  prefixes(): IterableIterator<K>;
  keys(): IterableIterator<K>;
  entries(): IterableIterator<[K, V]>;
  [Symbol.iterator](): IterableIterator<[K, V]>;
  inspect(): any;

  // Statics
  static from<I, J>(iterable: Iterable<[I, J]> | {[key: string]: J}): TrieMap<I, J>;
}
