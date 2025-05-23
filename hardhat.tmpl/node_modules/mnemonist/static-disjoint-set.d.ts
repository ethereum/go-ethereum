/**
 * Mnemonist StaticDisjointSet Typings
 * ====================================
 */
import {ArrayLike} from './utils/types';

export default class StaticDisjointSet {
  
  // Members
  dimension: number;
  size: number;

  // Constructor
  constructor(size: number);

  // Methods
  find(x: number): number;
  union(x: number, y: number): this;
  connected(x: number, y: number): boolean;
  mapping(): ArrayLike;
  compile(): Array<Array<number>>;
  inspect(): any;
}