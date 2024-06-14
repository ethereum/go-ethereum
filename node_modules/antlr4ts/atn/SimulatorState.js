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
exports.SimulatorState = void 0;
const Decorators_1 = require("../Decorators");
const ParserRuleContext_1 = require("../ParserRuleContext");
/**
 *
 * @author Sam Harwell
 */
let SimulatorState = class SimulatorState {
    constructor(outerContext, s0, useContext, remainingOuterContext) {
        this.outerContext = outerContext != null ? outerContext : ParserRuleContext_1.ParserRuleContext.emptyContext();
        this.s0 = s0;
        this.useContext = useContext;
        this.remainingOuterContext = remainingOuterContext;
    }
};
SimulatorState = __decorate([
    __param(1, Decorators_1.NotNull)
], SimulatorState);
exports.SimulatorState = SimulatorState;
//# sourceMappingURL=SimulatorState.js.map