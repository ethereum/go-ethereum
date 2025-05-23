/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
import { CharStream } from "./CharStream";
import { IntegerStack } from "./misc/IntegerStack";
import { LexerATNSimulator } from "./atn/LexerATNSimulator";
import { LexerNoViableAltException } from "./LexerNoViableAltException";
import { RecognitionException } from "./RecognitionException";
import { Recognizer } from "./Recognizer";
import { Token } from "./Token";
import { TokenFactory } from "./TokenFactory";
import { TokenSource } from "./TokenSource";
/** A lexer is recognizer that draws input symbols from a character stream.
 *  lexer grammars result in a subclass of this object. A Lexer object
 *  uses simplified match() and error recovery mechanisms in the interest
 *  of speed.
 */
export declare abstract class Lexer extends Recognizer<number, LexerATNSimulator> implements TokenSource {
    static readonly DEFAULT_MODE: number;
    static readonly MORE: number;
    static readonly SKIP: number;
    static get DEFAULT_TOKEN_CHANNEL(): number;
    static get HIDDEN(): number;
    static readonly MIN_CHAR_VALUE: number;
    static readonly MAX_CHAR_VALUE: number;
    _input: CharStream;
    protected _tokenFactorySourcePair: {
        source: TokenSource;
        stream: CharStream;
    };
    /** How to create token objects */
    protected _factory: TokenFactory;
    /** The goal of all lexer rules/methods is to create a token object.
     *  This is an instance variable as multiple rules may collaborate to
     *  create a single token.  nextToken will return this object after
     *  matching lexer rule(s).  If you subclass to allow multiple token
     *  emissions, then set this to the last token to be matched or
     *  something non-undefined so that the auto token emit mechanism will not
     *  emit another token.
     */
    _token: Token | undefined;
    /** What character index in the stream did the current token start at?
     *  Needed, for example, to get the text for current token.  Set at
     *  the start of nextToken.
     */
    _tokenStartCharIndex: number;
    /** The line on which the first character of the token resides */
    _tokenStartLine: number;
    /** The character position of first character within the line */
    _tokenStartCharPositionInLine: number;
    /** Once we see EOF on char stream, next token will be EOF.
     *  If you have DONE : EOF ; then you see DONE EOF.
     */
    _hitEOF: boolean;
    /** The channel number for the current token */
    _channel: number;
    /** The token type for the current token */
    _type: number;
    readonly _modeStack: IntegerStack;
    _mode: number;
    /** You can set the text for the current token to override what is in
     *  the input char buffer.  Set `text` or can set this instance var.
     */
    _text: string | undefined;
    constructor(input: CharStream);
    reset(): void;
    reset(resetInput: boolean): void;
    /** Return a token from this source; i.e., match a token on the char
     *  stream.
     */
    nextToken(): Token;
    /** Instruct the lexer to skip creating a token for current lexer rule
     *  and look for another token.  nextToken() knows to keep looking when
     *  a lexer rule finishes with token set to SKIP_TOKEN.  Recall that
     *  if token==undefined at end of any token rule, it creates one for you
     *  and emits it.
     */
    skip(): void;
    more(): void;
    mode(m: number): void;
    pushMode(m: number): void;
    popMode(): number;
    get tokenFactory(): TokenFactory;
    set tokenFactory(factory: TokenFactory);
    get inputStream(): CharStream;
    /** Set the char stream and reset the lexer */
    set inputStream(input: CharStream);
    get sourceName(): string;
    /** The standard method called to automatically emit a token at the
     *  outermost lexical rule.  The token object should point into the
     *  char buffer start..stop.  If there is a text override in 'text',
     *  use that to set the token's text.  Override this method to emit
     *  custom Token objects or provide a new factory.
     */
    emit(token: Token): Token;
    /** By default does not support multiple emits per nextToken invocation
     *  for efficiency reasons.  Subclass and override this method, nextToken,
     *  and getToken (to push tokens into a list and pull from that list
     *  rather than a single variable as this implementation does).
     */
    emit(): Token;
    emitEOF(): Token;
    get line(): number;
    set line(line: number);
    get charPositionInLine(): number;
    set charPositionInLine(charPositionInLine: number);
    /** What is the index of the current character of lookahead? */
    get charIndex(): number;
    /** Return the text matched so far for the current token or any
     *  text override.
     */
    get text(): string;
    /** Set the complete text of this token; it wipes any previous
     *  changes to the text.
     */
    set text(text: string);
    /** Override if emitting multiple tokens. */
    get token(): Token | undefined;
    set token(_token: Token | undefined);
    set type(ttype: number);
    get type(): number;
    set channel(channel: number);
    get channel(): number;
    abstract readonly channelNames: string[];
    abstract readonly modeNames: string[];
    /** Return a list of all Token objects in input char stream.
     *  Forces load of all tokens. Does not include EOF token.
     */
    getAllTokens(): Token[];
    notifyListeners(e: LexerNoViableAltException): void;
    getErrorDisplay(s: string | number): string;
    getCharErrorDisplay(c: number): string;
    /** Lexers can normally match any char in it's vocabulary after matching
     *  a token, so do the easy thing and just kill a character and hope
     *  it all works out.  You can instead use the rule invocation stack
     *  to do sophisticated error recovery if you are in a fragment rule.
     */
    recover(re: RecognitionException): void;
    recover(re: LexerNoViableAltException): void;
}
