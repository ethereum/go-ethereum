/**
 * Mnemonist LinkedList Typings
 * =============================
 */
export default class LinkedList<T> implements Iterable<T> {

  // Members
  size: number;

  // Methods
  clear(): void;
  first(): T | undefined;
  last(): T | undefined;
  peek(): T | undefined;
  push(value: T): number;
  shift(): T | undefined;
  unshift(value: T): number;
  forEach(callback: (value: T, index: number, list: this) => void, scope?: any): void;
  toArray(): Array<T>;
  values(): IterableIterator<T>;
  entries(): IterableIterator<[number, T]>;
  [Symbol.iterator](): IterableIterator<T>;
  toString(): string;
  toJSON(): Array<T>;
  inspect(): any;

  // Statics
  static from<I>(iterable: Iterable<I> | {[key: string] : I}): LinkedList<I>;
}