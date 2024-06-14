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
exports.ErrorInfo = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:28.7213647-07:00
const DecisionEventInfo_1 = require("./DecisionEventInfo");
const Decorators_1 = require("../Decorators");
/**
 * This class represents profiling event information for a syntax error
 * identified during prediction. Syntax errors occur when the prediction
 * algorithm is unable to identify an alternative which would lead to a
 * successful parse.
 *
 * @see Parser#notifyErrorListeners(Token, String, RecognitionException)
 * @see ANTLRErrorListener#syntaxError
 *
 * @since 4.3
 */
let ErrorInfo = class ErrorInfo extends DecisionEventInfo_1.DecisionEventInfo {
    /**
     * Constructs a new instance of the {@link ErrorInfo} class with the
     * specified detailed syntax error information.
     *
     * @param decision The decision number
     * @param state The final simulator state reached during prediction
     * prior to reaching the {@link ATNSimulator#ERROR} state
     * @param input The input token stream
     * @param startIndex The start index for the current prediction
     * @param stopIndex The index at which the syntax error was identified
     */
    constructor(decision, state, input, startIndex, stopIndex) {
        super(decision, state, input, startIndex, stopIndex, state.useContext);
    }
};
ErrorInfo = __decorate([
    __param(1, Decorators_1.NotNull),
    __param(2, Decorators_1.NotNull)
], ErrorInfo);
exports.ErrorInfo = ErrorInfo;
//# sourceMappingURL=ErrorInfo.js.map