"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.DecisionState = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:28.4381103-07:00
const ATNState_1 = require("./ATNState");
class DecisionState extends ATNState_1.ATNState {
    constructor() {
        super(...arguments);
        this.decision = -1;
        this.nonGreedy = false;
        this.sll = false;
    }
}
exports.DecisionState = DecisionState;
//# sourceMappingURL=DecisionState.js.map