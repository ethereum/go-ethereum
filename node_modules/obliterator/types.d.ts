export interface Sequence<T> {
  length: number;
  slice(from: number, to?: number): Sequence<T>;
  [index: number]: T;
}

export interface Mapping<K, V> {
  has(key: K): boolean;
  get(key: K): V | undefined;
  forEach(callback: (value: V, key: K) => void): void;
}

export type AnyMapping<K, V> = K extends keyof any
  ? Record<K, V> | Mapping<K, V>
  : Mapping<K, V>;

export type IntoInterator<T> = Iterable<T> | Iterator<T> | Sequence<T>;
export type IntoEntriesIterator<K, V> = IntoInterator<[K, V]>;
