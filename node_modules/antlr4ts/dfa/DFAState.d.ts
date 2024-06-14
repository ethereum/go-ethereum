/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { AcceptStateInfo } from "./AcceptStateInfo";
import { ATN } from "../atn/ATN";
import { ATNConfigSet } from "../atn/ATNConfigSet";
import { LexerActionExecutor } from "../atn/LexerActionExecutor";
import { SemanticContext } from "../atn/SemanticContext";
/** A DFA state represents a set of possible ATN configurations.
 *  As Aho, Sethi, Ullman p. 117 says "The DFA uses its state
 *  to keep track of all possible states the ATN can be in after
 *  reading each input symbol.  That is to say, after reading
 *  input a1a2..an, the DFA is in a state that represents the
 *  subset T of the states of the ATN that are reachable from the
 *  ATN's start state along some path labeled a1a2..an."
 *  In conventional NFA&rarr;DFA conversion, therefore, the subset T
 *  would be a bitset representing the set of states the
 *  ATN could be in.  We need to track the alt predicted by each
 *  state as well, however.  More importantly, we need to maintain
 *  a stack of states, tracking the closure operations as they
 *  jump from rule to rule, emulating rule invocations (method calls).
 *  I have to add a stack to simulate the proper lookahead sequences for
 *  the underlying LL grammar from which the ATN was derived.
 *
 *  I use a set of ATNConfig objects not simple states.  An ATNConfig
 *  is both a state (ala normal conversion) and a RuleContext describing
 *  the chain of rules (if any) followed to arrive at that state.
 *
 *  A DFA state may have multiple references to a particular state,
 *  but with different ATN contexts (with same or different alts)
 *  meaning that state was reached via a different set of rule invocations.
 */
export declare class DFAState {
    stateNumber: number;
    configs: ATNConfigSet;
    /** `edges.get(symbol)` points to target of symbol.
     */
    private readonly edges;
    private _acceptStateInfo;
    /** These keys for these edges are the top level element of the global context. */
    private readonly contextEdges;
    /** Symbols in this set require a global context transition before matching an input symbol. */
    private contextSymbols;
    /**
     * This list is computed by {@link ParserATNSimulator#predicateDFAState}.
     */
    predicates: DFAState.PredPrediction[] | undefined;
    /**
     * Constructs a new `DFAState`.
     *
     * @param configs The set of ATN configurations defining this state.
     */
    constructor(configs: ATNConfigSet);
    get isContextSensitive(): boolean;
    isContextSymbol(symbol: number): boolean;
    setContextSymbol(symbol: number): void;
    setContextSensitive(atn: ATN): void;
    get acceptStateInfo(): AcceptStateInfo | undefined;
    set acceptStateInfo(acceptStateInfo: AcceptStateInfo | undefined);
    get isAcceptState(): boolean;
    get prediction(): number;
    get lexerActionExecutor(): LexerActionExecutor | undefined;
    getTarget(symbol: number): DFAState | undefined;
    setTarget(symbol: number, target: DFAState): void;
    getEdgeMap(): Map<number, DFAState>;
    getContextTarget(invokingState: number): DFAState | undefined;
    setContextTarget(invokingState: number, target: DFAState): void;
    getContextEdgeMap(): Map<number, DFAState>;
    hashCode(): number;
    /**
     * Two {@link DFAState} instances are equal if their ATN configuration sets
     * are the same. This method is used to see if a state already exists.
     *
     * Because the number of alternatives and number of ATN configurations are
     * finite, there is a finite number of DFA states that can be processed.
     * This is necessary to show that the algorithm terminates.
     *
     * Cannot test the DFA state numbers here because in
     * {@link ParserATNSimulator#addDFAState} we need to know if any other state
     * exists that has this exact set of ATN configurations. The
     * {@link #stateNumber} is irrelevant.
     */
    equals(o: any): boolean;
    toString(): string;
}
export declare namespace DFAState {
    /** Map a predicate to a predicted alternative. */
    class PredPrediction {
        pred: SemanticContext;
        alt: number;
        constructor(pred: SemanticContext, alt: number);
        toString(): string;
    }
}
