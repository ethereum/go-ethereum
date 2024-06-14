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
exports.XPathTokenElement = void 0;
// CONVERSTION complete, Burt Harris 10/14/2016
const Decorators_1 = require("../../Decorators");
const TerminalNode_1 = require("../TerminalNode");
const Trees_1 = require("../Trees");
const XPathElement_1 = require("./XPathElement");
class XPathTokenElement extends XPathElement_1.XPathElement {
    constructor(tokenName, tokenType) {
        super(tokenName);
        this.tokenType = tokenType;
    }
    evaluate(t) {
        // return all children of t that match nodeName
        let nodes = [];
        for (let c of Trees_1.Trees.getChildren(t)) {
            if (c instanceof TerminalNode_1.TerminalNode) {
                if ((c.symbol.type === this.tokenType && !this.invert) ||
                    (c.symbol.type !== this.tokenType && this.invert)) {
                    nodes.push(c);
                }
            }
        }
        return nodes;
    }
}
__decorate([
    Decorators_1.Override
], XPathTokenElement.prototype, "evaluate", null);
exports.XPathTokenElement = XPathTokenElement;
//# sourceMappingURL=XPathTokenElement.js.map