/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Array2DHashSet } from "../misc/Array2DHashSet";
import { ATN } from "./ATN";
import { ATNConfig } from "./ATNConfig";
import { ATNState } from "./ATNState";
import { BitSet } from "../misc/BitSet";
import { IntervalSet } from "../misc/IntervalSet";
import { PredictionContext } from "./PredictionContext";
export declare class LL1Analyzer {
    /** Special value added to the lookahead sets to indicate that we hit
     *  a predicate during analysis if `seeThruPreds==false`.
     */
    static readonly HIT_PRED: number;
    atn: ATN;
    constructor(atn: ATN);
    /**
     * Calculates the SLL(1) expected lookahead set for each outgoing transition
     * of an {@link ATNState}. The returned array has one element for each
     * outgoing transition in `s`. If the closure from transition
     * *i* leads to a semantic predicate before matching a symbol, the
     * element at index *i* of the result will be `undefined`.
     *
     * @param s the ATN state
     * @returns the expected symbols for each outgoing transition of `s`.
     */
    getDecisionLookahead(s: ATNState | undefined): Array<IntervalSet | undefined> | undefined;
    /**
     * Compute set of tokens that can follow `s` in the ATN in the
     * specified `ctx`.
     *
     * If `ctx` is `undefined` and the end of the rule containing
     * `s` is reached, {@link Token#EPSILON} is added to the result set.
     * If `ctx` is not `undefined` and the end of the outermost rule is
     * reached, {@link Token#EOF} is added to the result set.
     *
     * @param s the ATN state
     * @param ctx the complete parser context, or `undefined` if the context
     * should be ignored
     *
     * @returns The set of tokens that can follow `s` in the ATN in the
     * specified `ctx`.
     */
    LOOK(/*@NotNull*/ s: ATNState, /*@NotNull*/ ctx: PredictionContext): IntervalSet;
    /**
     * Compute set of tokens that can follow `s` in the ATN in the
     * specified `ctx`.
     *
     * If `ctx` is `undefined` and the end of the rule containing
     * `s` is reached, {@link Token#EPSILON} is added to the result set.
     * If `ctx` is not `PredictionContext#EMPTY_LOCAL` and the end of the outermost rule is
     * reached, {@link Token#EOF} is added to the result set.
     *
     * @param s the ATN state
     * @param stopState the ATN state to stop at. This can be a
     * {@link BlockEndState} to detect epsilon paths through a closure.
     * @param ctx the complete parser context, or `undefined` if the context
     * should be ignored
     *
     * @returns The set of tokens that can follow `s` in the ATN in the
     * specified `ctx`.
     */
    LOOK(/*@NotNull*/ s: ATNState, /*@NotNull*/ ctx: PredictionContext, stopState: ATNState | null): IntervalSet;
    /**
     * Compute set of tokens that can follow `s` in the ATN in the
     * specified `ctx`.
     * <p/>
     * If `ctx` is {@link PredictionContext#EMPTY_LOCAL} and
     * `stopState` or the end of the rule containing `s` is reached,
     * {@link Token#EPSILON} is added to the result set. If `ctx` is not
     * {@link PredictionContext#EMPTY_LOCAL} and `addEOF` is `true`
     * and `stopState` or the end of the outermost rule is reached,
     * {@link Token#EOF} is added to the result set.
     *
     * @param s the ATN state.
     * @param stopState the ATN state to stop at. This can be a
     * {@link BlockEndState} to detect epsilon paths through a closure.
     * @param ctx The outer context, or {@link PredictionContext#EMPTY_LOCAL} if
     * the outer context should not be used.
     * @param look The result lookahead set.
     * @param lookBusy A set used for preventing epsilon closures in the ATN
     * from causing a stack overflow. Outside code should pass
     * `new HashSet<ATNConfig>` for this argument.
     * @param calledRuleStack A set used for preventing left recursion in the
     * ATN from causing a stack overflow. Outside code should pass
     * `new BitSet()` for this argument.
     * @param seeThruPreds `true` to true semantic predicates as
     * implicitly `true` and "see through them", otherwise `false`
     * to treat semantic predicates as opaque and add {@link #HIT_PRED} to the
     * result if one is encountered.
     * @param addEOF Add {@link Token#EOF} to the result if the end of the
     * outermost context is reached. This parameter has no effect if `ctx`
     * is {@link PredictionContext#EMPTY_LOCAL}.
     */
    protected _LOOK(s: ATNState, stopState: ATNState | undefined, ctx: PredictionContext, look: IntervalSet, lookBusy: Array2DHashSet<ATNConfig>, calledRuleStack: BitSet, seeThruPreds: boolean, addEOF: boolean): void;
}
