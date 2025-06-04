import { HKT } from './HKT';
/**
 * @since 1.0.0
 */
export declare const identity: <A>(a: A) => A;
/**
 * @since 1.0.0
 */
export declare const unsafeCoerce: <A, B>(a: A) => B;
/**
 * Thunk type
 */
export declare type Lazy<A> = () => A;
/**
 * @example
 * import { FunctionN } from 'fp-ts/lib/function'
 *
 * export const sum: FunctionN<[number, number], number> = (a, b) => a + b
 *
 * @since 1.16.0
 */
export declare type FunctionN<A extends Array<unknown>, B> = (...args: A) => B;
/** @deprecated */
export declare type Function1<A, B> = (a: A) => B;
/** @deprecated */
export declare type Function2<A, B, C> = (a: A, b: B) => C;
/** @deprecated */
export declare type Function3<A, B, C, D> = (a: A, b: B, c: C) => D;
/** @deprecated */
export declare type Function4<A, B, C, D, E> = (a: A, b: B, c: C, d: D) => E;
/** @deprecated */
export declare type Function5<A, B, C, D, E, F> = (a: A, b: B, c: C, d: D, e: E) => F;
/** @deprecated */
export declare type Function6<A, B, C, D, E, F, G> = (a: A, b: B, c: C, d: D, e: E, f: F) => G;
/** @deprecated */
export declare type Function7<A, B, C, D, E, F, G, H> = (a: A, b: B, c: C, d: D, e: E, f: F, g: G) => H;
/** @deprecated */
export declare type Function8<A, B, C, D, E, F, G, H, I> = (a: A, b: B, c: C, d: D, e: E, f: F, g: G, h: H) => I;
/** @deprecated */
export declare type Function9<A, B, C, D, E, F, G, H, I, J> = (a: A, b: B, c: C, d: D, e: E, f: F, g: G, h: H, i: I) => J;
/** @deprecated */
export declare type Curried2<A, B, C> = (a: A) => (b: B) => C;
/** @deprecated */
export declare type Curried3<A, B, C, D> = (a: A) => (b: B) => (c: C) => D;
/** @deprecated */
export declare type Curried4<A, B, C, D, E> = (a: A) => (b: B) => (c: C) => (d: D) => E;
/** @deprecated */
export declare type Curried5<A, B, C, D, E, F> = (a: A) => (b: B) => (c: C) => (d: D) => (e: E) => F;
/** @deprecated */
export declare type Curried6<A, B, C, D, E, F, G> = (a: A) => (b: B) => (c: C) => (d: D) => (e: E) => (f: F) => G;
/** @deprecated */
export declare type Curried7<A, B, C, D, E, F, G, H> = (a: A) => (b: B) => (c: C) => (d: D) => (e: E) => (f: F) => (g: G) => H;
/** @deprecated */
export declare type Curried8<A, B, C, D, E, F, G, H, I> = (a: A) => (b: B) => (c: C) => (d: D) => (e: E) => (f: F) => (g: G) => (h: H) => I;
/** @deprecated */
export declare type Curried9<A, B, C, D, E, F, G, H, I, J> = (a: A) => (b: B) => (c: C) => (d: D) => (e: E) => (f: F) => (g: G) => (h: H) => (i: I) => J;
export declare type Predicate<A> = (a: A) => boolean;
export declare type Refinement<A, B extends A> = (a: A) => a is B;
/**
 * @since 1.0.0
 */
export declare const not: <A>(predicate: Predicate<A>) => Predicate<A>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function or<A, B1 extends A, B2 extends A>(p1: Refinement<A, B1>, p2: Refinement<A, B2>): Refinement<A, B1 | B2>;
export declare function or<A>(p1: Predicate<A>, p2: Predicate<A>): Predicate<A>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare const and: <A>(p1: Predicate<A>, p2: Predicate<A>) => Predicate<A>;
export declare type Endomorphism<A> = (a: A) => A;
/** @deprecated */
export declare type BinaryOperation<A, B> = (a1: A, a2: A) => B;
/** @deprecated */
export declare type Kleisli<F, A, B> = (a: A) => HKT<F, B>;
/** @deprecated */
export declare type Cokleisli<F, A, B> = (fa: HKT<F, A>) => B;
/**
 * @since 1.0.0
 */
export declare const constant: <A>(a: A) => Lazy<A>;
/**
 * A thunk that returns always `true`
 *
 * @since 1.0.0
 */
