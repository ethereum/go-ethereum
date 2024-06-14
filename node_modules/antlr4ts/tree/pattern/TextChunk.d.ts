/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { Chunk } from "./Chunk";
/**
 * Represents a span of raw text (concrete syntax) between tags in a tree
 * pattern string.
 */
export declare class TextChunk extends Chunk {
    /**
     * This is the backing field for {@link #getText}.
     */
    private _text;
    /**
     * Constructs a new instance of {@link TextChunk} with the specified text.
     *
     * @param text The text of this chunk.
     * @exception IllegalArgumentException if `text` is not defined.
     */
    constructor(text: string);
    /**
     * Gets the raw text of this chunk.
     *
     * @returns The text of the chunk.
     */
    get text(): string;
    /**
     * {@inheritDoc}
     *
     * The implementation for {@link TextChunk} returns the result of
     * `text` in single quotes.
     */
    toString(): string;
}
