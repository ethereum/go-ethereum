/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { CharStream } from "./CharStream";
import { Token } from "./Token";
import { TokenFactory } from "./TokenFactory";
/**
 * A source of tokens must provide a sequence of tokens via {@link #nextToken()}
 * and also must reveal it's source of characters; {@link CommonToken}'s text is
 * computed from a {@link CharStream}; it only store indices into the char
 * stream.
 *
 * Errors from the lexer are never passed to the parser. Either you want to keep
 * going or you do not upon token recognition error. If you do not want to
 * continue lexing then you do not want to continue parsing. Just throw an
 * exception not under {@link RecognitionException} and Java will naturally toss
 * you all the way out of the recognizers. If you want to continue lexing then
 * you should not throw an exception to the parser--it has already requested a
 * token. Keep lexing until you get a valid one. Just report errors and keep
 * going, looking for a valid token.
 */
export interface TokenSource {
    /**
     * Return a {@link Token} object from your input stream (usually a
     * {@link CharStream}). Do not fail/return upon lexing error; keep chewing
     * on the characters until you get a good one; errors are not passed through
     * to the parser.
     */
    nextToken(): Token;
    /**
     * Get the line number for the current position in the input stream. The
     * first line in the input is line 1.
     *
     * @returns The line number for the current position in the input stream, or
     * 0 if the current token source does not track line numbers.
     */
    readonly line: number;
    /**
     * Get the index into the current line for the current position in the input
     * stream. The first character on a line has position 0.
     *
     * @returns The line number for the current position in the input stream, or
     * -1 if the current token source does not track character positions.
     */
    readonly charPositionInLine: number;
    /**
     * Get the {@link CharStream} from which this token source is currently
     * providing tokens.
     *
     * @returns The {@link CharStream} associated with the current position in
     * the input, or `undefined` if no input stream is available for the token
     * source.
     */
    readonly inputStream: CharStream | undefined;
    /**
     * Gets the name of the underlying input source. This method returns a
     * non-undefined, non-empty string. If such a name is not known, this method
     * returns {@link IntStream#UNKNOWN_SOURCE_NAME}.
     */
    readonly sourceName: string;
    /**
     * Gets or sets the `TokenFactory` this token source is currently using for
     * creating `Token` objects from the input.
     */
    tokenFactory: TokenFactory;
}
