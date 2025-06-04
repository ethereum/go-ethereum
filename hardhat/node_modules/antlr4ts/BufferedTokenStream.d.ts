/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Interval } from "./misc/Interval";
import { RuleContext } from "./RuleContext";
import { Token } from "./Token";
import { TokenSource } from "./TokenSource";
import { TokenStream } from "./TokenStream";
/**
 * This implementation of {@link TokenStream} loads tokens from a
 * {@link TokenSource} on-demand, and places the tokens in a buffer to provide
 * access to any previous token by index.
 *
 * This token stream ignores the value of {@link Token#getChannel}. If your
 * parser requires the token stream filter tokens to only those on a particular
 * channel, such as {@link Token#DEFAULT_CHANNEL} or
 * {@link Token#HIDDEN_CHANNEL}, use a filtering token stream such a
 * {@link CommonTokenStream}.
 */
export declare class BufferedTokenStream implements TokenStream {
    /**
     * The {@link TokenSource} from which tokens for this stream are fetched.
     */
    private _tokenSource;
    /**
     * A collection of all tokens fetched from the token source. The list is
     * considered a complete view of the input once {@link #fetchedEOF} is set
     * to `true`.
     */
    protected tokens: Token[];
    /**
     * The index into {@link #tokens} of the current token (next token to
     * {@link #consume}). {@link #tokens}`[`{@link #p}`]` should be
     * {@link #LT LT(1)}.
     *
     * This field is set to -1 when the stream is first constructed or when
     * {@link #setTokenSource} is called, indicating that the first token has
     * not yet been fetched from the token source. For additional information,
     * see the documentation of {@link IntStream} for a description of
     * Initializing Methods.
     */
    protected p: number;
    /**
     * Indicates whether the {@link Token#EOF} token has been fetched from
     * {@link #tokenSource} and added to {@link #tokens}. This field improves
     * performance for the following cases:
     *
     * * {@link #consume}: The lookahead check in {@link #consume} to prevent
     *   consuming the EOF symbol is optimized by checking the values of
     *   {@link #fetchedEOF} and {@link #p} instead of calling {@link #LA}.
     * * {@link #fetch}: The check to prevent adding multiple EOF symbols into
     *   {@link #tokens} is trivial with this field.
     */
    protected fetchedEOF: boolean;
    constructor(tokenSource: TokenSource);
    get tokenSource(): TokenSource;
    /** Reset this token stream by setting its token source. */
    set tokenSource(tokenSource: TokenSource);
    get index(): number;
    mark(): number;
    release(marker: number): void;
    seek(index: number): void;
    get size(): number;
    consume(): void;
    /** Make sure index `i` in tokens has a token.
     *
     * @returns `true` if a token is located at index `i`, otherwise
     *    `false`.
     * @see #get(int i)
     */
    protected sync(i: number): boolean;
    /** Add `n` elements to buffer.
     *
     * @returns The actual number of elements added to the buffer.
     */
    protected fetch(n: number): number;
    get(i: number): Token;
    /** Get all tokens from start..stop inclusively. */
    getRange(start: number, stop: number): Token[];
    LA(i: number): number;
    protected tryLB(k: number): Token | undefined;
    LT(k: number): Token;
    tryLT(k: number): Token | undefined;
    /**
     * Allowed derived classes to modify the behavior of operations which change
     * the current stream position by adjusting the target token index of a seek
     * operation. The default implementation simply returns `i`. If an
     * exception is thrown in this method, the current stream index should not be
     * changed.
     *
     * For example, {@link CommonTokenStream} overrides this method to ensure that
     * the seek target is always an on-channel token.
     *
     * @param i The target token index.
     * @returns The adjusted target token index.
     */
    protected adjustSeekIndex(i: number): number;
    protected lazyInit(): void;
    protected setup(): void;
    getTokens(): Token[];
    getTokens(start: number, stop: number): Token[];
    getTokens(start: number, stop: number, types: Set<number>): Token[];
    getTokens(start: number, stop: number, ttype: number): Token[];
    /**
     * Given a starting index, return the index of the next token on channel.
     * Return `i` if `tokens[i]` is on channel. Return the index of
     * the EOF token if there are no tokens on channel between `i` and
     * EOF.
     */
    protected nextTokenOnChannel(i: number, channel: number): number;
    /**
     * Given a starting index, return the index of the previous token on
     * channel. Return `i` if `tokens[i]` is on channel. Return -1
     * if there are no tokens on channel between `i` and 0.
     *
     * If `i` specifies an index at or after the EOF token, the EOF token
     * index is returned. This is due to the fact that the EOF token is treated
     * as though it were on every channel.
     */
    protected previousTokenOnChannel(i: number, channel: number): number;
    /** Collect all tokens on specified channel to the right of
     *  the current token up until we see a token on {@link Lexer#DEFAULT_TOKEN_CHANNEL} or
     *  EOF. If `channel` is `-1`, find any non default channel token.
     */
    getHiddenTokensToRight(tokenIndex: number, channel?: number): Token[];
    /** Collect all tokens on specified channel to the left of
     *  the current token up until we see a token on {@link Lexer#DEFAULT_TOKEN_CHANNEL}.
     *  If `channel` is `-1`, find any non default channel token.
     */
    getHiddenTokensToLeft(tokenIndex: number, channel?: number): Token[];
    protected filterForChannel(from: number, to: number, channel: number): Token[];
    get sourceName(): string;
    /** Get the text of all tokens in this buffer. */
    getText(): string;
    getText(interval: Interval): string;
    getText(context: RuleContext): string;
    getTextFromRange(start: any, stop: any): string;
    /** Get all tokens from lexer until EOF. */
    fill(): void;
    private isWritableToken;
    private isToken;
}
