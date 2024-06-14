"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __decorate = (this && this.__decorate) || function (decorators, target, key, desc) {
    var c = arguments.length, r = c < 3 ? target : desc === null ? desc = Object.getOwnPropertyDescriptor(target, key) : desc, d;
    if (typeof Reflect === "object" && typeof Reflect.decorate === "function") r = Reflect.decorate(decorators, target, key, desc);
    else for (var i = decorators.length - 1; i >= 0; i--) if (d = decorators[i]) r = (c < 3 ? d(r) : c > 3 ? d(target, key, r) : d(target, key)) || r;
    return c > 3 && r && Object.defineProperty(target, key, r), r;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.Lexer = void 0;
const CommonTokenFactory_1 = require("./CommonTokenFactory");
const IntegerStack_1 = require("./misc/IntegerStack");
const Interval_1 = require("./misc/Interval");
const IntStream_1 = require("./IntStream");
const LexerATNSimulator_1 = require("./atn/LexerATNSimulator");
const LexerNoViableAltException_1 = require("./LexerNoViableAltException");
const Decorators_1 = require("./Decorators");
const Recognizer_1 = require("./Recognizer");
const Token_1 = require("./Token");
/** A lexer is recognizer that draws input symbols from a character stream.
 *  lexer grammars result in a subclass of this object. A Lexer object
 *  uses simplified match() and error recovery mechanisms in the interest
 *  of speed.
 */
