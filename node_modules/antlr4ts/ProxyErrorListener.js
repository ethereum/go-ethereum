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
exports.ProxyErrorListener = void 0;
const Decorators_1 = require("./Decorators");
/**
 * This implementation of {@link ANTLRErrorListener} dispatches all calls to a
 * collection of delegate listeners. This reduces the effort required to support multiple
 * listeners.
 *
 * @author Sam Harwell
 */
class ProxyErrorListener {
    constructor(delegates) {
        this.delegates = delegates;
        if (!delegates) {
            throw new Error("Invalid delegates");
        }
    }
    getDelegates() {
        return this.delegates;
    }
    syntaxError(recognizer, offendingSymbol, line, charPositionInLine, msg, e) {
        this.delegates.forEach((listener) => {
            if (listener.syntaxError) {
                listener.syntaxError(recognizer, offendingSymbol, line, charPositionInLine, msg, e);
            }
        });
    }
}
__decorate([
    Decorators_1.Override,
    __param(0, Decorators_1.NotNull),
    __param(4, Decorators_1.NotNull)
], ProxyErrorListener.prototype, "syntaxError", null);
exports.ProxyErrorListener = ProxyErrorListener;
//# sourceMappingURL=ProxyErrorListener.js.map