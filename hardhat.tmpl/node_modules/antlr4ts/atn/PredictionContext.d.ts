/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Array2DHashMap } from "../misc/Array2DHashMap";
import { ATN } from "./ATN";
import { EqualityComparator } from "../misc/EqualityComparator";
import { Equatable } from "../misc/Stubs";
import { PredictionContextCache } from "./PredictionContextCache";
import { Recognizer } from "../Recognizer";
import { RuleContext } from "../RuleContext";
export declare abstract class PredictionContext implements Equatable {
    /**
     * Stores the computed hash code of this {@link PredictionContext}. The hash
     * code is computed in parts to match the following reference algorithm.
     *
     * ```
     * private int referenceHashCode() {
     *   int hash = {@link MurmurHash#initialize MurmurHash.initialize}({@link #INITIAL_HASH});
     *
     *   for (int i = 0; i &lt; this.size; i++) {
     *     hash = {@link MurmurHash#update MurmurHash.update}(hash, {@link #getParent getParent}(i));
     *   }
     *
     *   for (int i = 0; i &lt; this.size; i++) {
     *     hash = {@link MurmurHash#update MurmurHash.update}(hash, {@link #getReturnState getReturnState}(i));
     *   }
     *
     *   hash = {@link MurmurHash#finish MurmurHash.finish}(hash, 2 * this.size);
     *   return hash;
     * }
     * ```
     */
    private readonly cachedHashCode;
    constructor(cachedHashCode: number);
    protected static calculateEmptyHashCode(): number;
    protected static calculateSingleHashCode(parent: PredictionContext, returnState: number): number;
    protected static calculateHashCode(parents: PredictionContext[], returnStates: number[]): number;
    abstract readonly size: number;
    abstract getReturnState(index: number): number;
    abstract findReturnState(returnState: number): number;
    abstract getParent(index: number): PredictionContext;
    protected abstract addEmptyContext(): PredictionContext;
    protected abstract removeEmptyContext(): PredictionContext;
    static fromRuleContext(atn: ATN, outerContext: RuleContext, fullContext?: boolean): PredictionContext;
    private static addEmptyContext;
    private static removeEmptyContext;
    static join(context0: PredictionContext, context1: PredictionContext, contextCache?: PredictionContextCache): PredictionContext;
    static isEmptyLocal(context: PredictionContext): boolean;
    static getCachedContext(context: PredictionContext, contextCache: Array2DHashMap<PredictionContext, PredictionContext>, visited: PredictionContext.IdentityHashMap): PredictionContext;
    appendSingleContext(returnContext: number, contextCache: PredictionContextCache): PredictionContext;
    abstract appendContext(suffix: PredictionContext, contextCache: PredictionContextCache): PredictionContext;
    getChild(returnState: number): PredictionContext;
    abstract readonly isEmpty: boolean;
    abstract readonly hasEmpty: boolean;
    hashCode(): number;
    abstract equals(o: any): boolean;
    toStrings(recognizer: Recognizer<any, any> | undefined, currentState: number, stop?: PredictionContext): string[];
}
export declare class SingletonPredictionContext extends PredictionContext {
    parent: PredictionContext;
    returnState: number;
    constructor(parent: PredictionContext, returnState: number);
    getParent(index: number): PredictionContext;
    getReturnState(index: number): number;
    findReturnState(returnState: number): number;
    get size(): number;
    get isEmpty(): boolean;
    get hasEmpty(): boolean;
    appendContext(suffix: PredictionContext, contextCache: PredictionContextCache): PredictionContext;
    protected addEmptyContext(): PredictionContext;
    protected removeEmptyContext(): PredictionContext;
    equals(o: any): boolean;
}
export declare namespace PredictionContext {
    const EMPTY_LOCAL: PredictionContext;
    const EMPTY_FULL: PredictionContext;
    const EMPTY_LOCAL_STATE_KEY: number;
    const EMPTY_FULL_STATE_KEY: number;
    class IdentityHashMap extends Array2DHashMap<PredictionContext, PredictionContext> {
        constructor();
    }
    class IdentityEqualityComparator implements EqualityComparator<PredictionContext> {
        static readonly INSTANCE: IdentityEqualityComparator;
        private IdentityEqualityComparator;
        hashCode(obj: PredictionContext): number;
        equals(a: PredictionContext, b: PredictionContext): boolean;
    }
}
