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
exports.RuleStopState = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:36.7513856-07:00
const ATNState_1 = require("./ATNState");
const ATNStateType_1 = require("./ATNStateType");
const Decorators_1 = require("../Decorators");
/** The last node in the ATN for a rule, unless that rule is the start symbol.
 *  In that case, there is one transition to EOF. Later, we might encode
 *  references to all calls to this rule to compute FOLLOW sets for
 *  error handling.
 */
class RuleStopState extends ATNState_1.ATNState {
    get nonStopStateNumber() {
        return -1;
    }
    get stateType() {
        return ATNStateType_1.ATNStateType.RULE_STOP;
    }
}
__decorate([
    Decorators_1.Override
], RuleStopState.prototype, "nonStopStateNumber", null);
__decorate([
    Decorators_1.Override
], RuleStopState.prototype, "stateType", null);
exports.RuleStopState = RuleStopState;
//# sourceMappingURL=RuleStopState.js.map