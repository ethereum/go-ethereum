# Changelog

> **Tags:**
>
> - [New Feature]
> - [Bug Fix]
> - [Breaking Change]
> - [Documentation]
> - [Internal]
> - [Polish]
> - [Experimental]
> - [Deprecation]

**Note**: Gaps between patch versions are faulty/broken releases. **Note**: A feature tagged as Experimental is in a
high state of flux, you're at risk of it changing without notice.

# 1.19.3

- **Polish**
  - add pointer to `Eq` for those looking at deprecated `Setoid` interface, fix #889 (@gcanti)

# 1.19.2

- **Bug Fix**
  - fix `Reader.of` definition (@gcanti)

# 1.19.1

- **Bug Fix**
  - use default type parameters for constructors / lifting functions (@gcanti)

# 1.19.0

The goal of this release is to make the migration to v2 easier.

Since in v2 data types are no more implemented with classes, chainable APIs will be deprecated (in v1.20.0).

As an alternative, a `pipe` function is provided, along with suitable data-last top level functions (one for each deprecated method).

**Example**

Before

```ts
import * as O from 'fp-ts/lib/Option'

O.some(1)
  .map(n => n * 2)
  .chain(n === 0 ? O.none : O.some(1 / n))
  .filter(n => n > 1)
  .foldL(() => 'ko', () => 'ok')
```

After

```ts
import * as O from 'fp-ts/lib/Option'
import { pipe } from 'fp-ts/lib/pipeable'

pipe(
  O.some(1),
  O.map(n => n * 2),
  O.chain(n === 0 ? O.none : O.some(1 / n)),
  O.filter(n => n > 1),
  O.fold(() => 'ko', () => 'ok')
)
```

## Custom tslint rule

In order to make easier to spot all the occurrences of chainable APIs without depending on `@deprecated`, which would force you to migrate in one shot, a custom tslint rule is provided (`@obsolete`).

**Configuration**

Add the following lines to your `tslint.json` to turn the `@obsolete` rule on:

```diff
{
+  "rulesDirectory": ["./node_modules/fp-ts/rules"],
   "rules": {
+    "obsolete": true
   }
}
```

- **New Feature**
  - add `Eq` module (@gcanti)
  - backport top level data-last functions from v2 (@gcanti)
  - backport `pipeable` module form v2 (@gcanti)
  - backport `pipe` function form v2 (@gcanti)
  - backport `flow` function form v2 (@gcanti)
  - `Array`
    - add `isNonEmpty` (@Stouffi)
  - `IOEither`
    - add `foldIO` and `foldIOEither` (@bwlt)
  - `Record`
    - add `record` instance (@gcanti)
    - add `mapWithIndex` (@gcanti)
    - add `reduceWithIndex` (@gcanti)
    - add `foldMapWithIndex` (@gcanti)
    - add `reduceRightWithIndex` (@gcanti)
    - add `hasOwnProperty` (@gcanti)
  - `Map`
    - backport `updateAt` (@gcanti)
    - backport `modifyAt` (@gcanti)
  - `Ord`
    - make `Ord` a contravariant functor (@gcanti)
- **Bug Fix**
  - fix `MonadThrow` definition (@gcanti)
  - `TraversableWithIndex`
    - fix `TraverseWithIndex2` definition (@gcanti)
    - fix `TraverseWithIndex2C` definition (@gcanti)
