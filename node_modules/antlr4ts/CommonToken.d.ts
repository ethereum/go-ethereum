/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { ATNSimulator } from "./atn/ATNSimulator";
import { CharStream } from "./CharStream";
import { Recognizer } from "./Recognizer";
import { Token } from "./Token";
import { TokenSource } from "./TokenSource";
import { WritableToken } from "./WritableToken";
export declare class CommonToken implements WritableToken {
    /**
     * An empty {@link Tuple2} which is used as the default value of
     * {@link #source} for tokens that do not have a source.
     */
    protected static readonly EMPTY_SOURCE: {
        source?: TokenSource;
        stream?: CharStream;
    };
    /**
     * This is the backing field for `type`.
     */
    private _type;
    /**
     * This is the backing field for {@link #getLine} and {@link #setLine}.
     */
    private _line;
    /**
     * This is the backing field for {@link #getCharPositionInLine} and
     * {@link #setCharPositionInLine}.
     */
    private _charPositionInLine;
    /**
     * This is the backing field for {@link #getChannel} and
     * {@link #setChannel}.
     */
    private _channel;
    /**
     * This is the backing field for {@link #getTokenSource} and
     * {@link #getInputStream}.
     *
     * These properties share a field to reduce the memory footprint of
     * {@link CommonToken}. Tokens created by a {@link CommonTokenFactory} from
     * the same source and input stream share a reference to the same
     * {@link Tuple2} containing these values.
     */
    protected source: {
        source?: TokenSource;
        stream?: CharStream;
    };
    /**
     * This is the backing field for {@link #getText} when the token text is
     * explicitly set in the constructor or via {@link #setText}.
     *
     * @see `text`
     */
    private _text?;
    /**
     * This is the backing field for `tokenIndex`.
     */
    protected index: number;
    /**
     * This is the backing field for `startIndex`.
     */
    protected start: number;
    /**
     * This is the backing field for `stopIndex`.
     */
    private stop;
    constructor(type: number, text?: string, source?: {
        source?: TokenSource;
        stream?: CharStream;
    }, channel?: number, start?: number, stop?: number);
    /**
     * Constructs a new {@link CommonToken} as a copy of another {@link Token}.
     *
     * If `oldToken` is also a {@link CommonToken} instance, the newly
     * constructed token will share a reference to the {@link #text} field and
     * the {@link Tuple2} stored in {@link #source}. Otherwise, {@link #text} will
     * be assigned the result of calling {@link #getText}, and {@link #source}
     * will be constructed from the result of {@link Token#getTokenSource} and
     * {@link Token#getInputStream}.
     *
     * @param oldToken The token to copy.
     */
    static fromToken(oldToken: Token): CommonToken;
    get type(): number;
    set type(type: number);
    get line(): number;
    set line(line: number);
    get text(): string | undefined;
    /**
     * Explicitly set the text for this token. If {code text} is not
     * `undefined`, then {@link #getText} will return this value rather than
     * extracting the text from the input.
     *
     * @param text The explicit text of the token, or `undefined` if the text
     * should be obtained from the input along with the start and stop indexes
     * of the token.
     */
    set text(text: string | undefined);
    get charPositionInLine(): number;
    set charPositionInLine(charPositionInLine: number);
    get channel(): number;
    set channel(channel: number);
    get startIndex(): number;
    set startIndex(start: number);
    get stopIndex(): number;
    set stopIndex(stop: number);
    get tokenIndex(): number;
    set tokenIndex(index: number);
    get tokenSource(): TokenSource | undefined;
    get inputStream(): CharStream | undefined;
    toString(): string;
    toString<TSymbol, ATNInterpreter extends ATNSimulator>(recognizer: Recognizer<TSymbol, ATNInterpreter> | undefined): string;
}
