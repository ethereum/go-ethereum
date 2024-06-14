/**
 * The sentinel value returned by producers to replace the draft with undefined.
 */
declare const NOTHING: unique symbol;
/**
 * To let Immer treat your class instances as plain immutable objects
 * (albeit with a custom prototype), you must define either an instance property
 * or a static property on each of your custom classes.
 *
 * Otherwise, your class instance will never be drafted, which means it won't be
 * safe to mutate in a produce callback.
 */
declare const DRAFTABLE: unique symbol;

type AnyFunc = (...args: any[]) => any;
type PrimitiveType = number | string | boolean;
/** Object types that should never be mapped */
type AtomicObject = Function | Promise<any> | Date | RegExp;
/**
 * If the lib "ES2015.Collection" is not included in tsconfig.json,
 * types like ReadonlyArray, WeakMap etc. fall back to `any` (specified nowhere)
 * or `{}` (from the node types), in both cases entering an infinite recursion in
 * pattern matching type mappings
 * This type can be used to cast these types to `void` in these cases.
 */
type IfAvailable<T, Fallback = void> = true | false extends (T extends never ? true : false) ? Fallback : keyof T extends never ? Fallback : T;
/**
 * These should also never be mapped but must be tested after regular Map and
 * Set
 */
type WeakReferences = IfAvailable<WeakMap<any, any>> | IfAvailable<WeakSet<any>>;
type WritableDraft<T> = {
    -readonly [K in keyof T]: Draft<T[K]>;
};
/** Convert a readonly type into a mutable type, if possible */
type Draft<T> = T extends PrimitiveType ? T : T extends AtomicObject ? T : T extends ReadonlyMap<infer K, infer V> ? Map<Draft<K>, Draft<V>> : T extends ReadonlySet<infer V> ? Set<Draft<V>> : T extends WeakReferences ? T : T extends object ? WritableDraft<T> : T;
/** Convert a mutable type into a readonly type */
type Immutable<T> = T extends PrimitiveType ? T : T extends AtomicObject ? T : T extends ReadonlyMap<infer K, infer V> ? ReadonlyMap<Immutable<K>, Immutable<V>> : T extends ReadonlySet<infer V> ? ReadonlySet<Immutable<V>> : T extends WeakReferences ? T : T extends object ? {
    readonly [K in keyof T]: Immutable<T[K]>;
} : T;
interface Patch {
    op: "replace" | "remove" | "add";
    path: (string | number)[];
    value?: any;
}
type PatchListener = (patches: Patch[], inversePatches: Patch[]) => void;
/**
 * Utility types
 */
type PatchesTuple<T> = readonly [T, Patch[], Patch[]];
type ValidRecipeReturnType<State> = State | void | undefined | (State extends undefined ? typeof NOTHING : never);
type ReturnTypeWithPatchesIfNeeded<State, UsePatches extends boolean> = UsePatches extends true ? PatchesTuple<State> : State;
/**
 * Core Producer inference
 */
type InferRecipeFromCurried<Curried> = Curried extends (base: infer State, ...rest: infer Args) => any ? ReturnType<Curried> extends State ? (draft: Draft<State>, ...rest: Args) => ValidRecipeReturnType<Draft<State>> : never : never;
type InferInitialStateFromCurried<Curried> = Curried extends (base: infer State, ...rest: any[]) => any ? State : never;
type InferCurriedFromRecipe<Recipe, UsePatches extends boolean> = Recipe extends (draft: infer DraftState, ...args: infer RestArgs) => any ? ReturnType<Recipe> extends ValidRecipeReturnType<DraftState> ? (base: Immutable<DraftState>, ...args: RestArgs) => ReturnTypeWithPatchesIfNeeded<DraftState, UsePatches> : never : never;
type InferCurriedFromInitialStateAndRecipe<State, Recipe, UsePatches extends boolean> = Recipe extends (draft: Draft<State>, ...rest: infer RestArgs) => ValidRecipeReturnType<State> ? (base?: State | undefined, ...args: RestArgs) => ReturnTypeWithPatchesIfNeeded<State, UsePatches> : never;
/**
 * The `produce` function takes a value and a "recipe function" (whose
 * return value often depends on the base state). The recipe function is
 * free to mutate its first argument however it wants. All mutations are
 * only ever applied to a __copy__ of the base state.
 *
 * Pass only a function to create a "curried producer" which relieves you
 * from passing the recipe function every time.
 *
 * Only plain objects and arrays are made mutable. All other objects are
 * considered uncopyable.
 *
 * Note: This function is __bound__ to its `Immer` instance.
 *
 * @param {any} base - the initial state
 * @param {Function} producer - function that receives a proxy of the base state as first argument and which can be freely modified
 * @param {Function} patchListener - optional function that will be called with all the patches produced here
 * @returns {any} a new state, or the initial state if nothing was modified
 */
