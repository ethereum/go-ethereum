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
var __param = (this && this.__param) || function (paramIndex, decorator) {
    return function (target, key) { decorator(target, key, paramIndex); }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.CommonTokenStream = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:50.3953157-07:00
const BufferedTokenStream_1 = require("./BufferedTokenStream");
const Decorators_1 = require("./Decorators");
const Token_1 = require("./Token");
/**
 * This class extends {@link BufferedTokenStream} with functionality to filter
 * token streams to tokens on a particular channel (tokens where
 * {@link Token#getChannel} returns a particular value).
 *
 * This token stream provides access to all tokens by index or when calling
 * methods like {@link #getText}. The channel filtering is only used for code
 * accessing tokens via the lookahead methods {@link #LA}, {@link #LT}, and
 * {@link #LB}.
 *
 * By default, tokens are placed on the default channel
 * ({@link Token#DEFAULT_CHANNEL}), but may be reassigned by using the
 * `->channel(HIDDEN)` lexer command, or by using an embedded action to
 * call {@link Lexer#setChannel}.
 *
 * Note: lexer rules which use the `->skip` lexer command or call
 * {@link Lexer#skip} do not produce tokens at all, so input text matched by
 * such a rule will not be available as part of the token stream, regardless of
 * channel.
 */
let CommonTokenStream = class CommonTokenStream extends BufferedTokenStream_1.BufferedTokenStream {
    /**
     * Constructs a new {@link CommonTokenStream} using the specified token
     * source and filtering tokens to the specified channel. Only tokens whose
     * {@link Token#getChannel} matches `channel` or have the
     * `Token.type` equal to {@link Token#EOF} will be returned by the
     * token stream lookahead methods.
     *
     * @param tokenSource The token source.
     * @param channel The channel to use for filtering tokens.
     */
    constructor(tokenSource, channel = Token_1.Token.DEFAULT_CHANNEL) {
        super(tokenSource);
        this.channel = channel;
    }
    adjustSeekIndex(i) {
        return this.nextTokenOnChannel(i, this.channel);
    }
    tryLB(k) {
        if ((this.p - k) < 0) {
            return undefined;
        }
        let i = this.p;
        let n = 1;
        // find k good tokens looking backwards
        while (n <= k && i > 0) {
            // skip off-channel tokens
            i = this.previousTokenOnChannel(i - 1, this.channel);
            n++;
        }
        if (i < 0) {
            return undefined;
        }
        return this.tokens[i];
    }
    tryLT(k) {
        //System.out.println("enter LT("+k+")");
        this.lazyInit();
        if (k === 0) {
            throw new RangeError("0 is not a valid lookahead index");
        }
        if (k < 0) {
            return this.tryLB(-k);
        }
        let i = this.p;
        let n = 1; // we know tokens[p] is a good one
        // find k good tokens
        while (n < k) {
            // skip off-channel tokens, but make sure to not look past EOF
            if (this.sync(i + 1)) {
                i = this.nextTokenOnChannel(i + 1, this.channel);
            }
            n++;
        }
        //		if ( i>range ) range = i;
        return this.tokens[i];
    }
    /** Count EOF just once. */
    getNumberOfOnChannelTokens() {
        let n = 0;
        this.fill();
        for (let t of this.tokens) {
            if (t.channel === this.channel) {
                n++;
            }
            if (t.type === Token_1.Token.EOF) {
                break;
            }
        }
        return n;
    }
};
__decorate([
    Decorators_1.Override
], CommonTokenStream.prototype, "adjustSeekIndex", null);
__decorate([
    Decorators_1.Override
], CommonTokenStream.prototype, "tryLB", null);
__decorate([
    Decorators_1.Override
], CommonTokenStream.prototype, "tryLT", null);
CommonTokenStream = __decorate([
    __param(0, Decorators_1.NotNull)
], CommonTokenStream);
exports.CommonTokenStream = CommonTokenStream;
//# sourceMappingURL=CommonTokenStream.js.map