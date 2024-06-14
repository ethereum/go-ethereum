import { Comonad1 } from './Comonad';
import { Foldable2v1 } from './Foldable2v';
import { HKT, Kind, Kind2, Kind3, URIS, URIS2, URIS3 } from './HKT';
import { Monad, Monad1, Monad2, Monad2C, Monad3, Monad3C } from './Monad';
import { Eq } from './Eq';
import { Show } from './Show';
import { Traversable2v1 } from './Traversable2v';
declare module './HKT' {
    interface URItoKind<A> {
        Tree: Tree<A>;
    }
}
export declare const URI = "Tree";
export declare type URI = typeof URI;
export declare type Forest<A> = Array<Tree<A>>;
/**
 * @since 1.6.0
 */
export declare class Tree<A> {
    readonly value: A;
    readonly forest: Forest<A>;
    readonly _A: A;
    readonly _URI: URI;
    constructor(value: A, forest: Forest<A>);
    /** @obsolete */
    map<B>(f: (a: A) => B): Tree<B>;
    /** @obsolete */
    ap<B>(fab: Tree<(a: A) => B>): Tree<B>;
    /**
     * Flipped version of `ap`
     * @since 1.6.0
     * @obsolete
     */
    ap_<B, C>(this: Tree<(b: B) => C>, fb: Tree<B>): Tree<C>;
    /** @obsolete */
    chain<B>(f: (a: A) => Tree<B>): Tree<B>;
    /** @obsolete */
    extract(): A;
    /** @obsolete */
    extend<B>(f: (fa: Tree<A>) => B): Tree<B>;
    /** @obsolete */
    reduce<B>(b: B, f: (b: B, a: A) => B): B;
    inspect(): string;
    toString(): string;
}
/**
 * @since 1.17.0
 */
export declare function getShow<A>(S: Show<A>): Show<Tree<A>>;
/**
 * Use `getEq`
 *
 * @since 1.6.0
 * @deprecated
 */
export declare const getSetoid: <A>(E: Eq<A>) => Eq<Tree<A>>;
/**
 * @since 1.19.0
 */
export declare function getEq<A>(E: Eq<A>): Eq<Tree<A>>;
/**
 * @since 1.6.0
 */
export declare const tree: Monad1<URI> & Foldable2v1<URI> & Traversable2v1<URI> & Comonad1<URI>;
/**
 * Neat 2-dimensional drawing of a forest
 *
 * @since 1.6.0
 */
export declare const drawForest: (forest: Tree<string>[]) => string;
/**
 * Neat 2-dimensional drawing of a tree
 *
 * @example
 * import { Tree, drawTree, tree } from 'fp-ts/lib/Tree'
 *
 * const fa = new Tree('a', [
 *   tree.of('b'),
 *   tree.of('c'),
 *   new Tree('d', [tree.of('e'), tree.of('f')])
 * ])
 *
 * assert.strictEqual(drawTree(fa), `a
 * ├─ b
 * ├─ c
 * └─ d
 *    ├─ e
 *    └─ f`)
 *
 *
 * @since 1.6.0
 */
export declare const drawTree: (tree: Tree<string>) => string;
/**
 * Build a tree from a seed value
 *
 * @since 1.6.0
 */
export declare const unfoldTree: <A, B>(b: B, f: (b: B) => [A, B[]]) => Tree<A>;
/**
 * Build a tree from a seed value
 *
 * @since 1.6.0
 */
export declare const unfoldForest: <A, B>(bs: B[], f: (b: B) => [A, B[]]) => Tree<A>[];
/**
 * Monadic tree builder, in depth-first order
 *
 * @since 1.6.0
 */
