/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { CommonToken } from "../../CommonToken";
/**
 * A {@link Token} object representing a token of a particular type; e.g.,
 * `<ID>`. These tokens are created for {@link TagChunk} chunks where the
 * tag corresponds to a lexer rule or token type.
 */
export declare class TokenTagToken extends CommonToken {
    /**
     * This is the backing field for `tokenName`.
     */
    private _tokenName;
    /**
     * This is the backing field for `label`.
     */
    private _label;
    /**
     * Constructs a new instance of {@link TokenTagToken} with the specified
     * token name, type, and label.
     *
     * @param tokenName The token name.
     * @param type The token type.
     * @param label The label associated with the token tag, or `undefined` if
     * the token tag is unlabeled.
     */
    constructor(tokenName: string, type: number, label?: string);
    /**
     * Gets the token name.
     * @returns The token name.
     */
    get tokenName(): string;
    /**
     * Gets the label associated with the rule tag.
     *
     * @returns The name of the label associated with the rule tag, or
     * `undefined` if this is an unlabeled rule tag.
     */
    get label(): string | undefined;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link TokenTagToken} returns the token tag
     * formatted with `<` and `>` delimiters.
     */
    get text(): string;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link TokenTagToken} returns a string of the form
     * `tokenName:type`.
     */
    toString(): string;
}
