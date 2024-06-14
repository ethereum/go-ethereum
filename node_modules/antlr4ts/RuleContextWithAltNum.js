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
exports.RuleContextWithAltNum = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:57.4741196-07:00
const ATN_1 = require("./atn/ATN");
const Decorators_1 = require("./Decorators");
const ParserRuleContext_1 = require("./ParserRuleContext");
/** A handy class for use with
 *
 *  options {contextSuperClass=org.antlr.v4.runtime.RuleContextWithAltNum;}
 *
 *  that provides a backing field / impl for the outer alternative number
 *  matched for an internal parse tree node.
 *
 *  I'm only putting into Java runtime as I'm certain I'm the only one that
 *  will really every use this.
 */
class RuleContextWithAltNum extends ParserRuleContext_1.ParserRuleContext {
    constructor(parent, invokingStateNumber) {
        if (invokingStateNumber !== undefined) {
            super(parent, invokingStateNumber);
        }
        else {
            super();
        }
        this._altNumber = ATN_1.ATN.INVALID_ALT_NUMBER;
    }
    get altNumber() {
        return this._altNumber;
    }
    // @Override
    set altNumber(altNum) {
        this._altNumber = altNum;
    }
}
__decorate([
    Decorators_1.Override
], RuleContextWithAltNum.prototype, "altNumber", null);
exports.RuleContextWithAltNum = RuleContextWithAltNum;
//# sourceMappingURL=RuleContextWithAltNum.js.map