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
exports.NoViableAltException = void 0;
const Parser_1 = require("./Parser");
const RecognitionException_1 = require("./RecognitionException");
const Decorators_1 = require("./Decorators");
/** Indicates that the parser could not decide which of two or more paths
 *  to take based upon the remaining input. It tracks the starting token
 *  of the offending input and also knows where the parser was
 *  in the various paths when the error. Reported by reportNoViableAlternative()
 */
class NoViableAltException extends RecognitionException_1.RecognitionException {
    constructor(recognizer, input, startToken, offendingToken, deadEndConfigs, ctx) {
        if (recognizer instanceof Parser_1.Parser) {
            if (input === undefined) {
                input = recognizer.inputStream;
            }
            if (startToken === undefined) {
                startToken = recognizer.currentToken;
            }
            if (offendingToken === undefined) {
                offendingToken = recognizer.currentToken;
            }
            if (ctx === undefined) {
                ctx = recognizer.context;
            }
        }
        super(recognizer, input, ctx);
        this._deadEndConfigs = deadEndConfigs;
        this._startToken = startToken;
        this.setOffendingToken(recognizer, offendingToken);
    }
    get startToken() {
        return this._startToken;
    }
    get deadEndConfigs() {
        return this._deadEndConfigs;
    }
}
__decorate([
    Decorators_1.NotNull
], NoViableAltException.prototype, "_startToken", void 0);
exports.NoViableAltException = NoViableAltException;
//# sourceMappingURL=NoViableAltException.js.map