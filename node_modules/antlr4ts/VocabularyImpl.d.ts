/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Vocabulary } from "./Vocabulary";
/**
 * This class provides a default implementation of the {@link Vocabulary}
 * interface.
 *
 * @author Sam Harwell
 */
export declare class VocabularyImpl implements Vocabulary {
    /**
     * Gets an empty {@link Vocabulary} instance.
     *
     * No literal or symbol names are assigned to token types, so
     * {@link #getDisplayName(int)} returns the numeric value for all tokens
     * except {@link Token#EOF}.
     */
    static readonly EMPTY_VOCABULARY: VocabularyImpl;
    private readonly literalNames;
    private readonly symbolicNames;
    private readonly displayNames;
    private _maxTokenType;
    /**
     * Constructs a new instance of {@link VocabularyImpl} from the specified
     * literal, symbolic, and display token names.
     *
     * @param literalNames The literal names assigned to tokens, or an empty array
     * if no literal names are assigned.
     * @param symbolicNames The symbolic names assigned to tokens, or
     * an empty array if no symbolic names are assigned.
     * @param displayNames The display names assigned to tokens, or an empty array
     * to use the values in `literalNames` and `symbolicNames` as
     * the source of display names, as described in
     * {@link #getDisplayName(int)}.
     *
     * @see #getLiteralName(int)
     * @see #getSymbolicName(int)
     * @see #getDisplayName(int)
     */
    constructor(literalNames: Array<string | undefined>, symbolicNames: Array<string | undefined>, displayNames: Array<string | undefined>);
    get maxTokenType(): number;
    getLiteralName(tokenType: number): string | undefined;
    getSymbolicName(tokenType: number): string | undefined;
    getDisplayName(tokenType: number): string;
}
