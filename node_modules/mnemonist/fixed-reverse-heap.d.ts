/**
 * Mnemonist FixedReverseHeap Typings
 * ===================================
 */
import {IArrayLikeConstructor} from './utils/types';

type HeapComparator<T> = (a: T, b: T) => number;

export default class FixedReverseHeap<T> {

  // Members
  capacity: number;
  size: number;

  // Constructor
  constructor(ArrayClass: IArrayLikeConstructor, comparator: HeapComparator<T>, capacity: number);
  constructor(ArrayClass: IArrayLikeConstructor, capacity: number);

  // Methods
  clear(): void;
  push(item: T): number;
  consume(): Iterable<T>;
  toArray(): Iterable<T>;
  inspect(): any;
}
