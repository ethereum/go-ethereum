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
exports.BlockEndState = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:27.9125304-07:00
const ATNState_1 = require("./ATNState");
const ATNStateType_1 = require("./ATNStateType");
const Decorators_1 = require("../Decorators");
/** Terminal node of a simple `(a|b|c)` block. */
class BlockEndState extends ATNState_1.ATNState {
    get stateType() {
        return ATNStateType_1.ATNStateType.BLOCK_END;
    }
}
__decorate([
    Decorators_1.Override
], BlockEndState.prototype, "stateType", null);
exports.BlockEndState = BlockEndState;
//# sourceMappingURL=BlockEndState.js.map