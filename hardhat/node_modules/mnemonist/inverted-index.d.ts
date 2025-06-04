/**
 * Mnemonist InvertedIndex Typings
 * ================================
 */
type Tokenizer = (key: any) => Array<string>;
type TokenizersTuple = [Tokenizer, Tokenizer];

export default class InvertedIndex<D> implements Iterable<D> {

  // Members
  dimension: number;
  size: number;

  // Constructor
  constructor(tokenizer?: Tokenizer);
  constructor(tokenizers?: TokenizersTuple);

  // Methods
  clear(): void;
  add(document: D): this;
  get(query: any): Array<D>;
  forEach(callback: (document: D, index: number, invertedIndex: this) => void, scope?: any): void;
  documents(): IterableIterator<D>;
  tokens(): IterableIterator<string>;
  [Symbol.iterator](): IterableIterator<D>;
  inspect(): any;

  // Statics
  static from<I>(
    iterable: Iterable<I> | {[key: string] : I},
    tokenizer?: Tokenizer | TokenizersTuple
  ): InvertedIndex<I>;
}