- **Deprecations**
  - `HKT`
    - deprecate `URI2HKT<n>` in favour of `URItoKind<n>` (@gcanti)
    - deprecate `Type<n>` in favour of `Kind<n>` (@gcanti)
  - deprecate `Setoid` in favour of `Eq` (@gcanti)
  - `Applicative`
    - deprecate `when` (@gcanti)
    - deprecate `getMonoid` (@gcanti)
  - `Apply`
    - deprecate `applyFirst`, use `pipeable`'s `apFirst` (@gcanti)
    - deprecate `applySecond`, use `pipeable`'s `apSecond` (@gcanti)
  - `Array`
    - deprecate `catOptions` in favour of `compact` (@gcanti)
    - deprecate `mapOptions` in favour of `filterMap` (@gcanti)
    - deprecate uncurried `filter` in favour of curried, data-last `filter` (@gcanti)
    - deprecate uncurried `partition` in favour of curried, data-last `partition` (@gcanti)
    - deprecate uncurried `partitionMap` in favour of curried, data-last `partitionMap` (@gcanti)
    - deprecate `fold` / `foldL` in favour of `foldLeft` (@gcanti)
    - deprecate `foldr` in favour of `foldRight` (@gcanti)
    - deprecate `take` in favour of `takeLeft` (@gcanti)
    - deprecate `takeEnd` in favour of `takeRight` (@gcanti)
    - deprecate `takeWhile` in favour of `takeLeftWhile` (@gcanti)
    - deprecate `span` in favour of `spanLeft` (@gcanti)
    - deprecate `drop` in favour of `dropLeft` (@gcanti)
    - deprecate `dropEnd` in favour of `dropRight` (@gcanti)
    - deprecate `dropWhile` in favour of `dropLeftWhile` (@gcanti)
    - deprecate uncurried `findIndex` in favour of curried, data-last `findIndex` (@gcanti)
    - deprecate uncurried `findFirst` in favour of curried, data-last `findFirst` (@gcanti)
    - deprecate uncurried `findFirstMap` in favour of curried, data-last `findFirstMap` (@gcanti)
    - deprecate uncurried `findLast` in favour of curried, data-last `findLast` (@gcanti)
    - deprecate uncurried `findLastMap` in favour of curried, data-last `findLastMap` (@gcanti)
    - deprecate uncurried `findLastIndex` in favour of curried, data-last `findLastIndex` (@gcanti)
    - deprecate uncurried `insertAt` in favour of curried, data-last `insertAt` (@gcanti)
    - deprecate uncurried `updateAt` in favour of curried, data-last `updateAt` (@gcanti)
    - deprecate uncurried `deleteAt` in favour of curried, data-last `deleteAt` (@gcanti)
    - deprecate uncurried `modifyAt` in favour of curried, data-last `modifyAt` (@gcanti)
    - deprecate uncurried `rotate` in favour of curried, data-last `rotate` (@gcanti)
    - deprecate uncurried `chop` in favour of curried, data-last `chop` (@gcanti)
    - deprecate `split` in favour of `splitAt` (@gcanti)
    - deprecate uncurried `chunksOf` in favour of curried, data-last `chunksOf` (@gcanti)
  - `Chain`
    - deprecate `flatten`, use `pipeable`'s `flatten` (@gcanti)
  - `Const`
    - deprecate `Const` constructor in favour of `make` (@gcanti)
  - `Contravariant`
    - deprecate `lift`, use `pipeable`'s `contramap` (@gcanti)
  - `Exception` module is deprecated (@gcanti)
  - `Extend`
    - deprecate `duplicate`, use `pipeable`'s `duplicate` (@gcanti)
  - `Foldable2v`
    - deprecate `fold` (@gcanti)
    - deprecate `sequence_`, use `traverse_` (@gcanti)
    - deprecate `oneOf` (@gcanti)
    - deprecate `sum` (@gcanti)
    - deprecate `product` (@gcanti)
    - deprecate `elem` (@gcanti)
    - deprecate `findFirst` (@gcanti)
    - deprecate `min` (@gcanti)
    - deprecate `max` (@gcanti)
    - deprecate `toArray` (@gcanti)
  - `Free` module is deprecated (@gcanti)
  - `FreeGroup` module is deprecated (@gcanti)
  - `function`
    - deprecate `Function*` types, use `FunctionN` (@gcanti)
    - deprecate `Kleisli` type (@gcanti)
    - deprecate `Cokleisli` type (@gcanti)
    - deprecate `concat` function, use `Array`'s `getSemigroup` (@gcanti)
    - deprecate `compose` function in favour of `flow` (@gcanti)
    - deprecate `pipe` function in favour of `flow` (@gcanti)
    - deprecate `curried` function (@gcanti)
    - deprecate `curry` function (@gcanti)
    - deprecate `toString` function, use `Show` type class (@gcanti)
    - deprecate `apply` function (@gcanti)
    - deprecate `applyFlipped` function (@gcanti)
    - deprecate `constIdentity` function (@gcanti)
    - deprecate `phantom` (@gcanti)
    - deprecate `or` (@gcanti)
    - deprecate `and` (@gcanti)
    - deprecate `on` (@gcanti)
    - deprecate `BinaryOperator` (@gcanti)
  - `Functor`
    - deprecate `lift`, use `pipeable`'s `map` (@gcanti)
    - deprecate `voidRight` (@gcanti)
    - deprecate `voidLeft` (@gcanti)
    - deprecate `flap` (@gcanti)
  - `Validation`
    - deprecate `Validation` module in favour of `Either`'s:
      - `getValidation` (@gcanti)
      - `getValidationSemigroup` (@gcanti)
      - `getValidationMonoid` (@gcanti)
  - `Either`
    - deprecate `getCompactable`, `getFilterable` in favour of `getWitherable` (@gcanti)
  - `IOEither`
    - deprecate `right` in favour of `rightIO` (@gcanti)
    - deprecate `left` in favour of `leftIO` (@gcanti)
    - deprecate `fromLeft` in favour of `left2v` (@gcanti)
    - add `right2v` (@gcanti)
  - `IxIO` module is deprecated (@gcanti)
  - `IxMonad` module is deprecated (@gcanti)
  - `Map`
    - deprecate `insert` in favour of `insertAt` (@gcanti)
    - deprecate `remove` in favour of `deleteAt` (@gcanti)
  - `MonadThrow`
    - deprecate `fromOption` (@gcanti)
    - deprecate `fromEither` (@gcanti)
  - `Monoid`
    - deprecate `getArrayMonoid` in favour of `Array`'s `getMonoid` (@gcanti)
  - `Monoidal` module is deprecated (@gcanti)
  - `NonEmptyArray`
    - deprecate uncurried `groupBy` in favour of curried, data-last `groupBy` (@gcanti)
    - deprecate `findFirst` in favour of `Array`'s `findFirst` (@gcanti)
    - deprecate `findLast` in favour of `Array`'s `findLast` (@gcanti)
    - deprecate `findIndex` in favour of `Array`'s `findIndex` (@gcanti)
    - deprecate `findLastIndex` in favour of `Array`'s `findLastIndex` (@gcanti)
    - deprecate uncurried `insertAt` in favour of curried, data-last `insertAt` (@gcanti)
    - deprecate uncurried `updateAt` in favour of curried, data-last `updateAt` (@gcanti)
    - deprecate uncurried `modifyAt` in favour of curried, data-last `modifyAt` (@gcanti)
    - deprecate uncurried `filter` in favour of curried, data-last `filter` (@gcanti)
    - deprecate uncurried `filterWithIndex` in favour of curried, data-last `filterWithIndex` (@gcanti)
  - `Ord`
    - deprecate `lessThan` in favour of `lt` (@gcanti)
    - deprecate `greaterThan` in favour of `gt` (@gcanti)
    - deprecate `lessThanOrEq` in favour of `leq` (@gcanti)
    - deprecate `greaterThanOrEq` in favour of `geq` (@gcanti)
    - swap `contramap` arguments (@gcanti)
  - `Pair` module is deprecated (@gcanti)
  - `Profunctor`
    - deprecate `lmap` function (@gcanti)
    - deprecate `rmap` function (@gcanti)
  - `ReaderTaskEither`
    - deprecate `right` in favour of `rightTask` (@gcanti)
    - deprecate `left` in favour of `leftTask` (@gcanti)
    - deprecate `fromReader` in favour of `rightReader` (@gcanti)
    - deprecate `fromIO` in favour of `rightIO` (@gcanti)
    - deprecate `fromLeft` in favour of `left2v` (@gcanti)
    - add `right2v` (@gcanti)
  - `Record`
    - deprecate uncurried `collect` in favour of curried, data-last `collect` (@gcanti)
    - deprecate `insert` in favour of `insertAt` (@gcanti)
    - deprecate `remove` in favour of `deleteAt` (@gcanti)
    - deprecate uncurried `pop` in favour of curried, data-last `pop` (@gcanti)
    - deprecate `mapWithKey` in favour of `mapWithIndex` (@gcanti)
    - deprecate `reduceWithKey` in favour of `reduceWithIndex` (@gcanti)
    - deprecate `foldMapWithKey` in favour of `foldMapWithIndex` (@gcanti)
    - deprecate `foldrWithKey` in favour of `reduceRightWithIndex` (@gcanti)
    - deprecate `traverseWithKey` in favour of `traverseWithIndex` (@gcanti)
    - deprecate `traverse` in favour of `traverse2v` (@gcanti)
    - deprecate uncurried `filterWithIndex` in favour of curried, data-last `filterWithIndex` (@gcanti)
    - deprecate uncurried `partitionMap` in favour of curried, data-last `partitionMap` (@gcanti)
    - deprecate uncurried `partition` in favour of curried, data-last `partition` (@gcanti)
    - deprecate `wither` in favour of `record.wither` (@gcanti)
    - deprecate `wilt` in favour of `record.wilt` (@gcanti)
    - deprecate uncurried `filterMap` in favour of curried, data-last `filterMap` (@gcanti)
    - deprecate uncurried `partitionMapWithKey` in favour of curried, data-last `partitionMapWithIndex` (@gcanti)
    - deprecate uncurried `partitionWithKey` in favour of curried, data-last `partitionWithIndex` (@gcanti)
    - deprecate uncurried `filterMapWithKey` in favour of curried, data-last `filterMapWithIndex` (@gcanti)
    - deprecate uncurried `filter` in favour of curried, data-last `filter` (@gcanti)
    - deprecate uncurried `map` in favour of curried, data-last `map` (@gcanti)
    - deprecate uncurried `foldMap` in favour of curried, data-last `foldMap` (@gcanti)
    - deprecate `foldr` in favour of `reduceRight` (@gcanti)
  - `StrMap` module is deprecated (@gcanti)
  - `Task`
    - deprecate `delay` in favour of `delay2v` (@gcanti)
  - `TaskEither`
    - deprecate `right` in favour of `rightTask` (@gcanti)
    - deprecate `left` in favour of `leftTask` (@gcanti)
    - deprecate `fromIO` in favour of `rightIO` (@gcanti)
    - deprecate `fromLeft` in favour of `left2v` (@gcanti)
    - add `right2v` (@gcanti)
  - `These`
    - deprecate `this_` in favour of `left` (@gcanti)
    - deprecate `that` in favour of `right` (@gcanti)
    - deprecate `fromThese` in favour of `toTuple` (@gcanti)
    - deprecate `theseLeft` in favour of `getLeft` (@gcanti)
    - deprecate `theseRight` in favour of `getRight` (@gcanti)
    - deprecate `isThis` in favour of `isLeft` (@gcanti)
    - deprecate `isThat` in favour of `isRight` (@gcanti)
    - deprecate `thisOrBoth` in favour of `leftOrBoth` (@gcanti)
    - deprecate `thatOrBoth` in favour of `rightOrBoth` (@gcanti)
    - deprecate `theseThis` in favour of `getLeftOnly` (@gcanti)
    - deprecate `theseThat` in favour of `getRightOnly` (@gcanti)
  - `Trace` module is deprecated (@gcanti)
  - `Validation` module is deprecated, use `Either`'s `getValidation` (@gcanti)
  - `Writer`
    - deprecate `listens` in favour of `listens2v` (@gcanti)
    - deprecate `censor` in favour of `censor2v` (@gcanti)
  - `Zipper` module is deprecated (@gcanti)

