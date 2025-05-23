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
exports.ErrorNode = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:47.4646355-07:00
const Decorators_1 = require("../Decorators");
const TerminalNode_1 = require("./TerminalNode");
/** Represents a token that was consumed during resynchronization
 *  rather than during a valid match operation. For example,
 *  we will create this kind of a node during single token insertion
 *  and deletion as well as during "consume until error recovery set"
 *  upon no viable alternative exceptions.
 */
class ErrorNode extends TerminalNode_1.TerminalNode {
    constructor(token) {
        super(token);
    }
    accept(visitor) {
        return visitor.visitErrorNode(this);
    }
}
__decorate([
    Decorators_1.Override
], ErrorNode.prototype, "accept", null);
exports.ErrorNode = ErrorNode;
//# sourceMappingURL=ErrorNode.js.map