/**
 * Mnemonist BitVector Typings
 * ============================
 */
type BitVectorOptions = {
  initialLength?: number;
  initialCapacity?: number;
  policy?: (capacity: number) => number;
}

export default class BitVector implements Iterable<number> {

  // Members
  capacity: number;
  length: number;
  size: number;

  // Constructor
  constructor(length: number);
  constructor(options: BitVectorOptions);

  // Methods
  clear(): void;
  set(index: number, value?: boolean | number): this;
  reset(index: number, value: boolean | number): void;
  flip(index: number, value: boolean | number): void;
  reallocate(capacity: number): this;
  grow(capacity?: number): this;
  resize(length: number): this;
  push(value: boolean | number): number;
  pop(): number | undefined;
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
