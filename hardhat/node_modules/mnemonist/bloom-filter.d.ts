/**
 * Mnemonist BloomFilter Typings
 * ==============================
 */
type BloomFilterOptions = {
  capacity: number;
  errorRate?: number;
}

export default class BloomFilter {

  // Members
  capacity: number;
  errorRate: number;
  hashFunctions: number;

  // Constructor
  constructor(capacity: number);
  constructor(options: BloomFilterOptions);

  // Methods
  clear(): void;
  add(string: string): this;
  test(string: string): boolean;
  toJSON(): Uint8Array;

  // Statics
  from(iterable: Iterable<string>, options?: number | BloomFilterOptions): BloomFilter;
}
