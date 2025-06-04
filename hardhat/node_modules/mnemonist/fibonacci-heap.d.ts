/**
 * Mnemonist FibonacciHeap Typings
 * ================================
 */
type FibonacciHeapComparator<T> = (a: T, b: T) => number;

export default class FibonacciHeap<T> {

  // Members
  size: number;

  // Constructor
  constructor(comparator?: FibonacciHeapComparator<T>);

  // Methods
  clear(): void;
  push(item: T): number;
  peek(): T | undefined;
  pop(): T | undefined;
  inspect(): any;

  // Statics
  static from<I>(
    iterable: Iterable<I> | {[key: string] : I},
    comparator?: FibonacciHeapComparator<I>
  ): FibonacciHeap<I>;
}

export class MinFibonacciHeap<T> {

  // Members
  size: number;

  // Constructor
  constructor(comparator?: FibonacciHeapComparator<T>);

  // Methods
  clear(): void;
  push(item: T): number;
  peek(): T | undefined;
  pop(): T | undefined;
  inspect(): any;

  // Statics
  static from<I>(iterable: Iterable<I> | {[key: string] : I}): FibonacciHeap<I>;
}

export class MaxFibonacciHeap<T> {

  // Members
  size: number;

  // Constructor
  constructor(comparator?: FibonacciHeapComparator<T>);

  // Methods
  clear(): void;
  push(item: T): number;
  peek(): T | undefined;
  pop(): T | undefined;
  inspect(): any;

  // Statics
  static from<I>(iterable: Iterable<I> | {[key: string] : I}): FibonacciHeap<I>;
}
