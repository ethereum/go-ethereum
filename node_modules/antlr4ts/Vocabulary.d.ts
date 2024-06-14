/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
/**
 * This interface provides information about the vocabulary used by a
 * recognizer.
 *
 * @see Recognizer.vocabulary
 * @author Sam Harwell
 */
export interface Vocabulary {
    /**
     * Returns the highest token type value. It can be used to iterate from
     * zero to that number, inclusively, thus querying all stored entries.
     * @returns the highest token type value
     */
    readonly maxTokenType: number;
    /**
     * Gets the string literal associated with a token type. The string returned
     * by this method, when not `undefined`, can be used unaltered in a parser
     * grammar to represent this token type.
     *
     * The following table shows examples of lexer rules and the literal
     * names assigned to the corresponding token types.
     *
     * <table>
     *  <tr>
     *   <th>Rule</th>
     *   <th>Literal Name</th>
     *   <th>Java String Literal</th>
     *  </tr>
     *  <tr>
     *   <td>`THIS : 'this';`</td>
     *   <td>`'this'`</td>
     *   <td>`"'this'"`</td>
     *  </tr>
     *  <tr>
     *   <td>`SQUOTE : '\'';`</td>
     *   <td>`'\''`</td>
     *   <td>`"'\\''"`</td>
     *  </tr>
     *  <tr>
     *   <td>`ID : [A-Z]+;`</td>
     *   <td>n/a</td>
     *   <td>`undefined`</td>
     *  </tr>
     * </table>
     *
     * @param tokenType The token type.
     *
     * @returns The string literal associated with the specified token type, or
     * `undefined` if no string literal is associated with the type.
     */
    getLiteralName(tokenType: number): string | undefined;
    /**
     * Gets the symbolic name associated with a token type. The string returned
     * by this method, when not `undefined`, can be used unaltered in a parser
     * grammar to represent this token type.
     *
     * This method supports token types defined by any of the following
     * methods:
     *
     * * Tokens created by lexer rules.
     * * Tokens defined in a `tokens{}` block in a lexer or parser
     *   grammar.
     * * The implicitly defined `EOF` token, which has the token type
     *   {@link Token#EOF}.
     *
     * The following table shows examples of lexer rules and the literal
     * names assigned to the corresponding token types.
     *
     * <table>
     *  <tr>
     *   <th>Rule</th>
     *   <th>Symbolic Name</th>
     *  </tr>
     *  <tr>
     *   <td>`THIS : 'this';`</td>
     *   <td>`THIS`</td>
     *  </tr>
     *  <tr>
     *   <td>`SQUOTE : '\'';`</td>
     *   <td>`SQUOTE`</td>
     *  </tr>
     *  <tr>
     *   <td>`ID : [A-Z]+;`</td>
     *   <td>`ID`</td>
     *  </tr>
     * </table>
     *
     * @param tokenType The token type.
     *
     * @returns The symbolic name associated with the specified token type, or
     * `undefined` if no symbolic name is associated with the type.
     */
    getSymbolicName(tokenType: number): string | undefined;
    /**
     * Gets the display name of a token type.
     *
     * ANTLR provides a default implementation of this method, but
     * applications are free to override the behavior in any manner which makes
     * sense for the application. The default implementation returns the first
     * result from the following list which produces a non-`undefined`
     * result.
     *
     * 1. The result of {@link #getLiteralName}
     * 1. The result of {@link #getSymbolicName}
     * 1. The result of {@link Integer#toString}
     *
     * @param tokenType The token type.
     *
     * @returns The display name of the token type, for use in error reporting or
     * other user-visible messages which reference specific token types.
     */
    getDisplayName(tokenType: number): string;
}
