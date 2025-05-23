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
exports.ProxyParserErrorListener = void 0;
const ProxyErrorListener_1 = require("./ProxyErrorListener");
const Decorators_1 = require("./Decorators");
/**
 * @author Sam Harwell
 */
class ProxyParserErrorListener extends ProxyErrorListener_1.ProxyErrorListener {
    constructor(delegates) {
        super(delegates);
    }
    reportAmbiguity(recognizer, dfa, startIndex, stopIndex, exact, ambigAlts, configs) {
        this.getDelegates()
            .forEach((listener) => {
            if (listener.reportAmbiguity) {
                listener.reportAmbiguity(recognizer, dfa, startIndex, stopIndex, exact, ambigAlts, configs);
            }
        });
    }
    reportAttemptingFullContext(recognizer, dfa, startIndex, stopIndex, conflictingAlts, conflictState) {
        this.getDelegates()
            .forEach((listener) => {
            if (listener.reportAttemptingFullContext) {
                listener.reportAttemptingFullContext(recognizer, dfa, startIndex, stopIndex, conflictingAlts, conflictState);
            }
        });
    }
    reportContextSensitivity(recognizer, dfa, startIndex, stopIndex, prediction, acceptState) {
        this.getDelegates()
            .forEach((listener) => {
            if (listener.reportContextSensitivity) {
                listener.reportContextSensitivity(recognizer, dfa, startIndex, stopIndex, prediction, acceptState);
            }
        });
    }
}
__decorate([
    Decorators_1.Override
], ProxyParserErrorListener.prototype, "reportAmbiguity", null);
__decorate([
    Decorators_1.Override
], ProxyParserErrorListener.prototype, "reportAttemptingFullContext", null);
__decorate([
    Decorators_1.Override
], ProxyParserErrorListener.prototype, "reportContextSensitivity", null);
exports.ProxyParserErrorListener = ProxyParserErrorListener;
//# sourceMappingURL=ProxyParserErrorListener.js.map