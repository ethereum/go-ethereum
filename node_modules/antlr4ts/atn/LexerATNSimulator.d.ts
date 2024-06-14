/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATN } from "./ATN";
import { ATNConfig } from "./ATNConfig";
import { ATNConfigSet } from "./ATNConfigSet";
import { ATNSimulator } from "./ATNSimulator";
import { ATNState } from "./ATNState";
import { CharStream } from "../CharStream";
import { DFA } from "../dfa/DFA";
import { DFAState } from "../dfa/DFAState";
import { Lexer } from "../Lexer";
import { LexerActionExecutor } from "./LexerActionExecutor";
import { Transition } from "./Transition";
/** "dup" of ParserInterpreter */
export declare class LexerATNSimulator extends ATNSimulator {
    optimize_tail_calls: boolean;
    protected recog: Lexer | undefined;
    /** The current token's starting index into the character stream.
     *  Shared across DFA to ATN simulation in case the ATN fails and the
     *  DFA did not have a previous accept state. In this case, we use the
     *  ATN-generated exception object.
     */
    protected startIndex: number;
    /** line number 1..n within the input */
    private _line;
    /** The index of the character relative to the beginning of the line 0..n-1 */
    private _charPositionInLine;
    protected mode: number;
    /** Used during DFA/ATN exec to record the most recent accept configuration info */
    protected prevAccept: LexerATNSimulator.SimState;
    constructor(/*@NotNull*/ atn: ATN);
    constructor(/*@NotNull*/ atn: ATN, recog: Lexer | undefined);
    copyState(simulator: LexerATNSimulator): void;
    match(input: CharStream, mode: number): number;
    reset(): void;
    protected matchATN(input: CharStream): number;
    protected execATN(input: CharStream, ds0: DFAState): number;
    /**
     * Get an existing target state for an edge in the DFA. If the target state
     * for the edge has not yet been computed or is otherwise not available,
     * this method returns `undefined`.
     *
     * @param s The current DFA state
     * @param t The next input symbol
     * @returns The existing target DFA state for the given input symbol
     * `t`, or `undefined` if the target state for this edge is not
     * already cached
     */
    protected getExistingTargetState(s: DFAState, t: number): DFAState | undefined;
    /**
     * Compute a target state for an edge in the DFA, and attempt to add the
     * computed state and corresponding edge to the DFA.
     *
     * @param input The input stream
     * @param s The current DFA state
     * @param t The next input symbol
     *
     * @returns The computed target DFA state for the given input symbol
     * `t`. If `t` does not lead to a valid DFA state, this method
     * returns {@link #ERROR}.
     */
    protected computeTargetState(input: CharStream, s: DFAState, t: number): DFAState;
    protected failOrAccept(prevAccept: LexerATNSimulator.SimState, input: CharStream, reach: ATNConfigSet, t: number): number;
    /** Given a starting configuration set, figure out all ATN configurations
     *  we can reach upon input `t`. Parameter `reach` is a return
     *  parameter.
     */
    protected getReachableConfigSet(input: CharStream, closure: ATNConfigSet, reach: ATNConfigSet, t: number): void;
    protected accept(input: CharStream, lexerActionExecutor: LexerActionExecutor | undefined, startIndex: number, index: number, line: number, charPos: number): void;
    protected getReachableTarget(trans: Transition, t: number): ATNState | undefined;
    protected computeStartState(input: CharStream, p: ATNState): ATNConfigSet;
    /**
     * Since the alternatives within any lexer decision are ordered by
     * preference, this method stops pursuing the closure as soon as an accept
     * state is reached. After the first accept state is reached by depth-first
     * search from `config`, all other (potentially reachable) states for
     * this rule would have a lower priority.
     *
     * @returns `true` if an accept state is reached, otherwise
     * `false`.
     */
    protected closure(input: CharStream, config: ATNConfig, configs: ATNConfigSet, currentAltReachedAcceptState: boolean, speculative: boolean, treatEofAsEpsilon: boolean): boolean;
    protected getEpsilonTarget(input: CharStream, config: ATNConfig, t: Transition, configs: ATNConfigSet, speculative: boolean, treatEofAsEpsilon: boolean): ATNConfig | undefined;
    /**
     * Evaluate a predicate specified in the lexer.
     *
     * If `speculative` is `true`, this method was called before
     * {@link #consume} for the matched character. This method should call
     * {@link #consume} before evaluating the predicate to ensure position
     * sensitive values, including {@link Lexer#getText}, {@link Lexer#getLine},
     * and {@link Lexer#getCharPositionInLine}, properly reflect the current
     * lexer state. This method should restore `input` and the simulator
     * to the original state before returning (i.e. undo the actions made by the
     * call to {@link #consume}.
     *
     * @param input The input stream.
     * @param ruleIndex The rule containing the predicate.
     * @param predIndex The index of the predicate within the rule.
     * @param speculative `true` if the current index in `input` is
     * one character before the predicate's location.
     *
     * @returns `true` if the specified predicate evaluates to
     * `true`.
     */
    protected evaluatePredicate(input: CharStream, ruleIndex: number, predIndex: number, speculative: boolean): boolean;
    protected captureSimState(settings: LexerATNSimulator.SimState, input: CharStream, dfaState: DFAState): void;
    protected addDFAEdge(/*@NotNull*/ p: DFAState, t: number, /*@NotNull*/ q: ATNConfigSet): DFAState;
    protected addDFAEdge(/*@NotNull*/ p: DFAState, t: number, /*@NotNull*/ q: DFAState): void;
    /** Add a new DFA state if there isn't one with this set of
     * 	configurations already. This method also detects the first
     * 	configuration containing an ATN rule stop state. Later, when
     * 	traversing the DFA, we will know which rule to accept.
     */
    protected addDFAState(configs: ATNConfigSet): DFAState;
    getDFA(mode: number): DFA;
    /** Get the text matched so far for the current token.
     */
    getText(input: CharStream): string;
    get line(): number;
    set line(line: number);
    get charPositionInLine(): number;
    set charPositionInLine(charPositionInLine: number);
    consume(input: CharStream): void;
    getTokenName(t: number): string;
}
export declare namespace LexerATNSimulator {
    const debug: boolean;
    const dfa_debug: boolean;
    /** When we hit an accept state in either the DFA or the ATN, we
     *  have to notify the character stream to start buffering characters
     *  via {@link IntStream#mark} and record the current state. The current sim state
     *  includes the current index into the input, the current line,
     *  and current character position in that line. Note that the Lexer is
     *  tracking the starting line and characterization of the token. These
     *  variables track the "state" of the simulator when it hits an accept state.
     *
     *  We track these variables separately for the DFA and ATN simulation
     *  because the DFA simulation often has to fail over to the ATN
     *  simulation. If the ATN simulation fails, we need the DFA to fall
     *  back to its previously accepted state, if any. If the ATN succeeds,
     *  then the ATN does the accept and the DFA simulator that invoked it
     *  can simply return the predicted token type.
     */
    class SimState {
        index: number;
        line: number;
        charPos: number;
        dfaState?: DFAState;
        reset(): void;
    }
}
