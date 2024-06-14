/**
 * Mnemonist BitSet Typings
 * =========================
 */
export default class BitSet implements Iterable<number> {

  // Members
  length: number;
  size: number;

  // Constructor
  constructor(length: number);

  // Methods
  clear(): void;
  set(index: number, value?: boolean | number): void;
  reset(index: number, value: boolean | number): void;
  flip(index: number, value: boolean | number): void;
  get(index: number): number;
  test(index: number): boolean;
  rank(r: number): number;
  select(r: number): number;
  forEach(callback: (index: number, value: number, set: this) => void, scope?: any): void;
  values(): IterableIterator<number>;
  entries(): IterableIterator<[number, number]>;
  [Symbol.iterator](): IterableIterator<number>;
  inspect(): any;
  toJSON(): Array<number>;
}
