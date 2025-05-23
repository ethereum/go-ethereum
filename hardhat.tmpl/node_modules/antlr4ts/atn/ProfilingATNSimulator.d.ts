/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNConfigSet } from "./ATNConfigSet";
import { BitSet } from "../misc/BitSet";
import { DecisionInfo } from "./DecisionInfo";
import { DFA } from "../dfa/DFA";
import { DFAState } from "../dfa/DFAState";
import { Parser } from "../Parser";
import { ParserATNSimulator } from "./ParserATNSimulator";
import { ParserRuleContext } from "../ParserRuleContext";
import { PredictionContextCache } from "./PredictionContextCache";
import { SemanticContext } from "./SemanticContext";
import { SimulatorState } from "./SimulatorState";
import { TokenStream } from "../TokenStream";
/**
 * @since 4.3
 */
export declare class ProfilingATNSimulator extends ParserATNSimulator {
    protected decisions: DecisionInfo[];
    protected numDecisions: number;
    protected _input: TokenStream | undefined;
    protected _startIndex: number;
    protected _sllStopIndex: number;
    protected _llStopIndex: number;
    protected currentDecision: number;
    protected currentState: SimulatorState | undefined;
    /** At the point of LL failover, we record how SLL would resolve the conflict so that
     *  we can determine whether or not a decision / input pair is context-sensitive.
     *  If LL gives a different result than SLL's predicted alternative, we have a
     *  context sensitivity for sure. The converse is not necessarily true, however.
     *  It's possible that after conflict resolution chooses minimum alternatives,
     *  SLL could get the same answer as LL. Regardless of whether or not the result indicates
     *  an ambiguity, it is not treated as a context sensitivity because LL prediction
     *  was not required in order to produce a correct prediction for this decision and input sequence.
     *  It may in fact still be a context sensitivity but we don't know by looking at the
     *  minimum alternatives for the current input.
     */
    protected conflictingAltResolvedBySLL: number;
    constructor(parser: Parser);
    adaptivePredict(/*@NotNull*/ input: TokenStream, decision: number, outerContext: ParserRuleContext | undefined): number;
    adaptivePredict(/*@NotNull*/ input: TokenStream, decision: number, outerContext: ParserRuleContext | undefined, useContext: boolean): number;
    protected getStartState(dfa: DFA, input: TokenStream, outerContext: ParserRuleContext, useContext: boolean): SimulatorState | undefined;
    protected computeStartState(dfa: DFA, globalContext: ParserRuleContext, useContext: boolean): SimulatorState;
    protected computeReachSet(dfa: DFA, previous: SimulatorState, t: number, contextCache: PredictionContextCache): SimulatorState | undefined;
    protected getExistingTargetState(previousD: DFAState, t: number): DFAState | undefined;
    protected computeTargetState(dfa: DFA, s: DFAState, remainingGlobalContext: ParserRuleContext, t: number, useContext: boolean, contextCache: PredictionContextCache): [DFAState, ParserRuleContext | undefined];
    protected evalSemanticContextImpl(pred: SemanticContext, parserCallStack: ParserRuleContext, alt: number): boolean;
    protected reportContextSensitivity(dfa: DFA, prediction: number, acceptState: SimulatorState, startIndex: number, stopIndex: number): void;
    protected reportAttemptingFullContext(dfa: DFA, conflictingAlts: BitSet, conflictState: SimulatorState, startIndex: number, stopIndex: number): void;
    protected reportAmbiguity(dfa: DFA, D: DFAState, startIndex: number, stopIndex: number, exact: boolean, ambigAlts: BitSet, configs: ATNConfigSet): void;
    getDecisionInfo(): DecisionInfo[];
    getCurrentState(): SimulatorState | undefined;
}
