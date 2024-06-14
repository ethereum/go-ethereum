/**
 * Mnemonist SuffixArray Typings
 * ==============================
 */
export default class SuffixArray {
  
  // Members
  array: Array<number>;
  length: number;
  string: string | Array<string>;

  // Constructor
  constructor(string: string | Array<string>);

  // Methods
  toString(): string;
  toJSON(): Array<number>;
  inspect(): any;
}

export class GeneralizedSuffixArray {

  // Members
  array: Array<number>;
  length: number;
  size: number;
  text: string | Array<string>;

  // Constructor
  constructor(strings: Array<string> | Array<Array<string>>);

  // Methods
  longestCommonSubsequence(): string | Array<string>;
  toString(): string;
  toJSON(): Array<number>;
  inspect(): any;
}