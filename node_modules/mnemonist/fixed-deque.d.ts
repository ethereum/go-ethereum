/**
 * Mnemonist FixedDeque Typings
 * =============================
 */
import {IArrayLikeConstructor} from './utils/types';

export default class FixedDeque<T> implements Iterable<T> {

  // Members
  capacity: number;
  size: number;

  // Constructor
  constructor(ArrayClass: IArrayLikeConstructor, capacity: number);

  // Methods
  clear(): void;
  push(item: T): number;
  unshift(item: T): number;
  pop(): T | undefined;
  shift(): T | undefined;
  peekFirst(): T | undefined;
  peekLast(): T | undefined;
  get(index: number): T | undefined;
  forEach(callback: (item: T, index: number, buffer: this) => void, scope?: any): void;
  toArray(): Iterable<T>;
  values(): IterableIterator<T>;
  entries(): IterableIterator<[number, T]>;
  [Symbol.iterator](): IterableIterator<T>;
  inspect(): any;

  // Statics
  static from<I>(iterable: Iterable<I> | {[key: string] : I}, ArrayClass: IArrayLikeConstructor, capacity?: number): FixedDeque<I>;
}
