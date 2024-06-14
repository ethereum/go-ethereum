/**
 * Mnemonist Queue Typings
 * ========================
 */
export default class Queue<T> implements Iterable<T> {

  // Members
  size: number;

  // Methods
  clear(): void;
  enqueue(item: T): number;
  dequeue(): T | undefined;
  peek(): T | undefined;
  forEach(callback: (item: T, index: number, queue: this) => void, scope?: any): void;
  toArray(): Array<T>;
  values(): IterableIterator<T>;
  entries(): IterableIterator<[number, T]>;
  [Symbol.iterator](): IterableIterator<T>;
  toString(): string;
  toJSON(): Array<T>;
  inspect(): any;

  // Statics
  static from<I>(iterable: Iterable<I> | {[key: string] : I}): Queue<I>;
  static of<I>(...items: Array<I>): Queue<I>;
}
