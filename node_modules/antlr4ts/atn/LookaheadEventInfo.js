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
exports.LookaheadEventInfo = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:30.6852565-07:00
const DecisionEventInfo_1 = require("./DecisionEventInfo");
const Decorators_1 = require("../Decorators");
/**
 * This class represents profiling event information for tracking the lookahead
 * depth required in order to make a prediction.
 *
 * @since 4.3
 */
let LookaheadEventInfo = class LookaheadEventInfo extends DecisionEventInfo_1.DecisionEventInfo {
    /**
     * Constructs a new instance of the {@link LookaheadEventInfo} class with
     * the specified detailed lookahead information.
     *
     * @param decision The decision number
     * @param state The final simulator state containing the necessary
     * information to determine the result of a prediction, or `undefined` if
     * the final state is not available
     * @param input The input token stream
     * @param startIndex The start index for the current prediction
     * @param stopIndex The index at which the prediction was finally made
     * @param fullCtx `true` if the current lookahead is part of an LL
     * prediction; otherwise, `false` if the current lookahead is part of
     * an SLL prediction
     */
    constructor(decision, state, predictedAlt, input, startIndex, stopIndex, fullCtx) {
        super(decision, state, input, startIndex, stopIndex, fullCtx);
        this.predictedAlt = predictedAlt;
    }
};
LookaheadEventInfo = __decorate([
    __param(3, Decorators_1.NotNull)
], LookaheadEventInfo);
exports.LookaheadEventInfo = LookaheadEventInfo;
//# sourceMappingURL=LookaheadEventInfo.js.map