# 1.18.2

- **Polish**
  - fix `NonEmptyArray` definition (@gcanti)
- **Deprecation**
  - deprecate `NonEmptyArray.make` in favour of `cons` (@gcanti)

# 1.18.1

- **Bug Fix**
  - use explicit `concat` function for `getEndomorphismMonoid`, #870 (@mlegenhausen)

# 1.18.0

- **New Feature**
  - add `absurd` function, closes #847 (@gcanti)

# 1.17.4

- **Bug Fix**
  - Don't set `target: es6` in tsconfig.es6.json, fix #863 (@FruitieX)

# 1.17.3

- **Polish**
  - remove `reverse` (mutable) from `NonEmptyArray2v` interface (@gcanti)

# 1.17.2

- **Polish**
  - add `Bifunctor2C` interface (@gcanti)
  - add `Profunctor2C` interface (@gcanti)
  - replace `Array<any>` with `Array<unknown>` in `FunctionN` definition (@ta2gch)
  - add refinement overloads to `filter` / `partition` (`Filterable` type class) (@gcanti)
  - add refinement overloads to `filterWithIndex` / `partitionWithIndex` (`FilterableWithIndex` type class) (@gcanti)
- **Deprecation**
  - deprecate `Array.filter`, `Array.partition` in favour of `Array.array.filter` and `Array.array.partition` (@gcanti)

# 1.17.1

- **Polish**
  - make `Type<URI1, A>` not assignable to `Type<URI2, A>`, closes #536 (@gcanti)

# 1.17.0

- **New Feature**
  - add `Show` type class and related instances (@gcanti)
  - add `fromNonEmptyArray2v` to `Zipper` module (@DenisFrezzato)
  - add `getOrElse` / `getOrElseL` to `TaskEither` (@zanza00)
  - `NonEmptyArray2v` module
    - add `modifyAt` (@gcanti)
    - add `copy` (@gcanti)
  - mark `NonEmptyArray2v` as official module
- **Deprecations**
  - deprecate `liftA<n>` functions (@gcanti)
  - deprecate `NonEmptyArray` module (@gcanti)

# 1.16.1

- **New Feature**
  - add `findFistMap` and `findLastMap` to `Array` module, closes #788 (@sledorze)
  - add `cons` / `snoc` to `NonEmptyArray2v` module, closes #800 (@sledorze)
  - add `Traced` comonad, closes #798 (@gcanti)
  - add `tryCatch` to `Validation` module (@gcanti)
  - add `FunctionN` type alias (@ta2gch)
  - add `MonadThrow` and related instances (@gcanti)
  - add es6 module step to build for tree-shaking support (@FruitieX)
  - add `parseJSON` / `stringifyJSON` to `Either` module (@gcanti)
  - add `Magma` (@gcanti)
  - add `fromFoldableMap` to `Record` module (@gcanti)
- **Polish**
  - `snoc` / `cons` in `Array` now return a `NonEmptyArray` (@sledorze)
  - replace `any` with `unknown` in `Console` module (@gcanti)
  - replace `any` with `unknown` in `Trace` module (@gcanti)

# 1.15.1

- **Regression**
  - revert `SequenceT*` deletion and prevent distribution of conditional types in `sequenceT`, `sequenceS`, fix #790 (@gcanti)

# 1.15.0

**Note**. This version requires `typescript@3.1+` (mapped tuples)

- **New Feature**
  - add `Apply.sequenceS`, closes #688 (@gcanti)
  - make `function.tuple` variadic (@gcanti)
  - make `Semigroup.getTupleSemigroup` variadic (@gcanti)
  - make `Monoid.getTupleMonoid` variadic (@gcanti)
  - make `Ord.getTupleOrd` variadic (@gcanti)
  - make `Setoid.getTupleSetoid` variadic (@gcanti)
  - make `Ring.getTupleRing` variadic (@gcanti)
  - make `Apply.sequenceT` variadic (@gcanti)
- **Experimental**
  - add `NonEmptyArray2v` module (type level non empty arrays), closes #735 (@gcanti)

# 1.14.4

- **Polish**
  - Add overloads to `sequenceT` to allow more arguments (up to 8) (@cdimitroulas)

# 1.14.3

- **Deprecation**
  - deprecate `StrMap.traverseWithKey` in favour of `strmap.traverseWithIndex` (@gcanti)
  - deprecate `OptionT.getOptionT` in favour of `OptionT.getOptionT2v`
  - deprecate useless `OptionT` functions (@gcanti)
  - deprecate `EitherT.getEitherT` in favour of `EitherT.getEitherT2v` (@gcanti)
  - deprecate useless `EitherT` functions (@gcanti)
  - deprecate `ReaderT.getReaderT` in favour of `ReaderT.getReaderT2v` (@gcanti)
  - deprecate useless `ReaderT` functions (@gcanti)
  - deprecate `StateT.getStateT` in favour of `StateT.getStateT2v` (@gcanti)
  - deprecate useless `StateT` functions (@gcanti)
  - deprecate `Ord.getProductOrd` / `Ring.getProductRing` in favour of `Ord.getTupleOrd` / `Ring.getTupleRing` (@gcanti)

# 1.14.2

- **Deprecation**
  - deprecate `Setoid.getRecordSetoid` in favour of `Setoid.getStructSetoid` (@gcanti)
  - deprecate `Setoid.getProductSetoid` in favour of `Setoid.getTupleSetoid` (@gcanti)

# 1.14.1

