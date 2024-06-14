import type {Sequence} from './types';

type NextFunction<V> = () => IteratorResult<V>;

export default class ObliteratorIterator<V> implements IterableIterator<V> {
  // Constructor
  constructor(next: NextFunction<V>);

  // Well-known methods
  next(): IteratorResult<V>;
  [Symbol.iterator](): IterableIterator<V>;

  // Static methods
  static of<T>(...args: T[]): ObliteratorIterator<T>;
  static empty<T>(): ObliteratorIterator<T>;
  static is(value: any): boolean;
  static fromSequence<T>(sequence: Sequence<T>): ObliteratorIterator<T>;
}