interface IProduce {
    /** Curried producer that infers the recipe from the curried output function (e.g. when passing to setState) */
    <Curried>(recipe: InferRecipeFromCurried<Curried>, initialState?: InferInitialStateFromCurried<Curried>): Curried;
    /** Curried producer that infers curried from the recipe  */
    <Recipe extends AnyFunc>(recipe: Recipe): InferCurriedFromRecipe<Recipe, false>;
    /** Curried producer that infers curried from the State generic, which is explicitly passed in.  */
    <State>(recipe: (state: Draft<State>, initialState: State) => ValidRecipeReturnType<State>): (state?: State) => State;
    <State, Args extends any[]>(recipe: (state: Draft<State>, ...args: Args) => ValidRecipeReturnType<State>, initialState: State): (state?: State, ...args: Args) => State;
    <State>(recipe: (state: Draft<State>) => ValidRecipeReturnType<State>): (state: State) => State;
    <State, Args extends any[]>(recipe: (state: Draft<State>, ...args: Args) => ValidRecipeReturnType<State>): (state: State, ...args: Args) => State;
    /** Curried producer with initial state, infers recipe from initial state */
    <State, Recipe extends Function>(recipe: Recipe, initialState: State): InferCurriedFromInitialStateAndRecipe<State, Recipe, false>;
    /** Normal producer */
    <Base, D = Draft<Base>>(// By using a default inferred D, rather than Draft<Base> in the recipe, we can override it.
    base: Base, recipe: (draft: D) => ValidRecipeReturnType<D>, listener?: PatchListener): Base;
}
/**
 * Like `produce`, but instead of just returning the new state,
 * a tuple is returned with [nextState, patches, inversePatches]
 *
 * Like produce, this function supports currying
 */
interface IProduceWithPatches {
    <Recipe extends AnyFunc>(recipe: Recipe): InferCurriedFromRecipe<Recipe, true>;
    <State, Recipe extends Function>(recipe: Recipe, initialState: State): InferCurriedFromInitialStateAndRecipe<State, Recipe, true>;
    <Base, D = Draft<Base>>(base: Base, recipe: (draft: D) => ValidRecipeReturnType<D>, listener?: PatchListener): PatchesTuple<Base>;
}

type Objectish = AnyObject | AnyArray | AnyMap | AnySet;
type AnyObject = {
    [key: string]: any;
};
type AnyArray = Array<any>;
type AnySet = Set<any>;
type AnyMap = Map<any, any>;

/** Returns true if the given value is an Immer draft */
declare function isDraft(value: any): boolean;
/** Returns true if the given value can be drafted by Immer */
declare function isDraftable(value: any): boolean;
/** Get the underlying object that is represented by the given draft */
declare function original<T>(value: T): T | undefined;
/**
 * Freezes draftable objects. Returns the original object.
 * By default freezes shallowly, but if the second argument is `true` it will freeze recursively.
 *
 * @param obj
 * @param deep
 */
declare function freeze<T>(obj: T, deep?: boolean): T;