- **New Feature**
  - add `Map` module (@joshburgess)
  - `Record`
    - functions now support subtypes of `string` for the `K` type parameter (@gcanti)
    - add `every` (@gcanti)
    - add `some` (@gcanti)
    - add `elem` (@gcanti)
  - add `function.constVoid` (@leemhenson)
  - `Set`
    - add `empty` (@gcanti)
    - add `foldMap` (@gcanti)
  - add `Reader.getSemigroup`, `Reader.getMonoid` (@gcanti)
  - `NonEmptyArray`
    - Add `getSetoid` (@MaximeRDY)
    - add `NonEmptyArray.prototype.toArrayMap` (@gcanti)
    - add `NonEmptyArray.prototype.some` (@gcanti)
    - add `NonEmptyArray.prototype.every` (@gcanti)
  - `StrMap`
    - add `StrMap.prototype.every` (@gcanti)
    - add `StrMap.prototype.some` (@gcanti)
    - add `elem` (@gcanti)
  - add `Tree.elem` (@gcanti)
- **Polish**
  - many optimizations (@sledorze)
  - ensure 100% coverage (@gcanti)
- **Deprecations**
  - deprecate `Array.index` in favour of `Array.lookup` (@gcanti)
  - deprecate `NonEmptyArray.prototype.index` in favour of `NonEmptyArray.prototype.lookup` (@gcanti)
  - deprecate `Record.isSubdictionary` in favour of `Record.isSubrecord` (@gcanti)
  - deprecate `Semigroup.getDictionarySemigroup`, `Monoid.getDictionaryMonoid` in favour of `Record.getMonoid` (@gcanti)
  - deprecate `Array.getArraySemigroup` (@gcanti)
  - deprecate `Set.member` in favour of `Set.elem` (@gcanti)
  - deprecate `Array.member` in favour of `Array.elem` (@gcanti)
  - `Record` / `StrMap`: fix withIndex names (@gcanti)
  - use "Struct" instead of "Record", and "Tuple" instead of "Product" (@gcanti)
- **Internal**
  - drop Type-level integrity check (@gcanti)

# 1.13.0

- **New Feature**
  - add `Array.unzip` (@user753)
  - add `Group` type class (@gcanti)
  - add `FreeGroup` module (@gcanti)
  - add `These` functions (@gcanti)
    - `thisOrBoth`
    - `thatOrBoth`
    - `theseThis`
    - `theseThat`
    - `fromOptions`
    - `fromEither`

# 1.12.3

- **Polish**
  - support for constrained domain in `Record` module, closes #685 (@gcanti)
  - optimize `Foldable2v.toArray` (@gcanti)

# 1.12.2

- **Bug Fix**
  - fix `Tree.drawTree` (@gcanti)

# 1.12.1

- **Bug Fix**
  - `array.map` should be safe when executed with a binary function, fix #675 (@gcanti)

# 1.12.0

- **Deprecation**
  - deprecate `Set.difference` in favour of `difference2v` (@gcanti)
- **New Feature**
  - add `Array.union` (@gcanti)
  - add `Array.intersection` (@gcanti)
  - add `Array.difference` (@gcanti)
  - add `Set.compact` (@gcanti)
  - add `Set.separate` (@gcanti)
  - add `Set.filterMap` (@gcanti)
  - add `getCompactableComposition` (@gcanti)
  - add `getFilterableComposition` (@gcanti)
  - add `chainFirst`, `chainSecond` to `TaskEither` (@gcanti)
  - add `NonEmptyArray.prototype.filterWithIndex` (@gcanti)
  - add WithKey variants to `Record` (@gcanti)
    - `reduceWithKey`
    - `foldMapWithKey`
    - `foldrWithKey`
    - `partitionMapWithIndex`
    - `partitionWithIndex`
    - `filterMapWithIndex`
    - `filterWithIndex`
  - add `FunctorWithIndex` type class (@MaximeRDY)
    - `Array` instance (@MaximeRDY)
    - `NonEmptyArray` instance (@MaximeRDY)
    - `StrMap` instance (@gcanti)
    - `getFunctorWithIndexComposition` (@MaximeRDY)
  - add `FoldableWithIndex` type class (@gcanti)
    - `Array` instance (@gcanti)
    - `NonEmptyArray` instance (@gcanti)
    - `StrMap` instance (@gcanti)
  - add `TraversableWithIndex` type class (@gcanti)
    - `Array` instance (@gcanti)
    - `NonEmptyArray` instance (@gcanti)
    - `StrMap` instance (@gcanti)
  - add `FilterableWithIndex` type class (@gcanti)
    - `Array` instance (@gcanti)
    - `StrMap` instance (@gcanti)
- **Internal**
  - upgrade to `typescript@3.2.1` (@gcanti)

# 1.11.3

- **Deprecation**
  - `Array`
    - `refine` in favour of `filter` (@gcanti)
  - `Either`
    - `.prototype.refineOrElse` in favour of `.prototype.filterOrElse` (@gcanti)
    - `.prototype.refineOrElseL` in favour of `.prototype.filterOrElseL` (@gcanti)
    - `fromRefinement` in favour of `fromPredicate` (@gcanti)
  - `Option`
    - `.prototype.refine` in favour of `.prototype.filter` (@gcanti)
    - `fromRefinement` in favour of `fromPredicate` (@gcanti)
- **Polish**
  - use built-in `Record` type in `Record` module (@gcanti)
  - add support for refinements (@gcanti)
    - `Array`
      - `takeWhile`
      - `span`
    - `NonEmptyArray`
      - `.prototype.filter`
    - `ReaderTaskEither`
      - `fromPredicate`
    - `Record`
      - `filter`
    - `Set`
      - `filter`
      - `partition`
    - `StrMap`
      - `filter`
    - `TaskEither`
      - `.prototype.filterOrElse`
      - `.prototype.filterOrElseL`
      - `fromPredicate`
    - `Validation`
      - `fromPredicate`

# 1.11.2

- **Bug Fix**
  - fix `function.toString` when input does not have `Object` on its prototype chain (@gcanti)

# 1.11.1

- **Polish**
  - `ReaderTaskEither.tryCatch`: add the environment as the second argument of the `onrejected` handler (@ascariandrea)

# 1.11.0

- **Deprecation**
  - deprecate `Either.tryCatch` in favour of `Either.tryCatch2v` (@gcanti)
  - deprecate `IOEither.tryCatch` in favour of `IOEither.tryCatch2v` (@gcanti)
- **New Feature**
  - add `Strong` type class (@gcanti)
  - add `Choice` type class (@gcanti)
  - use `unknown` type instead of `{}`, #539 (@gcanti)
  - use `HKT4`, `URIS4`, `URI2HKT4`, #555 (@babakness)
  - `NonEmptyArray` enhancement #627 (@sledorze)
    - `index`
    - `findFirst`
    - `findLast`
    - `findIndex`
    - `findLastIndex`
    - `insertAt`
    - `updateAt`
    - `filter`
  - `TaskEither`
    - add `filterOrElse`, `filterOrElseL`, #619 (@gcanti)
  - `Reader`
    - add `Profunctor` instance #634 (@gcanti)
    - add `Strong` instance (@gcanti)
    - add `Choice` instance (@gcanti)
    - add `Category` instance (@gcanti)
  - add `Category4`, `Functor4`, `Profunctor4`, `Semigroupoid4`, `Strong4` (@gcanti)
