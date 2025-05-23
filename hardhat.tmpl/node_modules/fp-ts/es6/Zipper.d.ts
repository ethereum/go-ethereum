/**
 * @file Provides a pointed array, which is a non-empty zipper-like array structure that tracks an index (focus)
 * position in an array. Focus can be moved forward and backwards through the array.
 *
 * The array `[1, 2, 3, 4]` with focus on `3` is represented by `new Zipper([1, 2], 3, [4])`
 *
 * Adapted from
 *
 * - https://github.com/DavidHarrison/purescript-list-zipper
 * - https://github.com/thunklife/purescript-zipper
 * - https://github.com/scalaz/scalaz/blob/series/7.3.x/core/src/main/scala/scalaz/Zipper.scala
 */
import { Applicative1 } from './Applicative';
import { Comonad1 } from './Comonad';
import { Foldable2v1 } from './Foldable2v';
import { Monoid } from './Monoid';
import { NonEmptyArray } from './NonEmptyArray';
import { NonEmptyArray as NonEmptyArray2v } from './NonEmptyArray2v';
import { Option } from './Option';
import { Semigroup } from './Semigroup';
import { Traversable2v1 } from './Traversable2v';
import { Show } from './Show';
declare module './HKT' {
    interface URItoKind<A> {
        Zipper: Zipper<A>;
    }
}
export declare const URI = "Zipper";
export declare type URI = typeof URI;
/**
 * @since 1.9.0
 */
export declare class Zipper<A> {
    readonly lefts: Array<A>;
    readonly focus: A;
    readonly rights: Array<A>;
    readonly _A: A;
    readonly _URI: URI;
    length: number;
    constructor(lefts: Array<A>, focus: A, rights: Array<A>);
    /**
     * Update the focus in this zipper.
     * @since 1.9.0
     */
    update(a: A): Zipper<A>;
    /**
     * Apply `f` to the focus and update with the result.
     * @since 1.9.0
     */
    modify(f: (a: A) => A): Zipper<A>;
    /**
     * @since 1.9.0
     */
    toArray(): Array<A>;
    /**
     * @since 1.9.0
     */
    isOutOfBound(index: number): boolean;
    /**
     * Moves focus in the zipper, or `None` if there is no such element.
     * @since 1.9.0
     */
    move(f: (currentIndex: number) => number): Option<Zipper<A>>;
    /**
     * @since 1.9.0
     */
    up(): Option<Zipper<A>>;
    /**
     * @since 1.9.0
     */
    down(): Option<Zipper<A>>;
    /**
     * Moves focus to the start of the zipper.
     * @since 1.9.0
     */
    start(): Zipper<A>;
    /**
     * Moves focus to the end of the zipper.
     * @since 1.9.0
     */
    end(): Zipper<A>;
    /**
     * Inserts an element to the left of focus and focuses on the new element.
     * @since 1.9.0
     */
    insertLeft(a: A): Zipper<A>;
    /**
     * Inserts an element to the right of focus and focuses on the new element.
     * @since 1.9.0
     */
    insertRight(a: A): Zipper<A>;
    /**
     * Deletes the element at focus and moves the focus to the left. If there is no element on the left,
     * focus is moved to the right.
     * @since 1.9.0
     */
    deleteLeft(): Option<Zipper<A>>;
    /**
     * Deletes the element at focus and moves the focus to the right. If there is no element on the right,
     * focus is moved to the left.
     * @since 1.9.0
     */
    deleteRight(): Option<Zipper<A>>;
    /**
     * @since 1.9.0
     */
    map<B>(f: (a: A) => B): Zipper<B>;
    /**
     * @since 1.9.0
     */
    ap<B>(fab: Zipper<(a: A) => B>): Zipper<B>;
    /**
     * @since 1.9.0
     */
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    inspect(): string;
    toString(): string;
}
/**
 * @since 1.17.0
 */
export declare const getShow: <A>(S: Show<A>) => Show<Zipper<A>>;
/**
 * @since 1.9.0
 */
export declare const fromArray: <A>(as: A[], focusIndex?: number) => Option<Zipper<A>>;
/**
 * @since 1.9.0
 */
export declare const fromNonEmptyArray: <A>(nea: NonEmptyArray<A>) => Zipper<A>;
/**
 * @since 1.17.0
 */
export declare const fromNonEmptyArray2v: <A>(nea: NonEmptyArray2v<A>) => Zipper<A>;
/**
 * @since 1.9.0
 */
export declare const getSemigroup: <A>(S: Semigroup<A>) => Semigroup<Zipper<A>>;
/**
 * @since 1.9.0
 */
export declare const getMonoid: <A>(M: Monoid<A>) => Monoid<Zipper<A>>;
/**
 * @since 1.9.0
 */
export declare const zipper: Applicative1<URI> & Foldable2v1<URI> & Traversable2v1<URI> & Comonad1<URI>;
