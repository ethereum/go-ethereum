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
exports.FailedPredicateException = void 0;
const RecognitionException_1 = require("./RecognitionException");
const Decorators_1 = require("./Decorators");
const PredicateTransition_1 = require("./atn/PredicateTransition");
/** A semantic predicate failed during validation.  Validation of predicates
 *  occurs when normally parsing the alternative just like matching a token.
 *  Disambiguating predicate evaluation occurs when we test a predicate during
 *  prediction.
 */
let FailedPredicateException = class FailedPredicateException extends RecognitionException_1.RecognitionException {
    constructor(recognizer, predicate, message) {
        super(recognizer, recognizer.inputStream, recognizer.context, FailedPredicateException.formatMessage(predicate, message));
        let s = recognizer.interpreter.atn.states[recognizer.state];
        let trans = s.transition(0);
        if (trans instanceof PredicateTransition_1.PredicateTransition) {
            this._ruleIndex = trans.ruleIndex;
            this._predicateIndex = trans.predIndex;
        }
        else {
            this._ruleIndex = 0;
            this._predicateIndex = 0;
        }
        this._predicate = predicate;
        super.setOffendingToken(recognizer, recognizer.currentToken);
    }
    get ruleIndex() {
        return this._ruleIndex;
    }
    get predicateIndex() {
        return this._predicateIndex;
    }
    get predicate() {
        return this._predicate;
    }
    static formatMessage(predicate, message) {
        if (message) {
            return message;
        }
        return `failed predicate: {${predicate}}?`;
    }
};
__decorate([
    Decorators_1.NotNull
], FailedPredicateException, "formatMessage", null);
FailedPredicateException = __decorate([
    __param(0, Decorators_1.NotNull)
], FailedPredicateException);
exports.FailedPredicateException = FailedPredicateException;
//# sourceMappingURL=FailedPredicateException.js.map