- **Bug Fix**
  - fix `TaskEither.taskify` with immutable arguments, #637 (@DenisFrezzato)

# 1.10.1

- **Bug Fix**
  - backport #637 (@gcanti)

# 1.10.0

- **Deprecation**
  - deprecate `Foldable` in favour of `Foldable2v` (\*)
  - deprecate `Traversable` in favour of `Traversable2v` (\*)
- **New Feature**
  - `Array`
    - add `chop` function (@gcanti)
    - add `split` function (@gcanti)
    - add `chunksOf` function (@gcanti)
    - add `takeEnd` function (@gcanti)
    - add `dropEnd` function (@gcanti)
    - add `makeBy` function (@gcanti)
    - add `repeat` function (@gcanti)
    - add `replicate` function (@gcanti)
    - add `findLastIndex` function (@gcanti)
    - add array `comprehension` (@gcanti)
  - `NonEmptyArray`
    - add `length` method (@gcanti)
    - add `groupBy` function (@gcanti)
  - `StrMap`
    - add `empty` constant (@gcanti)
  - `Task`
    - add sequential instance (@giogonzo)
  - `TaskEither`
    - add `attempt` method (@gcanti)
    - add `bracket` function (@gcanti)
    - add sequential instance (@giogonzo)
    - add `foldTask` method (@gcanti)
    - add `foldTaskEither` method (@gcanti)
  - `ReaderTaskEither`
    - add sequential instance (@giogonzo)
  - add `MonadIO` module (@gcanti)
  - add `MonadTask` module (@gcanti)
  - add `Date` module (@gcanti)
  - add `Foldable2v` module + instances (@gcanti)
  - add `Traversable2v` module + instances (@gcanti)
  - add `Record` module (@gcanti)
- **Documentation**
  - refactor docs layout (@gcanti)
  - add examples to `Array` module (@gcanti)
  - type-check the examples while generating the documentation (@gcanti)
  - comparison with ramda
  - add example and explanation for `Array.member` (@fozcodes)
- **Internal**
  - upgrade to typescript@3.1.2 (@gcanti)
  - add `function.not` test case (@gibbok)

(\*) `Foldable` and `Traversable` will be replaced with `Foldable2v` and `Traversable2v` implementations in `fp-ts@2`

# 1.9.0

- **New Feature**
  - add `getSemigroup`, `getApplySemigroup`, `getApplyMonoid` to `TaskEither`, https://github.com/gcanti/fp-ts/pull/563 (@mlegenhausen)
  - add `increment` and `decrement` functions, https://github.com/gcanti/fp-ts/pull/557 (@gcanti)
  - add `Zipper` module, https://github.com/gcanti/fp-ts/pull/558 (@gcanti)
  - add `getMeetMonoid`, `getJoinMonoid`, https://github.com/gcanti/fp-ts/pull/548 (@gcanti)
- **Polish**
  - Never emit sourcemaps, https://github.com/gcanti/fp-ts/pull/569 (@scotttrinh)
  - add `Array.empty`, https://github.com/gcanti/fp-ts/pull/556 (@gcanti)

# 1.8.1

- **Bug Fix**
  - add module augmentation to `Free`, https://github.com/gcanti/fp-ts/pull/559 (@gcanti)

# 1.8.0

- **New Feature**
  - add `IORef` module (@gcanti)

# 1.7.1

- **Internal**
  - add refinement overloading to `Array.findFirst`, closes #522 (@gcanti)

# 1.7.0

- **New Feature**
  - add `Array.foldr`, `Array.foldrL` (@PierreCooper)
  - add `Compactable` type class and related instances (@raveclassic)
  - add `Filterable` type class and related instances (@raveclassic)
  - add `Whitherable` type class and related instances (@raveclassic)
  - add `State.prototype.applyFirst`, `State.prototype.applySecond` (@gcanti)
  - add `Option.getRefinement` (@gcanti)
  - add `Foldable.traverse` (@gcanti)
  - add `Option.getApplySemigroup`, `Option.getApplyMonoid` (@gcanti)
  - add `Either.getSemigroup`, `Either.getApplySemigroup`, `Either.getApplyMonoid` (@gcanti)
  - add `Task.delay` (@gcanti)
  - add `NonEmptyArray.group`, `NonEmptyArray.groupSort` (@MaximeRDY)
- **Bug Fix**
  - fix `Random.randomRange` implementation (@gcanti)
  - fix `Set.partitionMap` signature (@gcanti)
  - sort keys in `StrMap.collect` (@gcanti)
- **Deprecation**
  - deprecate `Traversable.traverse` (@raveclassic)
  - deprecate `Foldable.traverse_` (@gcanti)
- **Internal**
  - add `Applicative2C` overloadings to `Traversable.traverse` (@gcanti)

# 1.6.2

- **Bug Fix**
  - add missing `readonly` modifiers (@gcanti)
    - `BoundedJoinSemilattice`
    - `BoundedMeetSemilattice`
    - `HeytingAlgebra`
    - `JoinSemilattice`
    - `MeetSemilattice`
  - handle `null`, `undefined` in `function.toString` (@gcanti)
  - add overloadings to `Free.foldFree`, fix #470 (@gcanti)

# 1.6.1

- **Polish**
  - `Reader.local`, `ReaderTaskEither.local` should be able to change the environment type (@gcanti)
  - add `Reader.prototype.local`, `ReaderTaskEither.prototype.local` (@gcanti)

# 1.6.0

- **New Feature**
  - add `NonEmptyArray.prototype.last` (@raveclassic)
  - add `IOEither` module (@leemhenson)
  - add `orElse` method to `Either`, `Identity`, `Option` (@raveclassic)
  - add `Alt` instance to `TaskEither` (@gcanti)
  - add `NonEmptyArray.prototype.sort` (@raveclassic)
  - add `TaskEither.fromIOEither` (@gcanti)
  - add `applyFirst` method to `IO`, `Task`, `IOEither`, `TaskEither` (@gcanti)
  - move `ReaderTaskEither` from examples into `src` (@leemhenson)
  - add `NonEmptyArray.prototype.reverse` (@raveclassic)
  - add `TaskEither.fromPredicate` (@leemhenson)
  - add `Tree` module (@gcanti)
  - make `Either.filterOrElseL` more general (@gcanti)
  - add `Either.refineOrElse`, `Either.refineOrElseL` (@gcanti)
  - add `Either.fromRefinement` (@gcanti)
- **Bug Fix**
  - handle undefined errors in callback of `TaskEither.taskify` (@dmechas)
  - fix overloading typings of `TaskEither.taskify` (@gcanti)
- **Internal**
  - make `Writer.prototype.map` lazy (@gcanti)
- **Documentation**
  - handle `example` and `link` tags (@gcanti)

# 1.5.0

