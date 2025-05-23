/**
 * Mnemonist SymSpell Typings
 * ===========================
 */
type SymSpellVerbosity = 0 | 1 | 2;

type SymSpellOptions = {
  maxDistance?: number;
  verbosity?: SymSpellVerbosity
};

type SymSpellMatch = {
  term: string;
  distance: number;
  count: number;
}

export default class SymSpell {

  // Members
  size: number;

  // Constructor
  constructor(options?: SymSpellOptions);

  // Methods
  clear(): void;
  add(string: string): this;
  search(query: string): Array<SymSpellMatch>;

  // Statics
  static from(strings: Iterable<string> | {[key: string]: string}, options?: SymSpellOptions): SymSpell;
} 
