/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { CharStream } from "./CharStream";
import { CodePointBuffer } from "./CodePointBuffer";
import { Interval } from "./misc/Interval";
/**
 * Alternative to {@link ANTLRInputStream} which treats the input
 * as a series of Unicode code points, instead of a series of UTF-16
 * code units.
 *
 * Use this if you need to parse input which potentially contains
 * Unicode values > U+FFFF.
 */
export declare class CodePointCharStream implements CharStream {
    private readonly _array;
    private readonly _size;
    private readonly _name;
    private _position;
    protected constructor(array: Uint8Array | Uint16Array | Int32Array, position: number, remaining: number, name: string);
    get internalStorage(): Uint8Array | Uint16Array | Int32Array;
    /**
     * Constructs a {@link CodePointCharStream} which provides access
     * to the Unicode code points stored in {@code codePointBuffer}.
     */
    static fromBuffer(codePointBuffer: CodePointBuffer): CodePointCharStream;
    /**
     * Constructs a named {@link CodePointCharStream} which provides access
     * to the Unicode code points stored in {@code codePointBuffer}.
     */
    static fromBuffer(codePointBuffer: CodePointBuffer, name: string): CodePointCharStream;
    consume(): void;
    get index(): number;
    get size(): number;
    /** mark/release do nothing; we have entire buffer */
    mark(): number;
    release(marker: number): void;
    seek(index: number): void;
    get sourceName(): string;
    toString(): string;
    LA(i: number): number;
    /** Return the UTF-16 encoded string for the given interval */
    getText(interval: Interval): string;
}
