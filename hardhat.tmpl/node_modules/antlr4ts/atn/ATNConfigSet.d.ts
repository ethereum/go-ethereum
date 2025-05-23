/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Array2DHashSet } from "../misc/Array2DHashSet";
import { ATNConfig } from "./ATNConfig";
import { ATNSimulator } from "./ATNSimulator";
import { ATNState } from "./ATNState";
import { BitSet } from "../misc/BitSet";
import { ConflictInfo } from "./ConflictInfo";
import { JavaSet } from "../misc/Stubs";
import { PredictionContextCache } from "./PredictionContextCache";
/**
 * Represents a set of ATN configurations (see `ATNConfig`). As configurations are added to the set, they are merged
 * with other `ATNConfig` instances already in the set when possible using the graph-structured stack.
 *
 * An instance of this class represents the complete set of positions (with context) in an ATN which would be associated
 * with a single DFA state. Its internal representation is more complex than traditional state used for NFA to DFA
 * conversion due to performance requirements (both improving speed and reducing memory overhead) as well as supporting
 * features such as semantic predicates and non-greedy operators in a form to support ANTLR's prediction algorithm.
 *
 * @author Sam Harwell
 */
export declare class ATNConfigSet implements JavaSet<ATNConfig> {
    /**
     * This maps (state, alt) -> merged {@link ATNConfig}. The key does not account for
     * the {@link ATNConfig#getSemanticContext} of the value, which is only a problem if a single
     * `ATNConfigSet` contains two configs with the same state and alternative
     * but different semantic contexts. When this case arises, the first config
     * added to this map stays, and the remaining configs are placed in {@link #unmerged}.
     *
     * This map is only used for optimizing the process of adding configs to the set,
     * and is `undefined` for read-only sets stored in the DFA.
     */
    private mergedConfigs?;
    /**
     * This is an "overflow" list holding configs which cannot be merged with one
     * of the configs in {@link #mergedConfigs} but have a colliding key. This
     * occurs when two configs in the set have the same state and alternative but
     * different semantic contexts.
     *
     * This list is only used for optimizing the process of adding configs to the set,
     * and is `undefined` for read-only sets stored in the DFA.
     */
    private unmerged?;
    /**
     * This is a list of all configs in this set.
     */
    private configs;
    private _uniqueAlt;
    private _conflictInfo?;
    private _hasSemanticContext;
    private _dipsIntoOuterContext;
    /**
     * When `true`, this config set represents configurations where the entire
     * outer context has been consumed by the ATN interpreter. This prevents the
     * {@link ParserATNSimulator#closure} from pursuing the global FOLLOW when a
     * rule stop state is reached with an empty prediction context.
     *
     * Note: `outermostConfigSet` and {@link #dipsIntoOuterContext} should never
     * be true at the same time.
     */
    private outermostConfigSet;
    private cachedHashCode;
    constructor();
    constructor(set: ATNConfigSet, readonly: boolean);
    /**
     * Get the set of all alternatives represented by configurations in this
     * set.
     */
    getRepresentedAlternatives(): BitSet;
    get isReadOnly(): boolean;
    get isOutermostConfigSet(): boolean;
    set isOutermostConfigSet(outermostConfigSet: boolean);
    getStates(): Array2DHashSet<ATNState>;
    optimizeConfigs(interpreter: ATNSimulator): void;
    clone(readonly: boolean): ATNConfigSet;
    get size(): number;
    get isEmpty(): boolean;
    contains(o: any): boolean;
    [Symbol.iterator](): IterableIterator<ATNConfig>;
    toArray(): ATNConfig[];
    add(e: ATNConfig): boolean;
    add(e: ATNConfig, contextCache: PredictionContextCache | undefined): boolean;
    private updatePropertiesForMergedConfig;
    private updatePropertiesForAddedConfig;
    protected canMerge(left: ATNConfig, leftKey: {
        state: number;
        alt: number;
    }, right: ATNConfig): boolean;
    protected getKey(e: ATNConfig): {
        state: number;
        alt: number;
    };
    containsAll(c: Iterable<any>): boolean;
    addAll(c: Iterable<ATNConfig>): boolean;
    addAll(c: Iterable<ATNConfig>, contextCache: PredictionContextCache): boolean;
    clear(): void;
    equals(obj: any): boolean;
    hashCode(): number;
    toString(): string;
    toString(showContext: boolean): string;
    get uniqueAlt(): number;
    get hasSemanticContext(): boolean;
    set hasSemanticContext(value: boolean);
    get conflictInfo(): ConflictInfo | undefined;
    set conflictInfo(conflictInfo: ConflictInfo | undefined);
    get conflictingAlts(): BitSet | undefined;
    get isExactConflict(): boolean;
    get dipsIntoOuterContext(): boolean;
    get(index: number): ATNConfig;
    protected ensureWritable(): void;
}
