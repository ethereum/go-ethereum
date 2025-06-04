/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { CharStream } from "../../CharStream";
import { Token } from "../../Token";
import { TokenSource } from "../../TokenSource";
/**
 * A {@link Token} object representing an entire subtree matched by a parser
 * rule; e.g., `<expr>`. These tokens are created for {@link TagChunk}
 * chunks where the tag corresponds to a parser rule.
 */
export declare class RuleTagToken implements Token {
    /**
     * This is the backing field for `ruleName`.
     */
    private _ruleName;
    /**
     * The token type for the current token. This is the token type assigned to
     * the bypass alternative for the rule during ATN deserialization.
     */
    private bypassTokenType;
    /**
     * This is the backing field for `label`.
     */
    private _label?;
    /**
     * Constructs a new instance of {@link RuleTagToken} with the specified rule
     * name, bypass token type, and label.
     *
     * @param ruleName The name of the parser rule this rule tag matches.
     * @param bypassTokenType The bypass token type assigned to the parser rule.
     * @param label The label associated with the rule tag, or `undefined` if
     * the rule tag is unlabeled.
     *
     * @exception IllegalArgumentException if `ruleName` is not defined
     * or empty.
     */
    constructor(ruleName: string, bypassTokenType: number, label?: string);
    /**
     * Gets the name of the rule associated with this rule tag.
     *
     * @returns The name of the parser rule associated with this rule tag.
     */
    get ruleName(): string;
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
     * Rule tag tokens are always placed on the {@link #DEFAULT_CHANNEL}.
     */
    get channel(): number;
    /**
     * {@inheritDoc}
     *
     * This method returns the rule tag formatted with `<` and `>`
     * delimiters.
     */
    get text(): string;
    /**
     * {@inheritDoc}
     *
     * Rule tag tokens have types assigned according to the rule bypass
     * transitions created during ATN deserialization.
     */
    get type(): number;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link RuleTagToken} always returns 0.
     */
    get line(): number;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link RuleTagToken} always returns -1.
     */
    get charPositionInLine(): number;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link RuleTagToken} always returns -1.
     */
    get tokenIndex(): number;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link RuleTagToken} always returns -1.
     */
    get startIndex(): number;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link RuleTagToken} always returns -1.
     */
    get stopIndex(): number;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link RuleTagToken} always returns `undefined`.
     */
    get tokenSource(): TokenSource | undefined;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link RuleTagToken} always returns `undefined`.
     */
    get inputStream(): CharStream | undefined;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link RuleTagToken} returns a string of the form
     * `ruleName:bypassTokenType`.
     */
    toString(): string;
}