- **New Feature**
  - Allow the usage of a custom `Semigroup` for `StrMap.getMonoid` (@mlegenhausen)
  - add `applySecond` method to `IO`, `Task`, `TaskEither`, closes #418 (@gcanti)
  - add `TaskEither.fromIO` (@gcanti)
  - add `Apply.sequenceT` (@raveclassic)
  - add `TaskEither.taskify`, utility to convert callback-based node APIs, closes #422 (@gcanti)

# 1.4.1

- **Bug Fix**
  - fix semigroup usage in `Tuple.ap` implementation (@gcanti)

# 1.4.0

- **New Feature**
  - add `getDictionarySemigroup`, `getObjectSemigroup` to `Semigroup` (@raveclassic)
  - add `getDictionaryMonoid` to `Monoid` (@raveclassic)
  - add `Setoid.setoidDate` and `Ord.ordDate` (@mlegenhausen)
  - add `StrMap#filter` (@mlegenhausen)
  - add `Apply.getSemigroup`, `Applicative.getMonoid` (@gcanti)
  - add lattice typeclass hierarchy, closes #412 (@gcanti)
    - `BooleanAlgebra`
    - `BoundedDistributiveLattice`
    - `BoundedJoinSemilattice`
    - `BoundedLattice`
    - `BoundedMeetSemilattice`
    - `DistributiveLattice`
    - `HeytingAlgebra`
    - `JoinSemilattice`
    - `MeetSemilattice`
- **Internal**
  - add more `Travserable.sequence` overloadings (@gcanti)
  - upgrade to typescript@2.8.3 (@gcanti)
  - use `getObjectSemigroup` in `StrMap.concat` (@gcanti)

# 1.3.0

- **New Feature**
  - add `Array.uniq` (@alex-ketch)
  - add `NonEmptyArray.prototype.min` and `NonEmptyArray.prototype.max` (@raveclassic)
  - add `refine` method to `Option`, closes #396 (@wmaurer)
  - add `Ord.getDualOrd` (@gcanti)
  - add support for `Monad2C` and `Monad3C` to `OptionT`, closes #379 (@gcanti)
  - add `TaskEither.fromLeft` (@gcanti)
  - add `listen`, `pass`, `listens`, `censor` to `Writer` (@gcanti)
  - add `Option.fromRefinement` (@gcanti)
  - add `Array.sortBy`, `Array.sortBy1` (@gcanti)
  - add `Either.fromOptionL`, closes #384 (@gcanti)
  - add `Either.filterOrElse`, `Either.filterOrElseL`, closes #382 (@gcanti)
- **Bug Fix**
  - sort keys in `StrMap.reduce` (@gcanti)
- **Internal**
  - added rimraf and updated npm scripts (@wmaurer)
  - upgrade to tslint@5.9.1, tslint-config-standard@7.0.0 (@gcanti)
  - add issue template (@gcanti)
- **Polish**
  - optimize `Semigroup.fold` (@gcanti)

# 1.2.0

- **New Feature**
  - Make `TaskEither` an instance of `BiFunctor` (@teves-castro)
  - add `EitherT.bimap` (@teves-castro)
  - add `partitionMap` to `Set` (@sledorze)
  - add `getOrd` to `Option` (@sledorze)
  - add `partition` to `Set` (@sledorze)
  - add `contramap` to `Setoid` (@sledorze)
  - add `getOrd` to `Array` (@sledorze)
  - add `chain` to `Set` (@sledorze)
  - add `map` to `Set` (@sledorze)
  - add `fromArray` to `Set` (@sledorze)
  - add `StateT.fromState` (@gcanti)
  - add `StateT.liftF` (@gcanti)
  - add `ReaderT.fromReader` (@gcanti)
- **Bug Fix**
  - fix `Alt` instance of `Validation` (@sledorze)
  - fix `EitherT.chain` signature (@gcanti)
  - use `Type*` in `EitherT1`, `EitherT2` (@gcanti)
  - use `Type*` in `StateT1`, `StateT2` (@gcanti)
  - use `Type*` in `ReaderT1`, `ReaderT2` (@gcanti)
  - Add `readonly` modifier to type classes properties (@gcanti)
- **Internal**
  - make 'Set.difference' do less work (@sledorze)
  - remove unecessary closure creation in `lefts`, `rights` and `mapOption` (@sledorze)
  - remove unnecessary closures (mainly `fold`s) (@sledorze)
  - remove closure creation of Option 'ap' function (@sledorze)
  - add `URIS3` overloadings to `StateT` (@gcanti)
  - add `URIS3` overloadings to `ReaderT` (@gcanti)
  - use definite assignement assertion for phantom fields (@gcanti)
  - upgrade to prettier@1.11.0 (@gcanti)

# 1.1.0

- **New Feature**
  - add `scanLeft`, `scanRight` (@PaNaVTEC)
  - add an optional `onerror` argument to `Either.tryCatch`, fix #323 (@gcanti)
- **Bug Fix**
  - `Either.tryCatch` now refines the error (@gcanti)
- **Docs**
  - Option, alt method (@piq9117)
  - add laws for `Setoid`, `Ord`, `Functor`, `Apply`, `Applicative`, `Chain`, `Monad`, `Alt`, `Alternative`, `Plus`
    (@gcanti)
- **Internal**
  - upgrade to `typescript@2.7.2` (@gcanti)

# 1.0.1

- **Bug Fix**
  - add phantom fields to curried type classes, fix #316 (@gcanti)
  - fix `Unfoldable.replicateA` signatures (@gcanti)
- **Internal**
  - optimize Foldable.oneOf (@gcanti)
  - optimize Foldable.traverse\_ (@gcanti)
  - optimize Foldable.sequence\_ (@gcanti)
  - optimize Foldable.foldr (@gcanti)

# 1.0.0

- **Breaking Change**
  - see https://github.com/gcanti/fp-ts/pull/312 (@gcanti)

# 0.6.8

- **New Feature**
  - `Validation`: add `getOrElse`, `getOrElseValue`, closes #278 (@gcanti)
  - add `Functor2`, `Functor3`, `Apply2`, `Apply3`, `Applicative2`, `Applicative3`, `Chain2`, `Chain3`, `Monad2`,
    `Monad3` in order to better support MTL style (@gcanti)
- **Bug Fix**
  - Flow: reverse order of overloadings in curry declaration, fix #299 (@gcanti)
  - `Set`: fix `union` / `insert` / `toArray` / `reduce` definitions (@gcanti)
- **Experimental**
  - `Validation`: add `chain` (@gcanti)
- **Internal**
  - fix typescript@next errors (@gcanti)
- **Documentation**
  - add `StateTaskEither`, `TaskOption` examples (@gcanti)

# 0.6.7

- **New Feature**
  - Ordering: add fromNumber, toNumber (@gcanti)
  - function: add unsafeCoerce (@gcanti)
  - add Set module, closes #161 (@gcanti)

# 0.6.6