interface ProducersFns {
    produce: IProduce;
    produceWithPatches: IProduceWithPatches;
}
declare class Immer implements ProducersFns {
    autoFreeze_: boolean;
    useStrictShallowCopy_: boolean;
    constructor(config?: {
        autoFreeze?: boolean;
        useStrictShallowCopy?: boolean;
    });
    /**
     * The `produce` function takes a value and a "recipe function" (whose
     * return value often depends on the base state). The recipe function is
     * free to mutate its first argument however it wants. All mutations are
     * only ever applied to a __copy__ of the base state.
     *
     * Pass only a function to create a "curried producer" which relieves you
     * from passing the recipe function every time.
     *
     * Only plain objects and arrays are made mutable. All other objects are
     * considered uncopyable.
     *
     * Note: This function is __bound__ to its `Immer` instance.
     *
     * @param {any} base - the initial state
     * @param {Function} recipe - function that receives a proxy of the base state as first argument and which can be freely modified
     * @param {Function} patchListener - optional function that will be called with all the patches produced here
     * @returns {any} a new state, or the initial state if nothing was modified
     */
    produce: IProduce;
    produceWithPatches: IProduceWithPatches;
    createDraft<T extends Objectish>(base: T): Draft<T>;
    finishDraft<D extends Draft<any>>(draft: D, patchListener?: PatchListener): D extends Draft<infer T> ? T : never;
    /**
     * Pass true to automatically freeze all copies created by Immer.
     *
     * By default, auto-freezing is enabled.
     */
    setAutoFreeze(value: boolean): void;
    /**
     * Pass true to enable strict shallow copy.
     *
     * By default, immer does not copy the object descriptors such as getter, setter and non-enumrable properties.
     */
    setUseStrictShallowCopy(value: boolean): void;
    applyPatches<T extends Objectish>(base: T, patches: Patch[]): T;
}

/** Takes a snapshot of the current state of a draft and finalizes it (but without freezing). This is a great utility to print the current state during debugging (no Proxies in the way). The output of current can also be safely leaked outside the producer. */
declare function current<T>(value: T): T;

declare function enablePatches(): void;

declare function enableMapSet(): void;

/**
 * The `produce` function takes a value and a "recipe function" (whose
 * return value often depends on the base state). The recipe function is
 * free to mutate its first argument however it wants. All mutations are
 * only ever applied to a __copy__ of the base state.
 *
 * Pass only a function to create a "curried producer" which relieves you
 * from passing the recipe function every time.
 *
 * Only plain objects and arrays are made mutable. All other objects are
 * considered uncopyable.
 *
 * Note: This function is __bound__ to its `Immer` instance.
 *
 * @param {any} base - the initial state
 * @param {Function} producer - function that receives a proxy of the base state as first argument and which can be freely modified
 * @param {Function} patchListener - optional function that will be called with all the patches produced here
 * @returns {any} a new state, or the initial state if nothing was modified
 */
declare const produce: IProduce;
/**
 * Like `produce`, but `produceWithPatches` always returns a tuple
 * [nextState, patches, inversePatches] (instead of just the next state)
 */
declare const produceWithPatches: IProduceWithPatches;
/**
 * Pass true to automatically freeze all copies created by Immer.
 *
 * Always freeze by default, even in production mode
 */
declare const setAutoFreeze: (value: boolean) => void;
/**
 * Pass true to enable strict shallow copy.
 *
 * By default, immer does not copy the object descriptors such as getter, setter and non-enumrable properties.
 */
declare const setUseStrictShallowCopy: (value: boolean) => void;
/**
 * Apply an array of Immer patches to the first argument.
 *
 * This function is a producer, which means copy-on-write is in effect.
 */
declare const applyPatches: <T extends Objectish>(base: T, patches: Patch[]) => T;
/**
 * Create an Immer draft from the given base state, which may be a draft itself.
 * The draft can be modified until you finalize it with the `finishDraft` function.
 */
declare const createDraft: <T extends Objectish>(base: T) => Draft<T>;
/**
 * Finalize an Immer draft from a `createDraft` call, returning the base state
 * (if no changes were made) or a modified copy. The draft must *not* be
 * mutated afterwards.
 *
 * Pass a function as the 2nd argument to generate Immer patches based on the
 * changes that were made.
 */
declare const finishDraft: <D extends unknown>(draft: D, patchListener?: PatchListener | undefined) => D extends Draft<infer T> ? T : never;
/**
 * This function is actually a no-op, but can be used to cast an immutable type
 * to an draft type and make TypeScript happy
 *
 * @param value
 */
declare function castDraft<T>(value: T): Draft<T>;
/**
 * This function is actually a no-op, but can be used to cast a mutable type
 * to an immutable type and make TypeScript happy
 * @param value
 */
declare function castImmutable<T>(value: T): Immutable<T>;

export { Draft, Immer, Immutable, Objectish, Patch, PatchListener, applyPatches, castDraft, castImmutable, createDraft, current, enableMapSet, enablePatches, finishDraft, freeze, DRAFTABLE as immerable, isDraft, isDraftable, NOTHING as nothing, original, produce, produceWithPatches, setAutoFreeze, setUseStrictShallowCopy };
