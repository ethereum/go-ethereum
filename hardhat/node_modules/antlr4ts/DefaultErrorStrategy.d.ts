/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ANTLRErrorStrategy } from "./ANTLRErrorStrategy";
import { FailedPredicateException } from "./FailedPredicateException";
import { InputMismatchException } from "./InputMismatchException";
import { IntervalSet } from "./misc/IntervalSet";
import { NoViableAltException } from "./NoViableAltException";
import { Parser } from "./Parser";
import { ParserRuleContext } from "./ParserRuleContext";
import { RecognitionException } from "./RecognitionException";
import { Token } from "./Token";
import { TokenSource } from "./TokenSource";
/**
 * This is the default implementation of {@link ANTLRErrorStrategy} used for
 * error reporting and recovery in ANTLR parsers.
 */
export declare class DefaultErrorStrategy implements ANTLRErrorStrategy {
    /**
     * Indicates whether the error strategy is currently "recovering from an
     * error". This is used to suppress reporting multiple error messages while
     * attempting to recover from a detected syntax error.
     *
     * @see #inErrorRecoveryMode
     */
    protected errorRecoveryMode: boolean;
    /** The index into the input stream where the last error occurred.
     * 	This is used to prevent infinite loops where an error is found
     *  but no token is consumed during recovery...another error is found,
     *  ad nauseum.  This is a failsafe mechanism to guarantee that at least
     *  one token/tree node is consumed for two errors.
     */
    protected lastErrorIndex: number;
    protected lastErrorStates?: IntervalSet;
    /**
     * This field is used to propagate information about the lookahead following
     * the previous match. Since prediction prefers completing the current rule
     * to error recovery efforts, error reporting may occur later than the
     * original point where it was discoverable. The original context is used to
     * compute the true expected sets as though the reporting occurred as early
     * as possible.
     */
    protected nextTokensContext?: ParserRuleContext;
    /**
     * @see #nextTokensContext
     */
    protected nextTokensState: number;
    /**
     * {@inheritDoc}
     *
     * The default implementation simply calls {@link #endErrorCondition} to
     * ensure that the handler is not in error recovery mode.
     */
    reset(recognizer: Parser): void;
    /**
     * This method is called to enter error recovery mode when a recognition
     * exception is reported.
     *
     * @param recognizer the parser instance
     */
    protected beginErrorCondition(recognizer: Parser): void;
    /**
     * {@inheritDoc}
     */
    inErrorRecoveryMode(recognizer: Parser): boolean;
    /**
     * This method is called to leave error recovery mode after recovering from
     * a recognition exception.
     *
     * @param recognizer
     */
    protected endErrorCondition(recognizer: Parser): void;
    /**
     * {@inheritDoc}
     *
     * The default implementation simply calls {@link #endErrorCondition}.
     */
    reportMatch(recognizer: Parser): void;
    /**
     * {@inheritDoc}
     *
     * The default implementation returns immediately if the handler is already
     * in error recovery mode. Otherwise, it calls {@link #beginErrorCondition}
     * and dispatches the reporting task based on the runtime type of `e`
     * according to the following table.
     *
     * * {@link NoViableAltException}: Dispatches the call to
     *   {@link #reportNoViableAlternative}
     * * {@link InputMismatchException}: Dispatches the call to
     *   {@link #reportInputMismatch}
     * * {@link FailedPredicateException}: Dispatches the call to
     *   {@link #reportFailedPredicate}
     * * All other types: calls {@link Parser#notifyErrorListeners} to report
     *   the exception
     */
    reportError(recognizer: Parser, e: RecognitionException): void;
    protected notifyErrorListeners(recognizer: Parser, message: string, e: RecognitionException): void;
    /**
     * {@inheritDoc}
     *
     * The default implementation resynchronizes the parser by consuming tokens
     * until we find one in the resynchronization set--loosely the set of tokens
     * that can follow the current rule.
     */
    recover(recognizer: Parser, e: RecognitionException): void;
    /**
     * The default implementation of {@link ANTLRErrorStrategy#sync} makes sure
     * that the current lookahead symbol is consistent with what were expecting
     * at this point in the ATN. You can call this anytime but ANTLR only
     * generates code to check before subrules/loops and each iteration.
     *
     * Implements Jim Idle's magic sync mechanism in closures and optional
     * subrules. E.g.,
     *
     * ```antlr
     * a : sync ( stuff sync )* ;
     * sync : {consume to what can follow sync} ;
     * ```
     *
     * At the start of a sub rule upon error, {@link #sync} performs single
     * token deletion, if possible. If it can't do that, it bails on the current
     * rule and uses the default error recovery, which consumes until the
     * resynchronization set of the current rule.
     *
     * If the sub rule is optional (`(...)?`, `(...)*`, or block
     * with an empty alternative), then the expected set includes what follows
     * the subrule.
     *
     * During loop iteration, it consumes until it sees a token that can start a
     * sub rule or what follows loop. Yes, that is pretty aggressive. We opt to
     * stay in the loop as long as possible.
     *
     * **ORIGINS**
     *
     * Previous versions of ANTLR did a poor job of their recovery within loops.
     * A single mismatch token or missing token would force the parser to bail
     * out of the entire rules surrounding the loop. So, for rule
     *
     * ```antlr
     * classDef : 'class' ID '{' member* '}'
     * ```
     *
     * input with an extra token between members would force the parser to
     * consume until it found the next class definition rather than the next
     * member definition of the current class.
     *
     * This functionality cost a little bit of effort because the parser has to
     * compare token set at the start of the loop and at each iteration. If for
     * some reason speed is suffering for you, you can turn off this
     * functionality by simply overriding this method as a blank { }.
     */
    sync(recognizer: Parser): void;
    /**
     * This is called by {@link #reportError} when the exception is a
     * {@link NoViableAltException}.
     *
     * @see #reportError
     *
     * @param recognizer the parser instance
     * @param e the recognition exception
     */
    protected reportNoViableAlternative(recognizer: Parser, e: NoViableAltException): void;
    /**
     * This is called by {@link #reportError} when the exception is an
     * {@link InputMismatchException}.
     *
     * @see #reportError
     *
     * @param recognizer the parser instance
     * @param e the recognition exception
     */
    protected reportInputMismatch(recognizer: Parser, e: InputMismatchException): void;
    /**
     * This is called by {@link #reportError} when the exception is a
     * {@link FailedPredicateException}.
     *
     * @see #reportError
     *
     * @param recognizer the parser instance
     * @param e the recognition exception
     */
    protected reportFailedPredicate(recognizer: Parser, e: FailedPredicateException): void;
    /**
     * This method is called to report a syntax error which requires the removal
     * of a token from the input stream. At the time this method is called, the
     * erroneous symbol is current `LT(1)` symbol and has not yet been
     * removed from the input stream. When this method returns,
     * `recognizer` is in error recovery mode.
     *
     * This method is called when {@link #singleTokenDeletion} identifies
     * single-token deletion as a viable recovery strategy for a mismatched
     * input error.
     *
     * The default implementation simply returns if the handler is already in
     * error recovery mode. Otherwise, it calls {@link #beginErrorCondition} to
     * enter error recovery mode, followed by calling
     * {@link Parser#notifyErrorListeners}.
     *
     * @param recognizer the parser instance
     */
    protected reportUnwantedToken(recognizer: Parser): void;
    /**
     * This method is called to report a syntax error which requires the
     * insertion of a missing token into the input stream. At the time this
     * method is called, the missing token has not yet been inserted. When this
     * method returns, `recognizer` is in error recovery mode.
     *
     * This method is called when {@link #singleTokenInsertion} identifies
     * single-token insertion as a viable recovery strategy for a mismatched
     * input error.
     *
     * The default implementation simply returns if the handler is already in
     * error recovery mode. Otherwise, it calls {@link #beginErrorCondition} to
     * enter error recovery mode, followed by calling
     * {@link Parser#notifyErrorListeners}.
     *
     * @param recognizer the parser instance
     */
    protected reportMissingToken(recognizer: Parser): void;
    /**
     * {@inheritDoc}
     *
     * The default implementation attempts to recover from the mismatched input
     * by using single token insertion and deletion as described below. If the
     * recovery attempt fails, this method
     * {@link InputMismatchException}.
     *
     * **EXTRA TOKEN** (single token deletion)
     *
     * `LA(1)` is not what we are looking for. If `LA(2)` has the
     * right token, however, then assume `LA(1)` is some extra spurious
     * token and delete it. Then consume and return the next token (which was
     * the `LA(2)` token) as the successful result of the match operation.
     *
     * This recovery strategy is implemented by {@link #singleTokenDeletion}.
     *
     * **MISSING TOKEN** (single token insertion)
     *
     * If current token (at `LA(1)`) is consistent with what could come
     * after the expected `LA(1)` token, then assume the token is missing
     * and use the parser's {@link TokenFactory} to create it on the fly. The
     * "insertion" is performed by returning the created token as the successful
     * result of the match operation.
     *
     * This recovery strategy is implemented by {@link #singleTokenInsertion}.
     *
     * **EXAMPLE**
     *
     * For example, Input `i=(3;` is clearly missing the `')'`. When
     * the parser returns from the nested call to `expr`, it will have
     * call chain:
     *
     * ```
     * stat → expr → atom
     * ```
     *
     * and it will be trying to match the `')'` at this point in the
     * derivation:
     *
     * ```
     * => ID '=' '(' INT ')' ('+' atom)* ';'
     *                    ^
     * ```
     *
     * The attempt to match `')'` will fail when it sees `';'` and
     * call {@link #recoverInline}. To recover, it sees that `LA(1)==';'`
     * is in the set of tokens that can follow the `')'` token reference
     * in rule `atom`. It can assume that you forgot the `')'`.
     */
    recoverInline(recognizer: Parser): Token;
    /**
     * This method implements the single-token insertion inline error recovery
     * strategy. It is called by {@link #recoverInline} if the single-token
     * deletion strategy fails to recover from the mismatched input. If this
     * method returns `true`, `recognizer` will be in error recovery
     * mode.
     *
     * This method determines whether or not single-token insertion is viable by
     * checking if the `LA(1)` input symbol could be successfully matched
     * if it were instead the `LA(2)` symbol. If this method returns
     * `true`, the caller is responsible for creating and inserting a
     * token with the correct type to produce this behavior.
     *
     * @param recognizer the parser instance
     * @returns `true` if single-token insertion is a viable recovery
     * strategy for the current mismatched input, otherwise `false`
     */
    protected singleTokenInsertion(recognizer: Parser): boolean;
    /**
     * This method implements the single-token deletion inline error recovery
     * strategy. It is called by {@link #recoverInline} to attempt to recover
     * from mismatched input. If this method returns `undefined`, the parser and error
     * handler state will not have changed. If this method returns non-`undefined`,
     * `recognizer` will *not* be in error recovery mode since the
     * returned token was a successful match.
     *
     * If the single-token deletion is successful, this method calls
     * {@link #reportUnwantedToken} to report the error, followed by
     * {@link Parser#consume} to actually "delete" the extraneous token. Then,
     * before returning {@link #reportMatch} is called to signal a successful
     * match.
     *
     * @param recognizer the parser instance
     * @returns the successfully matched {@link Token} instance if single-token
     * deletion successfully recovers from the mismatched input, otherwise
     * `undefined`
     */
    protected singleTokenDeletion(recognizer: Parser): Token | undefined;
    /** Conjure up a missing token during error recovery.
     *
     *  The recognizer attempts to recover from single missing
     *  symbols. But, actions might refer to that missing symbol.
     *  For example, x=ID {f($x);}. The action clearly assumes
     *  that there has been an identifier matched previously and that
     *  $x points at that token. If that token is missing, but
     *  the next token in the stream is what we want we assume that
     *  this token is missing and we keep going. Because we
     *  have to return some token to replace the missing token,
     *  we have to conjure one up. This method gives the user control
     *  over the tokens returned for missing tokens. Mostly,
     *  you will want to create something special for identifier
     *  tokens. For literals such as '{' and ',', the default
     *  action in the parser or tree parser works. It simply creates
     *  a CommonToken of the appropriate type. The text will be the token.
     *  If you change what tokens must be created by the lexer,
     *  override this method to create the appropriate tokens.
     */
    protected getMissingSymbol(recognizer: Parser): Token;
    protected constructToken(tokenSource: TokenSource, expectedTokenType: number, tokenText: string, current: Token): Token;
    protected getExpectedTokens(recognizer: Parser): IntervalSet;
    /** How should a token be displayed in an error message? The default
     *  is to display just the text, but during development you might
     *  want to have a lot of information spit out.  Override in that case
     *  to use t.toString() (which, for CommonToken, dumps everything about
     *  the token). This is better than forcing you to override a method in
     *  your token objects because you don't have to go modify your lexer
     *  so that it creates a new Java type.
     */
    protected getTokenErrorDisplay(t: Token | undefined): string;
    protected getSymbolText(symbol: Token): string | undefined;
    protected getSymbolType(symbol: Token): number;
    protected escapeWSAndQuote(s: string): string;
    protected getErrorRecoverySet(recognizer: Parser): IntervalSet;
    /** Consume tokens until one matches the given token set. */
    protected consumeUntil(recognizer: Parser, set: IntervalSet): void;
}