- **New Feature**
  - `Array`: add `rotate` (@gcanti)
  - add some `Foldable` functions (@gcanti)
    - `minimum`
    - `maximum`
    - `sum`
    - `product`
    - `foldM`
    - `oneOf`
    - `elem`
    - `find`
  - `Option`: add `tryCatch` (@gcanti)
  - `Ring`: add `getProductRing` (@gcanti)
  - `Setoid`: add `getProductSetoid` (@gcanti)
  - `Ord`: add `getProductOrd` (@gcanti)
- **Internal**
  - perf optimizations (@sledorze, @gcanti)
  - travis: use node 8 (@gcanti)

# 0.6.5

- **Bug Fix**
  - Flow: use `$NonMaybeType` for `fromNullable` and `mapNullable` (@gcanti)

# 0.6.4

- **New Feature**
  - Array: add `getSemigroup` / `getMonoid`, fix #272 (@gcanti)
  - Setoid: add `getRecordSetoid` (@gcanti)
  - add missing `ap_` methods (Reader, State, Writer) (@gcanti)
  - type `is*` methods as type guards (Option, Either, Validation, These) (@gcanti)
- **Experimental**
  - add Flowtype support (@gcanti)
- **Polish**
  - Correct `ap_` parameter name (@OliverJAsh)
  - update Prettier version (@gcanti)
  - fix `getRecordSemigroup` signature (@gcanti)
  - fix `getRecordMonoid` signature (@gcanti)
  - format markdown files with prettier (@gcanti)

# 0.6.3

- **New Feature**
  - move semigroup methods from Monoid.ts to Semigroup.ts (@valery-paschenkov)

# 0.6.2

- **New Feature**
  - add `function.constNull` and `function.constUndefined` (@raveclassic)

# 0.6.1

- **Breaking Change**
  - upgrade to latest TypeScript (2.6.1), fix #244 (@gcanti)

# 0.5.4

- **New Feature**
  - `Array`: add `findFirst` and `findLast` functions (@chasent)
  - `Option`: add `getOrElseValue`, `filter`, `mapNullable` (@raveclassic)
  - `Either`: add `getOrElseValue` (@raveclassic)
  - `Either`: add `fromNullable` (@gcanti)
- **Bug Fix**
  - `Either`: `equals` now accepts a `Setoid<L>` other than `Setoid<A>`, fix #247 (@gcanti)
  - `Validation`: `equals` now accepts a `Setoid<L>` other than `Setoid<A>`, fix #247 (@gcanti)

# 0.5.3

- **New Feature**
  - add `Invariant` (@gcanti)
  - `Semigroup`: add `getRecordSemigroup`, `getRecordMonoid`, `getMeetSemigroup`, `getJoinSemigroup` (@gcanti)
  - `Ord`: add `getSemigroup`, `fromCompare`, `contramap` (@gcanti)
  - `Option`: add `toUndefined` method (@vegansk)
  - `These`: add `getMonad` (@gcanti)
  - `Foldable`: add `fold` (@gcanti)
  - add `TaskEither` (@gcanti)
  - `Validation`: add `fromEither` (@gcanti)
  - `Task`: add `fromIO` (@gcanti)
  - `Either`: pass value to `getOrElse` (@jiayihu)
  - `Array`: add `span` function (@gcanti)
- **Bug Fix**
  - `Array`: fix `takeWhile`, `dropWhile` (@gcanti)
- **Documentation**
  - add `Moore` machine example (@gcanti)
  - add MTL style example (@gcanti)
  - starting API documentation (@gcanti)
- **Internal**
  - fix `Semigroupoid` definition (@gcanti)
- **Polish**
  - `Ordering`: shorten `orderingSemigroup` definition (@gcanti)
  - `Task`: prefer `{}` to `any`, fix #231 (@OliverJAsh)
  - upgrade to `prettier@1.7.0` (@gcanti)
  - `These`: fix `fold` and `bimap` definitions (@gcanti)
  - fix `ArrayOption` example (@gcanti)
  - `State`: remove `Endomorphism` type alias (@gcanti)
  - `Monoidal`: use `liftA2` (@gcanti)

# 0.5.2

- **Bug Fix**
  - fixed EitherT to only run code on the left once, closes #219 (@nfma)
  - fixed OptionT to only run code on none once (@gcanti)

# 0.5.1

- **Breaking Change**
  - migrate to curried APIs when possible (@raveclassic, @gcanti)
  - remove useless static `of`s (@gcanti)
- **New Feature**
  - Array: add zip and zipWith (@gcanti)
  - Monoid: add getArrayMonoid (@gcanti)
  - Tuple
    - add toString (@gcanti)
    - add getApplicative (@gcanti)
    - add getChainRec (@gcanti)
  - Setoid: add getArraySetoid (@gcanti)
- **Bug fix**
  - Store
    - fix extend implementation (@gcanti)
    - fix toString (@gcanti)
- **Polish**
  - Plus: remove any from signatures (@gcanti)

# 0.4.6

- **New Feature**
  - add endomorphism monoid, fix #189 (@gcanti)
  - add a default implementation of `foldr` using `foldMap` (@gcanti)
  - add `insert`, `remove` and `pop` to `StrMap` (@gcanti)
  - improve `voidLeft`, `voidRight` type inference, fix #191 (@gcanti)
- **Bug Fix**
  - StrMap.size returns a wrong number of key/value pairs, fix #186 (@gcanti)
- **Documentation**
  - start book "fp-ts by examples"

# 0.4.5

- **New Feature**
  - add `contains`, `isNone`, `isSome`, `exists` methods to `Option` (@alexandervanhecke)
  - add `Exception` module (@gcanti)
  - add `Pair` module (@gcanti)
  - add `Trace` module (@gcanti)
  - add `IxMonad` module (@gcanti)
  - add `IxIO` module (@gcanti)
  - add `Either.fromOption` (@gcanti)
- **Documentation**
  - add `StateT` example (@gcanti)
  - add `IxIO` example (@gcanti)

# 0.4.3

- **New Feature**
  - add type-level dictionaries in order to reduce the number of overloadings (@gcanti)
  - add typechecks to the type-level HKT dictionary (@SimonMeskens)
  - add Task.tryCatch, closes #159 (@gcanti)
  - use the bottom `never` type for none, closes #160 (@gcanti)
  - add Random module (@gcanti)
  - add Console module (@gcanti)
  - add FantasyFilterable (@SimonMeskens)
  - add FantasyWitherable (@SimonMeskens)
- **Documentation**
  - add ReaderIO example (@gcanti)
  - add EitherOption example (@gcanti)
- **Polish**
  - TaskEither: rename fromPromise to tryCatch (@gcanti)
- **Internal**
  - add fix-prettier task (@gcanti)
  - remove typings-checker (doesn’t work with ts 2.4.1) (@gcanti)

# 0.4.0

- **Breaking Change**
  - Tuple (wrapped)
  - Dictionary (wrapped, renamed to StrMap)
  - changed
    - Applicative.getCompositionApplicative (also renamed to getApplicativeComposition)
    - Foldable.getCompositionFoldable (also renamed to getFoldableComposition)
    - Functor.getCompositionFunctor (also renamed to getFunctorComposition)
    - Traversable.getCompositionTraversable (also renamed to getTraversableComposition)
    - Free (usage)
    - NaturalTransformation
    - ReaderT
    - StateT
  - removed (temporarily or because the porting is not possible)
    - Id (not possible)
    - Traced
    - IxMonad
    - Mealy
    - FreeAp

