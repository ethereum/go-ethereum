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
exports.InputMismatchException = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:51.5187682-07:00
const RecognitionException_1 = require("./RecognitionException");
const Decorators_1 = require("./Decorators");
/** This signifies any kind of mismatched input exceptions such as
 *  when the current input does not match the expected token.
 */
let InputMismatchException = class InputMismatchException extends RecognitionException_1.RecognitionException {
    constructor(recognizer, state, context) {
        if (context === undefined) {
            context = recognizer.context;
        }
        super(recognizer, recognizer.inputStream, context);
        if (state !== undefined) {
            this.setOffendingState(state);
        }
        this.setOffendingToken(recognizer, recognizer.currentToken);
    }
};
InputMismatchException = __decorate([
    __param(0, Decorators_1.NotNull)
], InputMismatchException);
exports.InputMismatchException = InputMismatchException;
//# sourceMappingURL=InputMismatchException.js.map