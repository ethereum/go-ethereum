/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ANTLRErrorListener } from "./ANTLRErrorListener";
import { ATN } from "./atn/ATN";
import { ATNSimulator } from "./atn/ATNSimulator";
import { IntStream } from "./IntStream";
import { ParseInfo } from "./atn/ParseInfo";
import { RecognitionException } from "./RecognitionException";
import { RuleContext } from "./RuleContext";
import { Vocabulary } from "./Vocabulary";
export declare abstract class Recognizer<TSymbol, ATNInterpreter extends ATNSimulator> {
    static readonly EOF: number;
    private static tokenTypeMapCache;
    private static ruleIndexMapCache;
    private readonly _listeners;
    protected _interp: ATNInterpreter;
    private _stateNumber;
    abstract readonly ruleNames: string[];
    /**
     * Get the vocabulary used by the recognizer.
     *
     * @returns A {@link Vocabulary} instance providing information about the
     * vocabulary used by the grammar.
     */
    abstract readonly vocabulary: Vocabulary;
    /**
     * Get a map from token names to token types.
     *
     * Used for XPath and tree pattern compilation.
     */
    getTokenTypeMap(): ReadonlyMap<string, number>;
    /**
     * Get a map from rule names to rule indexes.
     *
     * Used for XPath and tree pattern compilation.
     */
    getRuleIndexMap(): ReadonlyMap<string, number>;
    getTokenType(tokenName: string): number;
    /**
     * If this recognizer was generated, it will have a serialized ATN
     * representation of the grammar.
     *
     * For interpreters, we don't know their serialized ATN despite having
     * created the interpreter from it.
     */
    get serializedATN(): string;
    /** For debugging and other purposes, might want the grammar name.
     *  Have ANTLR generate an implementation for this method.
     */
    abstract readonly grammarFileName: string;
    /**
     * Get the {@link ATN} used by the recognizer for prediction.
     *
     * @returns The {@link ATN} used by the recognizer for prediction.
     */
    get atn(): ATN;
    /**
     * Get the ATN interpreter used by the recognizer for prediction.
     *
     * @returns The ATN interpreter used by the recognizer for prediction.
     */
    get interpreter(): ATNInterpreter;
    /**
     * Set the ATN interpreter used by the recognizer for prediction.
     *
     * @param interpreter The ATN interpreter used by the recognizer for
     * prediction.
     */
    set interpreter(interpreter: ATNInterpreter);
    /** If profiling during the parse/lex, this will return DecisionInfo records
     *  for each decision in recognizer in a ParseInfo object.
     *
     * @since 4.3
     */
    get parseInfo(): Promise<ParseInfo | undefined>;
    /** What is the error header, normally line/character position information? */
    getErrorHeader(e: RecognitionException): string;
    /**
     * @exception NullPointerException if `listener` is `undefined`.
     */
    addErrorListener(listener: ANTLRErrorListener<TSymbol>): void;
    removeErrorListener(listener: ANTLRErrorListener<TSymbol>): void;
    removeErrorListeners(): void;
    getErrorListeners(): Array<ANTLRErrorListener<TSymbol>>;
    getErrorListenerDispatch(): ANTLRErrorListener<TSymbol>;
    sempred(_localctx: RuleContext | undefined, ruleIndex: number, actionIndex: number): boolean;
    precpred(localctx: RuleContext | undefined, precedence: number): boolean;
    action(_localctx: RuleContext | undefined, ruleIndex: number, actionIndex: number): void;
    get state(): number;
    /** Indicate that the recognizer has changed internal state that is
     *  consistent with the ATN state passed in.  This way we always know
     *  where we are in the ATN as the parser goes along. The rule
     *  context objects form a stack that lets us see the stack of
     *  invoking rules. Combine this and we have complete ATN
     *  configuration information.
     */
    set state(atnState: number);
    abstract readonly inputStream: IntStream | undefined;
}
