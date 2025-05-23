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
exports.PlusLoopbackState = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:35.0257730-07:00
const ATNStateType_1 = require("./ATNStateType");
const DecisionState_1 = require("./DecisionState");
const Decorators_1 = require("../Decorators");
/** Decision state for `A+` and `(A|B)+`.  It has two transitions:
 *  one to the loop back to start of the block and one to exit.
 */
class PlusLoopbackState extends DecisionState_1.DecisionState {
    get stateType() {
        return ATNStateType_1.ATNStateType.PLUS_LOOP_BACK;
    }
}
__decorate([
    Decorators_1.Override
], PlusLoopbackState.prototype, "stateType", null);
exports.PlusLoopbackState = PlusLoopbackState;
//# sourceMappingURL=PlusLoopbackState.js.map