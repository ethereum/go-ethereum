"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.IntegerStack = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:40.6647101-07:00
const IntegerList_1 = require("./IntegerList");
/**
 *
 * @author Sam Harwell
 */
class IntegerStack extends IntegerList_1.IntegerList {
    constructor(arg) {
        super(arg);
    }
    push(value) {
        this.add(value);
    }
    pop() {
        return this.removeAt(this.size - 1);
    }
    peek() {
        return this.get(this.size - 1);
    }
}
exports.IntegerStack = IntegerStack;
//# sourceMappingURL=IntegerStack.js.map