export declare const constTrue: () => boolean;
/**
 * A thunk that returns always `false`
 *
 * @since 1.0.0
 */
export declare const constFalse: () => boolean;
/**
 * A thunk that returns always `null`
 *
 * @since 1.0.0
 */
export declare const constNull: () => null;
/**
 * A thunk that returns always `undefined`
 *
 * @since 1.0.0
 */
export declare const constUndefined: () => undefined;
/**
 * A thunk that returns always `void`
 *
 * @since 1.14.0
 */
export declare const constVoid: () => void;
/**
 * Flips the order of the arguments to a function of two arguments.
 *
 * @since 1.0.0
 */
export declare const flip: <A, B, C>(f: Curried2<A, B, C>) => Curried2<B, A, C>;
/**
 * The `on` function is used to change the domain of a binary operator.
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const on: <B, C>(op: BinaryOperation<B, C>) => <A>(f: (a: A) => B) => BinaryOperation<A, C>;
/**
 * Use `flow`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function compose<A, B, C>(bc: (b: B) => C, ab: (a: A) => B): (a: A) => C;
/** @deprecated */
export declare function compose<A, B, C, D>(cd: (c: C) => D, bc: (b: B) => C, ab: (a: A) => B): (a: A) => D;
/** @deprecated */
export declare function compose<A, B, C, D, E>(de: (d: D) => E, cd: (c: C) => D, bc: (b: B) => C, ab: (a: A) => B): (a: A) => E;
/** @deprecated */
export declare function compose<A, B, C, D, E, F>(ef: (e: E) => F, de: (d: D) => E, cd: (c: C) => D, bc: (b: B) => C, ab: (a: A) => B): (a: A) => F;
/** @deprecated */
export declare function compose<A, B, C, D, E, F, G>(fg: (f: F) => G, ef: (e: E) => F, de: (d: D) => E, cd: (c: C) => D, bc: (b: B) => C, ab: (a: A) => B): (a: A) => G;
/** @deprecated */
export declare function compose<A, B, C, D, E, F, G, H>(gh: (g: G) => H, fg: (f: F) => G, ef: (e: E) => F, de: (d: D) => E, cd: (c: C) => D, bc: (b: B) => C, ab: (a: A) => B): (a: A) => H;
/** @deprecated */
export declare function compose<A, B, C, D, E, F, G, H, I>(hi: (h: H) => I, gh: (g: G) => H, fg: (f: F) => G, ef: (e: E) => F, de: (d: D) => E, cd: (c: C) => D, bc: (b: B) => C, ab: (a: A) => B): (a: A) => I;
/** @deprecated */
export declare function compose<A, B, C, D, E, F, G, H, I, J>(ij: (i: I) => J, hi: (h: H) => I, gh: (g: G) => H, fg: (f: F) => G, ef: (e: E) => F, de: (d: D) => E, cd: (c: C) => D, bc: (b: B) => C, ab: (a: A) => B): (a: A) => J;
/**
 * Use `flow`
 *
 * @since 1.0.0
 * @deprecated
 */
