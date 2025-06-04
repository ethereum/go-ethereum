/**
 * Mnemonist FixedStack Typings
 * =============================
 */
import {IArrayLikeConstructor} from './utils/types';

export default class FixedStack<T> implements Iterable<T> {

  // Members
  capacity: number;
  size: number;

  // Constructor
  constructor(ArrayClass: IArrayLikeConstructor, capacity: number);

  // Methods
  clear(): void;
  push(item: T): number;
  pop(): T | undefined;
  peek(): T | undefined;
  forEach(callback: (item: T, index: number, stack: this) => void, scope?: any): void;
  toArray(): Iterable<T>;
  values(): IterableIterator<T>;
  entries(): IterableIterator<[number, T]>;
  [Symbol.iterator](): IterableIterator<T>;
  toString(): string;
  toJSON(): Iterable<T>;
  inspect(): any;

  // Statics
  static from<I>(
    iterable: Iterable<I> | {[key: string] : I},
    ArrayClass: IArrayLikeConstructor,
    capacity?: number
  ): FixedStack<I>;
}
