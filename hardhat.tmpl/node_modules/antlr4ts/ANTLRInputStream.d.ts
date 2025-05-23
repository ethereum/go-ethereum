/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { CharStream } from "./CharStream";
import { Interval } from "./misc/Interval";
/**
 * Vacuum all input from a {@link Reader}/{@link InputStream} and then treat it
 * like a `char[]` buffer. Can also pass in a {@link String} or
 * `char[]` to use.
 *
 * If you need encoding, pass in stream/reader with correct encoding.
 *
 * @deprecated as of 4.7, please use `CharStreams` interface.
 */
export declare class ANTLRInputStream implements CharStream {
    /** The data being scanned */
    protected data: string;
    /** How many characters are actually in the buffer */
    protected n: number;
    /** 0..n-1 index into string of next char */
    protected p: number;
    /** What is name or source of this char stream? */
    name?: string;
    /** Copy data in string to a local char array */
    constructor(input: string);
    /** Reset the stream so that it's in the same state it was
     *  when the object was created *except* the data array is not
     *  touched.
     */
    reset(): void;
    consume(): void;
    LA(i: number): number;
    LT(i: number): number;
    /** Return the current input symbol index 0..n where n indicates the
     *  last symbol has been read.  The index is the index of char to
     *  be returned from LA(1).
     */
    get index(): number;
    get size(): number;
    /** mark/release do nothing; we have entire buffer */
    mark(): number;
    release(marker: number): void;
    /** consume() ahead until p==index; can't just set p=index as we must
     *  update line and charPositionInLine. If we seek backwards, just set p
     */
    seek(index: number): void;
    getText(interval: Interval): string;
    get sourceName(): string;
    toString(): string;
}
