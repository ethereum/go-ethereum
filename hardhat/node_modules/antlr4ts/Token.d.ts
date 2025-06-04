/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { CharStream } from "./CharStream";
import { TokenSource } from "./TokenSource";
/** A token has properties: text, type, line, character position in the line
 *  (so we can ignore tabs), token channel, index, and source from which
 *  we obtained this token.
 */
export interface Token {
    /**
     * Get the text of the token.
     */
    readonly text: string | undefined;
    /** Get the token type of the token */
    readonly type: number;
    /** The line number on which the 1st character of this token was matched,
     *  line=1..n
     */
    readonly line: number;
    /** The index of the first character of this token relative to the
     *  beginning of the line at which it occurs, 0..n-1
     */
    readonly charPositionInLine: number;
    /** Return the channel this token. Each token can arrive at the parser
     *  on a different channel, but the parser only "tunes" to a single channel.
     *  The parser ignores everything not on DEFAULT_CHANNEL.
     */
    readonly channel: number;
    /** An index from 0..n-1 of the token object in the input stream.
     *  This must be valid in order to print token streams and
     *  use TokenRewriteStream.
     *
     *  Return -1 to indicate that this token was conjured up since
     *  it doesn't have a valid index.
     */
    readonly tokenIndex: number;
    /** The starting character index of the token
     *  This method is optional; return -1 if not implemented.
     */
    readonly startIndex: number;
    /** The last character index of the token.
     *  This method is optional; return -1 if not implemented.
     */
    readonly stopIndex: number;
    /** Gets the {@link TokenSource} which created this token.
     */
    readonly tokenSource: TokenSource | undefined;
    /**
     * Gets the {@link CharStream} from which this token was derived.
     */
    readonly inputStream: CharStream | undefined;
}
export declare namespace Token {
    const INVALID_TYPE: number;
    /** During lookahead operations, this "token" signifies we hit rule end ATN state
     *  and did not follow it despite needing to.
     */
    const EPSILON: number;
    const MIN_USER_TOKEN_TYPE: number;
    const EOF: number;
    /** All tokens go to the parser (unless skip() is called in that rule)
     *  on a particular "channel".  The parser tunes to a particular channel
     *  so that whitespace etc... can go to the parser on a "hidden" channel.
     */
    const DEFAULT_CHANNEL: number;
    /** Anything on different channel than DEFAULT_CHANNEL is not parsed
     *  by parser.
     */
    const HIDDEN_CHANNEL: number;
    /**
     * This is the minimum constant value which can be assigned to a
     * user-defined token channel.
     *
     * The non-negative numbers less than {@link #MIN_USER_CHANNEL_VALUE} are
     * assigned to the predefined channels {@link #DEFAULT_CHANNEL} and
     * {@link #HIDDEN_CHANNEL}.
     *
     * @see `Token.channel`
     */
    const MIN_USER_CHANNEL_VALUE: number;
}
