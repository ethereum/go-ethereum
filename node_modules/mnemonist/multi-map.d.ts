/**
 * Mnemonist MultiMap Typings
 * ===========================
 */

interface MultiMap<K, V, C extends V[] | Set<V> = V[]> extends Iterable<[K, V]> {

  // Members
  dimension: number;
  size: number;

  // Methods
  clear(): void;
  set(key: K, value: V): this;
  delete(key: K): boolean;
  remove(key: K, value: V): boolean;
  has(key: K): boolean;
  get(key: K): C | undefined;
  multiplicity(key: K): number;
  forEach(callback: (value: V, key: K, map: this) => void, scope?: any): void;
  forEachAssociation(callback: (value: C, key: K, map: this) => void, scope?: any): void;
  keys(): IterableIterator<K>;
  values(): IterableIterator<V>;
  entries(): IterableIterator<[K, V]>;
  containers(): IterableIterator<C>;
  associations(): IterableIterator<[K, C]>;
  [Symbol.iterator](): IterableIterator<[K, V]>;
  inspect(): any;
  toJSON(): any;
}

interface MultiMapConstructor {
  new <K, V>(container: SetConstructor): MultiMap<K, V, Set<V>>;
  new <K, V>(container?: ArrayConstructor): MultiMap<K, V, V[]>;

  from<K, V>(
    iterable: Iterable<[K, V]> | {[key: string]: V},
    Container: SetConstructor
  ): MultiMap<K, V, Set<V>>;
  from<K, V>(
    iterable: Iterable<[K, V]> | {[key: string]: V},
    Container?: ArrayConstructor
  ): MultiMap<K, V, V[]>;
}

declare const MultiMap: MultiMapConstructor;
export default MultiMap;