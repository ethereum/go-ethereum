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
exports.InterpreterRuleContext = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:51.5898546-07:00
const Decorators_1 = require("./Decorators");
const ParserRuleContext_1 = require("./ParserRuleContext");
/**
 * This class extends {@link ParserRuleContext} by allowing the value of
 * {@link #getRuleIndex} to be explicitly set for the context.
 *
 * {@link ParserRuleContext} does not include field storage for the rule index
 * since the context classes created by the code generator override the
 * {@link #getRuleIndex} method to return the correct value for that context.
 * Since the parser interpreter does not use the context classes generated for a
 * parser, this class (with slightly more memory overhead per node) is used to
 * provide equivalent functionality.
 */
class InterpreterRuleContext extends ParserRuleContext_1.ParserRuleContext {
    constructor(ruleIndex, parent, invokingStateNumber) {
        if (invokingStateNumber !== undefined) {
            super(parent, invokingStateNumber);
        }
        else {
            super();
        }
        this._ruleIndex = ruleIndex;
    }
    get ruleIndex() {
        return this._ruleIndex;
    }
}
__decorate([
    Decorators_1.Override
], InterpreterRuleContext.prototype, "ruleIndex", null);
exports.InterpreterRuleContext = InterpreterRuleContext;
//# sourceMappingURL=InterpreterRuleContext.js.map