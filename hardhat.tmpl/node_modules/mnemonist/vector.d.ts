/**
 * Mnemonist Vector Typings
 * =========================
 */
import {IArrayLikeConstructor} from './utils/types';

type VectorOptions = {
  initialLength?: number;
  initialCapacity?: number;
  policy?: (capacity: number) => number;
}

export default class Vector implements Iterable<number> {

  // Members
  capacity: number;
  length: number;
  size: number;

  // Constructor
  constructor(ArrayClass: IArrayLikeConstructor, length: number | VectorOptions);

  // Methods
  clear(): void;
  set(index: number, value: number): this;
  reallocate(capacity: number): this;
  grow(capacity?: number): this;
  resize(length: number): this;
  push(value: number): number;
  pop(): number | undefined;
  get(index: number): number;
  forEach(callback: (index: number, value: number, set: this) => void, scope?: any): void;
  values(): IterableIterator<number>;
  entries(): IterableIterator<[number, number]>;
  [Symbol.iterator](): IterableIterator<number>;
  inspect(): any;
  toJSON(): any;

  // Statics
  static from<I>(iterable: Iterable<I> | {[key: string] : I}, ArrayClass: IArrayLikeConstructor, capacity?: number): Vector;
}

declare class TypedVector implements Iterable<number> {

  // Members
  capacity: number;
  length: number;
  size: number;

  // Constructor
  constructor(length: number | VectorOptions);

  // Methods
  clear(): void;
  set(index: number, value: number): this;
  reallocate(capacity: number): this;
  grow(capacity?: number): this;
  resize(length: number): this;
  push(value: number): number;
  pop(): number | undefined;
  get(index: number): number;
  forEach(callback: (index: number, value: number, set: this) => void, scope?: any): void;
  values(): IterableIterator<number>;
  entries(): IterableIterator<[number, number]>;
  [Symbol.iterator](): IterableIterator<number>;
  inspect(): any;
  toJSON(): any;

  // Statics
  static from<I>(iterable: Iterable<I> | {[key: string] : I}, capacity?: number): TypedVector;
}

export class Int8Vector extends TypedVector {}
export class Uint8Vector extends TypedVector {}
export class Uint8ClampedVector extends TypedVector {}
export class Int16Vector extends TypedVector {}
export class Uint16Vector extends TypedVector {}
export class Int32Vector extends TypedVector {}
export class Uint32Vector extends TypedVector {}
export class Float32Vector extends TypedVector {}
export class Float64Array extends TypedVector {}
