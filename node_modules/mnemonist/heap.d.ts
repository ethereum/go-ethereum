/**
 * Mnemonist Heap Typings
 * =======================
 */
type HeapComparator<T> = (a: T, b: T) => number;

export default class Heap<T> {

  // Members
  size: number;

  // Constructor
  constructor(comparator?: HeapComparator<T>);

  // Methods
  clear(): void;
  push(item: T): number;
  peek(): T | undefined;
  pop(): T | undefined;
  replace(item: T): T | undefined;
  pushpop(item: T): T | undefined;
  toArray(): Array<T>;
  consume(): Array<T>;
  inspect(): any;

  // Statics
  static from<I>(
    iterable: Iterable<I> | {[key: string] : I},
    comparator?: HeapComparator<I>
  ): Heap<I>;
}

export class MinHeap<T> {

  // Members
  size: number;

  // Constructor
  constructor(comparator?: HeapComparator<T>);

  // Methods
  clear(): void;
  push(item: T): number;
  peek(): T | undefined;
  pop(): T | undefined;
  replace(item: T): T | undefined;
  pushpop(item: T): T | undefined;
  toArray(): Array<T>;
  consume(): Array<T>;
  inspect(): any;
}

export class MaxHeap<T> {

  // Members
  size: number;

  // Constructor
  constructor(comparator?: HeapComparator<T>);

  // Methods
  clear(): void;
  push(item: T): number;
  peek(): T | undefined;
  pop(): T | undefined;
  replace(item: T): T | undefined;
  pushpop(item: T): T | undefined;
  toArray(): Array<T>;
  consume(): Array<T>;
  inspect(): any;
}

// Static helpers
export function push<T>(comparator: HeapComparator<T>, heap: Array<T>, item: T): void;
export function pop<T>(comparator: HeapComparator<T>, heap: Array<T>): T;
export function replace<T>(comparator: HeapComparator<T>, heap: Array<T>, item: T): T;
export function pushpop<T>(comparator: HeapComparator<T>, heap: Array<T>, item: T): T;
export function heapify<T>(comparator: HeapComparator<T>, array: Array<T>): void;
export function consume<T>(comparator: HeapComparator<T>, heap: Array<T>): Array<T>;

export function nsmallest<T>(comparator: HeapComparator<T>, n: number, values: Iterable<T>): Array<T>;
export function nsmallest<T>(n: number, values: Iterable<T>): Array<T>;
export function nlargest<T>(comparator: HeapComparator<T>, n: number, values: Iterable<T>): Array<T>;
export function nlargest<T>(n: number, values: Iterable<T>): Array<T>;