class Lexer extends Recognizer_1.Recognizer {
    constructor(input) {
        super();
        /** How to create token objects */
        this._factory = CommonTokenFactory_1.CommonTokenFactory.DEFAULT;
        /** What character index in the stream did the current token start at?
         *  Needed, for example, to get the text for current token.  Set at
         *  the start of nextToken.
         */
        this._tokenStartCharIndex = -1;
        /** The line on which the first character of the token resides */
        this._tokenStartLine = 0;
        /** The character position of first character within the line */
        this._tokenStartCharPositionInLine = 0;
        /** Once we see EOF on char stream, next token will be EOF.
         *  If you have DONE : EOF ; then you see DONE EOF.
         */
        this._hitEOF = false;
        /** The channel number for the current token */
        this._channel = 0;
        /** The token type for the current token */
        this._type = 0;
        this._modeStack = new IntegerStack_1.IntegerStack();
        this._mode = Lexer.DEFAULT_MODE;
        this._input = input;
        this._tokenFactorySourcePair = { source: this, stream: input };
    }
    static get DEFAULT_TOKEN_CHANNEL() {
        return Token_1.Token.DEFAULT_CHANNEL;
    }
    static get HIDDEN() {
        return Token_1.Token.HIDDEN_CHANNEL;
    }
    reset(resetInput) {
        // wack Lexer state variables
        if (resetInput === undefined || resetInput) {
            this._input.seek(0); // rewind the input
        }
        this._token = undefined;
        this._type = Token_1.Token.INVALID_TYPE;
        this._channel = Token_1.Token.DEFAULT_CHANNEL;
        this._tokenStartCharIndex = -1;
        this._tokenStartCharPositionInLine = -1;
        this._tokenStartLine = -1;
        this._text = undefined;
        this._hitEOF = false;
        this._mode = Lexer.DEFAULT_MODE;
        this._modeStack.clear();
        this.interpreter.reset();
    }
    /** Return a token from this source; i.e., match a token on the char
     *  stream.
     */
    nextToken() {
        if (this._input == null) {
            throw new Error("nextToken requires a non-null input stream.");
        }
        // Mark start location in char stream so unbuffered streams are
        // guaranteed at least have text of current token
        let tokenStartMarker = this._input.mark();
        try {
            outer: while (true) {
                if (this._hitEOF) {
                    return this.emitEOF();
                }
                this._token = undefined;
                this._channel = Token_1.Token.DEFAULT_CHANNEL;
                this._tokenStartCharIndex = this._input.index;
                this._tokenStartCharPositionInLine = this.interpreter.charPositionInLine;
                this._tokenStartLine = this.interpreter.line;
                this._text = undefined;
                do {
                    this._type = Token_1.Token.INVALID_TYPE;
                    //				System.out.println("nextToken line "+tokenStartLine+" at "+((char)input.LA(1))+
                    //								   " in mode "+mode+
                    //								   " at index "+input.index);
                    let ttype;
                    try {
                        ttype = this.interpreter.match(this._input, this._mode);
                    }
                    catch (e) {
                        if (e instanceof LexerNoViableAltException_1.LexerNoViableAltException) {
                            this.notifyListeners(e); // report error
                            this.recover(e);
                            ttype = Lexer.SKIP;
                        }
                        else {
                            throw e;
                        }
                    }
                    if (this._input.LA(1) === IntStream_1.IntStream.EOF) {
                        this._hitEOF = true;
                    }
                    if (this._type === Token_1.Token.INVALID_TYPE) {
                        this._type = ttype;
                    }
                    if (this._type === Lexer.SKIP) {
                        continue outer;
                    }
                } while (this._type === Lexer.MORE);
                if (this._token == null) {
                    return this.emit();
                }
                return this._token;
            }
        }
        finally {
            // make sure we release marker after match or
            // unbuffered char stream will keep buffering
            this._input.release(tokenStartMarker);
        }
    }
    /** Instruct the lexer to skip creating a token for current lexer rule
     *  and look for another token.  nextToken() knows to keep looking when
     *  a lexer rule finishes with token set to SKIP_TOKEN.  Recall that
     *  if token==undefined at end of any token rule, it creates one for you
     *  and emits it.
     */
    skip() {
        this._type = Lexer.SKIP;
    }
    more() {
        this._type = Lexer.MORE;
    }
    mode(m) {
        this._mode = m;
    }
    pushMode(m) {
        if (LexerATNSimulator_1.LexerATNSimulator.debug) {
            console.log("pushMode " + m);
        }
        this._modeStack.push(this._mode);
        this.mode(m);
    }
    popMode() {
        if (this._modeStack.isEmpty) {
            throw new Error("EmptyStackException");
        }
        if (LexerATNSimulator_1.LexerATNSimulator.debug) {
            console.log("popMode back to " + this._modeStack.peek());
        }
        this.mode(this._modeStack.pop());
        return this._mode;
    }
    get tokenFactory() {
        return this._factory;
    }
    // @Override
    set tokenFactory(factory) {
        this._factory = factory;
    }
    get inputStream() {
        return this._input;
    }
    /** Set the char stream and reset the lexer */
    set inputStream(input) {
        this.reset(false);
        this._input = input;
        this._tokenFactorySourcePair = { source: this, stream: this._input };
    }
    get sourceName() {
        return this._input.sourceName;
    }
    emit(token) {
        if (!token) {
            token = this._factory.create(this._tokenFactorySourcePair, this._type, this._text, this._channel, this._tokenStartCharIndex, this.charIndex - 1, this._tokenStartLine, this._tokenStartCharPositionInLine);
        }
        this._token = token;
        return token;
    }
    emitEOF() {
        let cpos = this.charPositionInLine;
        let line = this.line;
        let eof = this._factory.create(this._tokenFactorySourcePair, Token_1.Token.EOF, undefined, Token_1.Token.DEFAULT_CHANNEL, this._input.index, this._input.index - 1, line, cpos);
        this.emit(eof);
        return eof;
    }
    get line() {
        return this.interpreter.line;
    }
    set line(line) {
        this.interpreter.line = line;
    }
    get charPositionInLine() {
        return this.interpreter.charPositionInLine;
    }
    set charPositionInLine(charPositionInLine) {
        this.interpreter.charPositionInLine = charPositionInLine;
    }
    /** What is the index of the current character of lookahead? */
    get charIndex() {
        return this._input.index;
    }
    /** Return the text matched so far for the current token or any
     *  text override.
     */
    get text() {
        if (this._text != null) {
            return this._text;
        }
        return this.interpreter.getText(this._input);
    }
    /** Set the complete text of this token; it wipes any previous
     *  changes to the text.
     */
    set text(text) {
        this._text = text;
    }
    /** Override if emitting multiple tokens. */
    get token() { return this._token; }
    set token(_token) {
        this._token = _token;
    }
    set type(ttype) {
        this._type = ttype;
    }
    get type() {
        return this._type;
    }
    set channel(channel) {
        this._channel = channel;
    }
    get channel() {
        return this._channel;
    }
    /** Return a list of all Token objects in input char stream.
     *  Forces load of all tokens. Does not include EOF token.
     */
    getAllTokens() {
        let tokens = [];
        let t = this.nextToken();
        while (t.type !== Token_1.Token.EOF) {
            tokens.push(t);
            t = this.nextToken();
        }
        return tokens;
    }
    notifyListeners(e) {
        let text = this._input.getText(Interval_1.Interval.of(this._tokenStartCharIndex, this._input.index));
        let msg = "token recognition error at: '" +
            this.getErrorDisplay(text) + "'";
        let listener = this.getErrorListenerDispatch();
        if (listener.syntaxError) {
            listener.syntaxError(this, undefined, this._tokenStartLine, this._tokenStartCharPositionInLine, msg, e);
        }
    }
    getErrorDisplay(s) {
        if (typeof s === "number") {
            switch (s) {
                case Token_1.Token.EOF:
                    return "<EOF>";
                case 0x0a:
                    return "\\n";
                case 0x09:
                    return "\\t";
                case 0x0d:
                    return "\\r";
            }
            return String.fromCharCode(s);
        }
        return s.replace(/\n/g, "\\n")
            .replace(/\t/g, "\\t")
            .replace(/\r/g, "\\r");
    }
    getCharErrorDisplay(c) {
        let s = this.getErrorDisplay(c);
        return "'" + s + "'";
    }
    recover(re) {
        if (re instanceof LexerNoViableAltException_1.LexerNoViableAltException) {
            if (this._input.LA(1) !== IntStream_1.IntStream.EOF) {
                // skip a char and try again
                this.interpreter.consume(this._input);
            }
        }
        else {
            //System.out.println("consuming char "+(char)input.LA(1)+" during recovery");
            //re.printStackTrace();
            // TODO: Do we lose character or line position information?
            this._input.consume();
        }
    }
}
Lexer.DEFAULT_MODE = 0;
Lexer.MORE = -2;
Lexer.SKIP = -3;
Lexer.MIN_CHAR_VALUE = 0x0000;
Lexer.MAX_CHAR_VALUE = 0x10FFFF;
__decorate([
    Decorators_1.Override
], Lexer.prototype, "nextToken", null);
__decorate([
    Decorators_1.Override
], Lexer.prototype, "tokenFactory", null);
__decorate([
    Decorators_1.Override
], Lexer.prototype, "inputStream", null);
__decorate([
    Decorators_1.Override
], Lexer.prototype, "sourceName", null);
__decorate([
    Decorators_1.Override
], Lexer.prototype, "line", null);
__decorate([
    Decorators_1.Override
], Lexer.prototype, "charPositionInLine", null);
exports.Lexer = Lexer;
//# sourceMappingURL=Lexer.js.map