/**
 * Mnemonist FuzzyMultiMap Typings
 * ================================
 */
type HashFunction<K> = (key: any) => K;
type HashFunctionsTuple<K> = [HashFunction<K>, HashFunction<K>];
type FuzzyMultiMapContainer = ArrayConstructor | SetConstructor;

export default class FuzzyMultiMap<K, V> implements Iterable<V> {

  // Members
  dimension: number;
  size: number;

  // Constructor
  constructor(hashFunction: HashFunction<K>, Container?: FuzzyMultiMapContainer);
  constructor(hashFunctions: HashFunctionsTuple<K>, Container?: FuzzyMultiMapContainer);

  // Methods
  clear(): void;
  add(value: V): this;
  set(key: K, value: V): this;
  get(key: any): Array<V> | Set<V> | undefined;
  has(key: any): boolean;
  forEach(callback: (value: V, key: V) => void, scope?: any): void;
  values(): IterableIterator<V>;
  [Symbol.iterator](): IterableIterator<V>;
  inspect(): any;

  // Statics
  static from<I, J>(
    iterable: Iterable<[I, J]> | {[key: string]: J},
    hashFunction: HashFunction<I> | HashFunctionsTuple<I>,
    Container?: FuzzyMultiMapContainer
  ): FuzzyMultiMap<I, J>;
}
