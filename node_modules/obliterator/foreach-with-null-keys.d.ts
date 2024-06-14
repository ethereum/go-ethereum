import type {Sequence} from './types';

interface ForEachTrait<K, V> {
  forEach(callback: (value: V, key: K, self: this) => void): void;
}

interface PlainObject<T> {
  [key: string]: T;
}

export default function forEachWithNullKeys<V>(
  iterable: Set<V>,
  callback: (value: V, key: null) => void
): void;

export default function forEachWithNullKeys<K, V>(
  iterable: ForEachTrait<K, V>,
  callback: (value: V, key: K) => void
): void;

export default function forEachWithNullKeys<T>(
  iterable: Iterator<T> | Iterable<T> | Sequence<T>,
  callback: (item: T, key: null) => void
): void;

export default function forEachWithNullKeys<T>(
  object: PlainObject<T>,
  callback: (value: T, key: string) => void
): void;