# 0.3.5

- **New Feature**
  - Functor: add `flap`, closes #129 (@gcanti)
  - Add getSetoid instances, closes #131 (@gcanti)
  - Add "flipped" ap method to FantasyApply instances, closes #132 (@gcanti)
- **Polish**
  - Examples: correct TaskEither fold method (@OliverJAsh)

# 0.3.4

- **Bug Fix**
  - `Array.snoc` returns wrong results with nested arrays, fix #133 (@gcanti)

# 0.3.3

- **New Feature**
  - Functor: add `voidRight` / `voidLeft`, closes #120 (@gcanti)
  - Add `Mealy` machine, closes #122 (@gcanti)
  - Add `Filterable`, closes #124 (@gcanti)
  - Add `Witherable`, closes #125 (@gcanti)
- **Polish**
  - upgrade to ts 2.3.4
  - Either: make `right` === `of\
  - IxIO example: use new proof

# 0.3.2

- **Bug Fix**
  - IxMonad: remove wrong type constraint (@gcanti)

# 0.3.1

- **New Feature**
  - add `Free Applicative`, closes #106 (@gcanti)
  - Add `Semiring`, closes #107 (@gcanti)
  - Add `Ring`, closes #108 (@gcanti)
  - Add `Field`, closes #109 (@gcanti)
  - Improve `toString` methods, closes #116 (@gcanti)
- **Bug Fix**
  - NonEmptyArray: add missing static `of` (@gcanti)
  - add `_tag` type annotations, closes #118 (@gcanti)
- **Internal**
  - Change `proof`s of implementation (@rilut)
  - use prettier, closes #114 (@gcanti)

# 0.3.0

- **New Feature**
  - add `StateT` monad transformer, closes #104 (@gcanti)
  - add `Store` comonad, closes #100 (@rilut)
  - add `Last` monoid, closes #99 (@gcanti)
  - add `Id` monadfunctor (@gcanti)
  - Array: add extend instance (@gcanti)
  - NonEmptyArray: add comonad instance (@gcanti)
  - `examples` folder
  - `exercises` folder
- **Polish**
  - Tuple: remove StaticFunctor checking (@rilut)
- **Breaking Change** (@gcanti)
  - required typescript version: **2.3.3**
  - drop `Static` prefix in type classes
  - Change contramap signature, closes #32
  - Validation: remove deprecated functions
  - Foldable/toArray
  - Dictionary/fromFoldable
  - Dictionary/toUnfoldable
  - Profunctor/lmap
  - Profunctor/rmap
  - Unfoldable/replicate
  - compositions: renaming and signature changes
    - `getFunctorComposition` -> `getCompositionFunctor`
    - `getApplicativeComposition` -> `getCompositionApplicative`
    - `getFoldableComposition` -> `getCompositionFoldable`
    - `getTraversableComposition` -> `getCompositionTraversable`
  - `OptionT`, `EitherT`, `ReaderT` refactoring
  - drop `IxMonadT`, move `IxIO` to the `examples` folder
  - drop `Trans` module
  - `Free` refactoring
  - drop `rxjs` dependency
  - drop `lib-jsnext` folder
  - make `None` constructor private
  - remove `Pointed` and `Copointed` type classes

# 0.2.9

- **New Feature**
  - add Monoidal type class (@gcanti)
- **Bug Fix**
  - fix `foldMap`, closes #89 (@gcanti)
  - replace `instanceof` checks with valued `_tag`s, fix #96 (@gcanti, @sledorze)

# 0.2.8

- **New Feature**
  - Monoid: add `getFunctionStaticMonoid`, closes #70 (@gcanti)
  - Foldable: add `traverse_` and `sequence_`, closes #71 (@gcanti)
  - add `getStaticMonad` to `EitherT`, `OptionT`, `ReaderT`, closes #81 (@gcanti)
  - Applicative: add `when`, closes #77 (@gcanti)
  - indexed monad type class and `IxMonadT`, closes #73 (@gcanti)
  - Array / function: add refinements, closes #68 (@gcanti, @sledorze)
- **Bug Fix**
  - Either: `of` should return `Either`, fix #80 (@gcanti)
  - fix `toArray` (@gcanti)

# 0.2.7

- **New Feature**
  - `Foldable` module: add `intercalate` function, fix #65 (@gcanti)
  - Add `Profunctor` typeclass, fix #33 (@gcanti, @sledorze)
  - Add `These`, fix #47 (@gcanti)
  - `Apply` module: add `applyFirst` and `applySecond`, fix #60 (@sledorze)
- **Bug Fix**
  - fix `Either.ap` (@sledorze)

# 0.2.6

- **Polish**
  - expose experimental modules (@danielepolencic, @gcanti)

# 0.2.5

- **New Feature**
  - add `getOrElse` to `Either`, fix #39 (@sledorze)
  - add composition of functors, applicatives, foldables, traversables, fix #53 (@gcanti)
- **Experimental**
  - add `EitherT`, fix #36 (@gcanti)
  - add `OptionT`, fix #37 (@gcanti)
  - add `ReaderT`, fix #38 (@gcanti)
  - add `Trans` typeclass (`liftT`), fix #40 (@gcanti)
  - add `Free`, fix #42 (@gcanti)

# 0.2.4

- **Polish**
  - deprecate `validation.getApplicativeS` / `validation.getStaticApplicative` (@gcanti)

# 0.2.3

- **Bug Fix**
  - fix return types of `validation.success` / `validation.failure` (@gcanti)

# 0.2.2

- **Bug Fix**
  - fix `Some.reduce` so it calls `f`, https://github.com/gcanti/fp-ts/pull/45 (@leemhenson)

# 0.2.1

- **New Feature**
  - `Semigroupoid` type class (@gcanti)
  - `Rxjs` module (@gcanti)
  - `Tuple` module (@gcanti)
  - `Dictionary` module (@gcanti)
  - add phantom types to all data structures in order to allow type extraction (@gcanti)
  - add all exports for rollup (@gcanti)

# 0.2

- **Breaking Change**
  - complete refactoring: new technique to get higher kinded types and typeclasses

# 0.1.1

- **New Feature**
  - add support for fantasy-land
- **Breaking Change**
  - complete refactoring
  - remove `data` module
  - remove `newtype` module

# 0.0.4

- **Bug Fix**
  - fix `compose` definition for 5 or more functions (@bumbleblym)

# 0.0.3

- **New Feature**
  - make Array<T> a HKT and deprecate `to`,`from` helper functions, fix #5 (@gcanti)
  - add `Traced` comonad (@bumbleblym)
  - add `getOrElse` method to `Option` (@gcanti)
  - add NonEmptyArray, fix #12 (@gcanti)
- **Polish**
  - add tslint
- **Bug Fix**
  - fix `State` definition (@gcanti)

# 0.0.2

- **Bug Fix**
  - fix `ChainRec` definition (@gcanti)

# 0.0.1

Initial release