export declare function unfoldTreeM<M extends URIS3>(M: Monad3<M>): <U, L, A, B>(b: B, f: (b: B) => Kind3<M, U, L, [A, Array<B>]>) => Kind3<M, U, L, Tree<A>>;
export declare function unfoldTreeM<M extends URIS3, U, L>(M: Monad3C<M, U, L>): <A, B>(b: B, f: (b: B) => Kind3<M, U, L, [A, Array<B>]>) => Kind3<M, U, L, Tree<A>>;
export declare function unfoldTreeM<M extends URIS2>(M: Monad2<M>): <L, A, B>(b: B, f: (b: B) => Kind2<M, L, [A, Array<B>]>) => Kind2<M, L, Tree<A>>;
export declare function unfoldTreeM<M extends URIS2, L>(M: Monad2C<M, L>): <A, B>(b: B, f: (b: B) => Kind2<M, L, [A, Array<B>]>) => Kind2<M, L, Tree<A>>;
export declare function unfoldTreeM<M extends URIS>(M: Monad1<M>): <A, B>(b: B, f: (b: B) => Kind<M, [A, Array<B>]>) => Kind<M, Tree<A>>;
export declare function unfoldTreeM<M>(M: Monad<M>): <A, B>(b: B, f: (b: B) => HKT<M, [A, Array<B>]>) => HKT<M, Tree<A>>;
/**
 * Monadic forest builder, in depth-first order
 *
 * @since 1.6.0
 */
export declare function unfoldForestM<M extends URIS3>(M: Monad3<M>): <U, L, A, B>(bs: Array<B>, f: (b: B) => Kind3<M, U, L, [A, Array<B>]>) => Kind3<M, U, L, Forest<A>>;
export declare function unfoldForestM<M extends URIS3, U, L>(M: Monad3C<M, U, L>): <A, B>(bs: Array<B>, f: (b: B) => Kind3<M, U, L, [A, Array<B>]>) => Kind3<M, U, L, Forest<A>>;
export declare function unfoldForestM<M extends URIS2>(M: Monad2<M>): <L, A, B>(bs: Array<B>, f: (b: B) => Kind2<M, L, [A, Array<B>]>) => Kind2<M, L, Forest<A>>;
export declare function unfoldForestM<M extends URIS2, L>(M: Monad2C<M, L>): <A, B>(bs: Array<B>, f: (b: B) => Kind2<M, L, [A, Array<B>]>) => Kind2<M, L, Forest<A>>;
export declare function unfoldForestM<M extends URIS>(M: Monad1<M>): <A, B>(bs: Array<B>, f: (b: B) => Kind<M, [A, Array<B>]>) => Kind<M, Forest<A>>;
export declare function unfoldForestM<M>(M: Monad<M>): <A, B>(bs: Array<B>, f: (b: B) => HKT<M, [A, Array<B>]>) => HKT<M, Forest<A>>;
/**
 * @since 1.14.0
 */
export declare function elem<A>(E: Eq<A>): (a: A, fa: Tree<A>) => boolean;
/**
 * @since 1.19.0
 */
export declare function make<A>(a: A, forest?: Forest<A>): Tree<A>;
declare const ap: <A>(fa: Tree<A>) => <B>(fab: Tree<(a: A) => B>) => Tree<B>, apFirst: <B>(fb: Tree<B>) => <A>(fa: Tree<A>) => Tree<A>, apSecond: <B>(fb: Tree<B>) => <A>(fa: Tree<A>) => Tree<B>, chain: <A, B>(f: (a: A) => Tree<B>) => (ma: Tree<A>) => Tree<B>, chainFirst: <A, B>(f: (a: A) => Tree<B>) => (ma: Tree<A>) => Tree<A>, duplicate: <A>(ma: Tree<A>) => Tree<Tree<A>>, extend: <A, B>(f: (fa: Tree<A>) => B) => (ma: Tree<A>) => Tree<B>, flatten: <A>(mma: Tree<Tree<A>>) => Tree<A>, foldMap: <M>(M: import("./Monoid").Monoid<M>) => <A>(f: (a: A) => M) => (fa: Tree<A>) => M, map: <A, B>(f: (a: A) => B) => (fa: Tree<A>) => Tree<B>, reduce: <A, B>(b: B, f: (b: B, a: A) => B) => (fa: Tree<A>) => B, reduceRight: <A, B>(b: B, f: (a: A, b: B) => B) => (fa: Tree<A>) => B;
export { ap, apFirst, apSecond, chain, chainFirst, duplicate, extend, flatten, foldMap, map, reduce, reduceRight };
