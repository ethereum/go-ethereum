/**
 * Mnemonist SparseQueueSet Typings
 * =================================
 */
export default class SparseQueueSet implements Iterable<number> {

  // Members
  capacity: number;
  start: number;
  size: number;

  // Constructor
  constructor(length: number);

  // Methods
  clear(): void;
  has(value: number): boolean;
  enqueue(value: number): this;
  dequeue(): number | undefined;
  forEach(callback: (value: number, key: number, set: this) => void, scope?: any): void;
  values(): IterableIterator<number>;
  [Symbol.iterator](): IterableIterator<number>;
  inspect(): any;
}
