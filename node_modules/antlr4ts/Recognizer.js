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
exports.Recognizer = void 0;
const ConsoleErrorListener_1 = require("./ConsoleErrorListener");
const ProxyErrorListener_1 = require("./ProxyErrorListener");
const Decorators_1 = require("./Decorators");
const Token_1 = require("./Token");
const Utils = require("./misc/Utils");
class Recognizer {
    constructor() {
        this._listeners = [ConsoleErrorListener_1.ConsoleErrorListener.INSTANCE];
        this._stateNumber = -1;
    }
    /**
     * Get a map from token names to token types.
     *
     * Used for XPath and tree pattern compilation.
     */
    getTokenTypeMap() {
        let vocabulary = this.vocabulary;
        let result = Recognizer.tokenTypeMapCache.get(vocabulary);
        if (result == null) {
            let intermediateResult = new Map();
            for (let i = 0; i <= this.atn.maxTokenType; i++) {
                let literalName = vocabulary.getLiteralName(i);
                if (literalName != null) {
                    intermediateResult.set(literalName, i);
                }
                let symbolicName = vocabulary.getSymbolicName(i);
                if (symbolicName != null) {
                    intermediateResult.set(symbolicName, i);
                }
            }
            intermediateResult.set("EOF", Token_1.Token.EOF);
            result = intermediateResult;
            Recognizer.tokenTypeMapCache.set(vocabulary, result);
        }
        return result;
    }
    /**
     * Get a map from rule names to rule indexes.
     *
     * Used for XPath and tree pattern compilation.
     */
    getRuleIndexMap() {
        let ruleNames = this.ruleNames;
        if (ruleNames == null) {
            throw new Error("The current recognizer does not provide a list of rule names.");
        }
        let result = Recognizer.ruleIndexMapCache.get(ruleNames);
        if (result == null) {
            result = Utils.toMap(ruleNames);
            Recognizer.ruleIndexMapCache.set(ruleNames, result);
        }
        return result;
    }
    getTokenType(tokenName) {
        let ttype = this.getTokenTypeMap().get(tokenName);
        if (ttype != null) {
            return ttype;
        }
        return Token_1.Token.INVALID_TYPE;
    }
    /**
     * If this recognizer was generated, it will have a serialized ATN
     * representation of the grammar.
     *
     * For interpreters, we don't know their serialized ATN despite having
     * created the interpreter from it.
     */
    get serializedATN() {
        throw new Error("there is no serialized ATN");
    }
    /**
     * Get the {@link ATN} used by the recognizer for prediction.
     *
     * @returns The {@link ATN} used by the recognizer for prediction.
     */
    get atn() {
        return this._interp.atn;
    }
    /**
     * Get the ATN interpreter used by the recognizer for prediction.
     *
     * @returns The ATN interpreter used by the recognizer for prediction.
     */
    get interpreter() {
        return this._interp;
    }
    /**
     * Set the ATN interpreter used by the recognizer for prediction.
     *
     * @param interpreter The ATN interpreter used by the recognizer for
     * prediction.
     */
    set interpreter(interpreter) {
        this._interp = interpreter;
    }
    /** If profiling during the parse/lex, this will return DecisionInfo records
     *  for each decision in recognizer in a ParseInfo object.
     *
     * @since 4.3
     */
    get parseInfo() {
        return Promise.resolve(undefined);
    }
    /** What is the error header, normally line/character position information? */
    getErrorHeader(e) {
        let token = e.getOffendingToken();
        if (!token) {
            return "";
        }
        let line = token.line;
        let charPositionInLine = token.charPositionInLine;
        return "line " + line + ":" + charPositionInLine;
    }
    /**
     * @exception NullPointerException if `listener` is `undefined`.
     */
    addErrorListener(listener) {
        if (!listener) {
            throw new TypeError("listener must not be null");
        }
        this._listeners.push(listener);
    }
    removeErrorListener(listener) {
        let position = this._listeners.indexOf(listener);
        if (position !== -1) {
            this._listeners.splice(position, 1);
        }
    }
    removeErrorListeners() {
        this._listeners.length = 0;
    }
    getErrorListeners() {
        return this._listeners.slice(0);
    }
    getErrorListenerDispatch() {
        return new ProxyErrorListener_1.ProxyErrorListener(this.getErrorListeners());
    }
    // subclass needs to override these if there are sempreds or actions
    // that the ATN interp needs to execute
    sempred(_localctx, ruleIndex, actionIndex) {
        return true;
    }
    precpred(localctx, precedence) {
        return true;
    }
    action(_localctx, ruleIndex, actionIndex) {
        // intentionally empty
    }
    get state() {
        return this._stateNumber;
    }
    /** Indicate that the recognizer has changed internal state that is
     *  consistent with the ATN state passed in.  This way we always know
     *  where we are in the ATN as the parser goes along. The rule
     *  context objects form a stack that lets us see the stack of
     *  invoking rules. Combine this and we have complete ATN
     *  configuration information.
     */
    set state(atnState) {
        //		System.err.println("setState "+atnState);
        this._stateNumber = atnState;
        //		if ( traceATNStates ) _ctx.trace(atnState);
    }
}
Recognizer.EOF = -1;
Recognizer.tokenTypeMapCache = new WeakMap();
Recognizer.ruleIndexMapCache = new WeakMap();
__decorate([
    Decorators_1.SuppressWarnings("serial"),
    Decorators_1.NotNull
], Recognizer.prototype, "_listeners", void 0);
__decorate([
    Decorators_1.NotNull
], Recognizer.prototype, "getTokenTypeMap", null);
__decorate([
    Decorators_1.NotNull
], Recognizer.prototype, "getRuleIndexMap", null);
__decorate([
    Decorators_1.NotNull
], Recognizer.prototype, "serializedATN", null);
__decorate([
    Decorators_1.NotNull
], Recognizer.prototype, "atn", null);
__decorate([
    Decorators_1.NotNull,
    __param(0, Decorators_1.NotNull)
], Recognizer.prototype, "interpreter", null);
__decorate([
    Decorators_1.NotNull,
    __param(0, Decorators_1.NotNull)
], Recognizer.prototype, "getErrorHeader", null);
__decorate([
    __param(0, Decorators_1.NotNull)
], Recognizer.prototype, "addErrorListener", null);
__decorate([
    __param(0, Decorators_1.NotNull)
], Recognizer.prototype, "removeErrorListener", null);
__decorate([
    Decorators_1.NotNull
], Recognizer.prototype, "getErrorListeners", null);
exports.Recognizer = Recognizer;
//# sourceMappingURL=Recognizer.js.map