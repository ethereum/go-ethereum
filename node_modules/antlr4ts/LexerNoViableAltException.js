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
exports.LexerNoViableAltException = void 0;
const RecognitionException_1 = require("./RecognitionException");
const Decorators_1 = require("./Decorators");
const Interval_1 = require("./misc/Interval");
const Utils = require("./misc/Utils");
let LexerNoViableAltException = class LexerNoViableAltException extends RecognitionException_1.RecognitionException {
    constructor(lexer, input, startIndex, deadEndConfigs) {
        super(lexer, input);
        this._startIndex = startIndex;
        this._deadEndConfigs = deadEndConfigs;
    }
    get startIndex() {
        return this._startIndex;
    }
    get deadEndConfigs() {
        return this._deadEndConfigs;
    }
    get inputStream() {
        return super.inputStream;
    }
    toString() {
        let symbol = "";
        if (this._startIndex >= 0 && this._startIndex < this.inputStream.size) {
            symbol = this.inputStream.getText(Interval_1.Interval.of(this._startIndex, this._startIndex));
            symbol = Utils.escapeWhitespace(symbol, false);
        }
        // return String.format(Locale.getDefault(), "%s('%s')", LexerNoViableAltException.class.getSimpleName(), symbol);
        return `LexerNoViableAltException('${symbol}')`;
    }
};
__decorate([
    Decorators_1.Override
], LexerNoViableAltException.prototype, "inputStream", null);
__decorate([
    Decorators_1.Override
], LexerNoViableAltException.prototype, "toString", null);
LexerNoViableAltException = __decorate([
    __param(1, Decorators_1.NotNull)
], LexerNoViableAltException);
exports.LexerNoViableAltException = LexerNoViableAltException;
//# sourceMappingURL=LexerNoViableAltException.js.map