export declare function pipe<A, B, C>(ab: (a: A) => B, bc: (b: B) => C): (a: A) => C;
/** @deprecated */
export declare function pipe<A, B, C, D>(ab: (a: A) => B, bc: (b: B) => C, cd: (c: C) => D): (a: A) => D;
/** @deprecated */
export declare function pipe<A, B, C, D, E>(ab: (a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E): (a: A) => E;
/** @deprecated */
export declare function pipe<A, B, C, D, E, F>(ab: (a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F): (a: A) => F;
/** @deprecated */
export declare function pipe<A, B, C, D, E, F, G>(ab: (a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F, fg: (f: F) => G): (a: A) => G;
/** @deprecated */
export declare function pipe<A, B, C, D, E, F, G, H>(ab: (a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F, fg: (f: F) => G, gh: (g: G) => H): (a: A) => H;
/** @deprecated */
export declare function pipe<A, B, C, D, E, F, G, H, I>(ab: (a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F, fg: (f: F) => G, gh: (g: G) => H, hi: (h: H) => I): (a: A) => I;
/** @deprecated */
export declare function pipe<A, B, C, D, E, F, G, H, I, J>(ab: (a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F, fg: (f: F) => G, gh: (g: G) => H, hi: (h: H) => I, ij: (i: I) => J): (a: A) => J;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare const concat: <A>(x: A[], y: A[]) => A[];
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function curried(f: Function, n: number, acc: Array<any>): (this: any, x: any) => any;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare function curry<A, B, C>(f: Function2<A, B, C>): Curried2<A, B, C>;
/** @deprecated */
export declare function curry<A, B, C, D>(f: Function3<A, B, C, D>): Curried3<A, B, C, D>;
/** @deprecated */
export declare function curry<A, B, C, D, E>(f: Function4<A, B, C, D, E>): Curried4<A, B, C, D, E>;
/** @deprecated */
export declare function curry<A, B, C, D, E, F>(f: Function5<A, B, C, D, E, F>): Curried5<A, B, C, D, E, F>;
/** @deprecated */
export declare function curry<A, B, C, D, E, F, G>(f: Function6<A, B, C, D, E, F, G>): Curried6<A, B, C, D, E, F, G>;
/** @deprecated */
export declare function curry<A, B, C, D, E, F, G, H>(f: Function7<A, B, C, D, E, F, G, H>): Curried7<A, B, C, D, E, F, G, H>;
/** @deprecated */
export declare function curry<A, B, C, D, E, F, G, H, I>(f: Function8<A, B, C, D, E, F, G, H, I>): Curried8<A, B, C, D, E, F, G, H, I>;
/** @deprecated */
export declare function curry<A, B, C, D, E, F, G, H, I, J>(f: Function9<A, B, C, D, E, F, G, H, I, J>): Curried9<A, B, C, D, E, F, G, H, I, J>;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare const toString: (x: any) => string;
/**
 * @since 1.0.0
 */
export declare const tuple: <T extends any[]>(...t: T) => T;
/**
 * @since 1.0.0
 * @deprecated
 */
export declare const tupleCurried: <A>(a: A) => <B>(b: B) => [A, B];
/**
 * Applies a function to an argument ($)
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const apply: <A, B>(f: (a: A) => B) => (a: A) => B;
/**
 * Applies an argument to a function (#)
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const applyFlipped: <A>(a: A) => <B>(f: (a: A) => B) => B;
/**
 * For use with phantom fields
 *
 * @since 1.0.0
 * @deprecated
 */
export declare const phantom: any;
/**
 * A thunk that returns always the `identity` function.
 * For use with `applySecond` methods.
 *
 * @since 1.5.0
 * @deprecated
 */
export declare const constIdentity: () => <A>(a: A) => A;
/**
 * @since 1.9.0
 */
export declare const increment: (n: number) => number;
/**
 * @since 1.9.0
 */
export declare const decrement: (n: number) => number;
/**
 * @since 1.18.0
 */
export declare function absurd<A>(_: never): A;
/**
 * Function composition (from left to right).
 *
 * @example
 * import { flow } from 'fp-ts/lib/function'
 *
 * const len = (s: string): number => s.length
 * const double = (n: number): number => n * 2
 *
 * const f = flow(len, double)
 *
 * assert.strictEqual(f('aaa'), 6)
 *
 * @since 1.19.0
 */
export declare function flow<A extends Array<unknown>, B>(ab: (...a: A) => B): (...a: A) => B;
export declare function flow<A extends Array<unknown>, B, C>(ab: (...a: A) => B, bc: (b: B) => C): (...a: A) => C;
export declare function flow<A extends Array<unknown>, B, C, D>(ab: (...a: A) => B, bc: (b: B) => C, cd: (c: C) => D): (...a: A) => D;
export declare function flow<A extends Array<unknown>, B, C, D, E>(ab: (...a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E): (...a: A) => E;
export declare function flow<A extends Array<unknown>, B, C, D, E, F>(ab: (...a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F): (...a: A) => F;
export declare function flow<A extends Array<unknown>, B, C, D, E, F, G>(ab: (...a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F, fg: (f: F) => G): (...a: A) => G;
export declare function flow<A extends Array<unknown>, B, C, D, E, F, G, H>(ab: (...a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F, fg: (f: F) => G, gh: (g: G) => H): (...a: A) => H;
export declare function flow<A extends Array<unknown>, B, C, D, E, F, G, H, I>(ab: (...a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F, fg: (f: F) => G, gh: (g: G) => H, hi: (h: H) => I): (...a: A) => I;
export declare function flow<A extends Array<unknown>, B, C, D, E, F, G, H, I, J>(ab: (...a: A) => B, bc: (b: B) => C, cd: (c: C) => D, de: (d: D) => E, ef: (e: E) => F, fg: (f: F) => G, gh: (g: G) => H, hi: (h: H) => I, ij: (i: I) => J): (...a: A) => J;
