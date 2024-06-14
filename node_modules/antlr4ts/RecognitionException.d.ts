/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { CharStream } from "./CharStream";
import { IntervalSet } from "./misc/IntervalSet";
import { IntStream } from "./IntStream";
import { Lexer } from "./Lexer";
import { ParserRuleContext } from "./ParserRuleContext";
import { Recognizer } from "./Recognizer";
import { RuleContext } from "./RuleContext";
import { Token } from "./Token";
/** The root of the ANTLR exception hierarchy. In general, ANTLR tracks just
 *  3 kinds of errors: prediction errors, failed predicate errors, and
 *  mismatched input errors. In each case, the parser knows where it is
 *  in the input, where it is in the ATN, the rule invocation stack,
 *  and what kind of problem occurred.
 */
export declare class RecognitionException extends Error {
    /** The {@link Recognizer} where this exception originated. */
    private _recognizer?;
    private ctx?;
    private input?;
    /**
     * The current {@link Token} when an error occurred. Since not all streams
     * support accessing symbols by index, we have to track the {@link Token}
     * instance itself.
     */
    private offendingToken?;
    private _offendingState;
    constructor(lexer: Lexer | undefined, input: CharStream);
    constructor(recognizer: Recognizer<Token, any> | undefined, input: IntStream | undefined, ctx: ParserRuleContext | undefined);
    constructor(recognizer: Recognizer<Token, any> | undefined, input: IntStream | undefined, ctx: ParserRuleContext | undefined, message: string);
    /**
     * Get the ATN state number the parser was in at the time the error
     * occurred. For {@link NoViableAltException} and
     * {@link LexerNoViableAltException} exceptions, this is the
     * {@link DecisionState} number. For others, it is the state whose outgoing
     * edge we couldn't match.
     *
     * If the state number is not known, this method returns -1.
     */
    get offendingState(): number;
    protected setOffendingState(offendingState: number): void;
    /**
     * Gets the set of input symbols which could potentially follow the
     * previously matched symbol at the time this exception was thrown.
     *
     * If the set of expected tokens is not known and could not be computed,
     * this method returns `undefined`.
     *
     * @returns The set of token types that could potentially follow the current
     * state in the ATN, or `undefined` if the information is not available.
     */
    get expectedTokens(): IntervalSet | undefined;
    /**
     * Gets the {@link RuleContext} at the time this exception was thrown.
     *
     * If the context is not available, this method returns `undefined`.
     *
     * @returns The {@link RuleContext} at the time this exception was thrown.
     * If the context is not available, this method returns `undefined`.
     */
    get context(): RuleContext | undefined;
    /**
     * Gets the input stream which is the symbol source for the recognizer where
     * this exception was thrown.
     *
     * If the input stream is not available, this method returns `undefined`.
     *
     * @returns The input stream which is the symbol source for the recognizer
     * where this exception was thrown, or `undefined` if the stream is not
     * available.
     */
    get inputStream(): IntStream | undefined;
    getOffendingToken(recognizer?: Recognizer<Token, any>): Token | undefined;
    protected setOffendingToken<TSymbol extends Token>(recognizer: Recognizer<TSymbol, any>, offendingToken?: TSymbol): void;
    /**
     * Gets the {@link Recognizer} where this exception occurred.
     *
     * If the recognizer is not available, this method returns `undefined`.
     *
     * @returns The recognizer where this exception occurred, or `undefined` if
     * the recognizer is not available.
     */
    get recognizer(): Recognizer<any, any> | undefined;
}
