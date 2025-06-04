/**
 * Mnemonist FuzzyMap Typings
 * ==========================
 */
type HashFunction<K> = (key: any) => K;
type HashFunctionsTuple<K> = [HashFunction<K>, HashFunction<K>];

export default class FuzzyMap<K, V> implements Iterable<V> {

  // Members
  size: number;

  // Constructor
  constructor(hashFunction: HashFunction<K>);
  constructor(hashFunctionsTuple: HashFunctionsTuple<K>);

  // Methods
  clear(): void;
  add(key: V): this;
  set(key: K, value: V): this;
  get(key: any): V | undefined;
  has(key: any): boolean;
  forEach(callback: (value: V, key: V) => void, scope?: this): void;
  values(): IterableIterator<V>;
  [Symbol.iterator](): IterableIterator<V>;
  inspect(): any;

  // Statics
  static from<I, J>(
    iterable: Iterable<[I, J]> | {[key: string]: J},
    hashFunction: HashFunction<I> | HashFunctionsTuple<I>,
  ): FuzzyMap<I, J>;
}