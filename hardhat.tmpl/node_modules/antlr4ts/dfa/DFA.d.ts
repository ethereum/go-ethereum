/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Array2DHashSet } from "../misc/Array2DHashSet";
import { ATN } from "../atn/ATN";
import { ATNState } from "../atn/ATNState";
import { DecisionState } from "../atn/DecisionState";
import { DFAState } from "./DFAState";
import { TokensStartState } from "../atn/TokensStartState";
import { Vocabulary } from "../Vocabulary";
export declare class DFA {
    /**
     * A set of all states in the `DFA`.
     *
     * Note that this collection of states holds the DFA states for both SLL and LL prediction. Only the start state
     * needs to be differentiated for these cases, which is tracked by the `s0` and `s0full` fields.
     */
    readonly states: Array2DHashSet<DFAState>;
    s0: DFAState | undefined;
    s0full: DFAState | undefined;
    readonly decision: number;
    /** From which ATN state did we create this DFA? */
    atnStartState: ATNState;
    /**
     * Note: this field is accessed as `atnStartState.atn` in other targets. The TypeScript target keeps a separate copy
     * to avoid a number of additional null/undefined checks each time the ATN is accessed.
     */
    atn: ATN;
    private nextStateNumber;
    /**
     * `true` if this DFA is for a precedence decision; otherwise,
     * `false`. This is the backing field for {@link #isPrecedenceDfa}.
     */
    private precedenceDfa;
    /**
     * Constructs a `DFA` instance associated with a lexer mode.
     *
     * The start state for a `DFA` constructed with this constructor should be a `TokensStartState`, which is the start
     * state for a lexer mode. The prediction made by this DFA determines the lexer rule which matches the current
     * input.
     *
     * @param atnStartState The start state for the mode.
     */
    constructor(atnStartState: TokensStartState);
    /**
     * Constructs a `DFA` instance associated with a decision.
     *
     * @param atnStartState The decision associated with this DFA.
     * @param decision The decision number.
     */
    constructor(atnStartState: DecisionState, decision: number);
    /**
     * Gets whether this DFA is a precedence DFA. Precedence DFAs use a special
     * start state {@link #s0} which is not stored in {@link #states}. The
     * {@link DFAState#edges} array for this start state contains outgoing edges
     * supplying individual start states corresponding to specific precedence
     * values.
     *
     * @returns `true` if this is a precedence DFA; otherwise,
     * `false`.
     * @see Parser.precedence
     */
    get isPrecedenceDfa(): boolean;
    /**
     * Get the start state for a specific precedence value.
     *
     * @param precedence The current precedence.
     * @returns The start state corresponding to the specified precedence, or
     * `undefined` if no start state exists for the specified precedence.
     *
     * @ if this is not a precedence DFA.
     * @see `isPrecedenceDfa`
     */
    getPrecedenceStartState(precedence: number, fullContext: boolean): DFAState | undefined;
    /**
     * Set the start state for a specific precedence value.
     *
     * @param precedence The current precedence.
     * @param startState The start state corresponding to the specified
     * precedence.
     *
     * @ if this is not a precedence DFA.
     * @see `isPrecedenceDfa`
     */
    setPrecedenceStartState(precedence: number, fullContext: boolean, startState: DFAState): void;
    get isEmpty(): boolean;
    get isContextSensitive(): boolean;
    addState(state: DFAState): DFAState;
    toString(): string;
    toString(/*@NotNull*/ vocabulary: Vocabulary): string;
    toString(/*@NotNull*/ vocabulary: Vocabulary, ruleNames: string[] | undefined): string;
    toLexerString(): string;
}
