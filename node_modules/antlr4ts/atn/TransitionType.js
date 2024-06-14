"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
Object.defineProperty(exports, "__esModule", { value: true });
exports.TransitionType = void 0;
// ConvertTo-TS run at 2016-10-04T11:26:37.8530496-07:00
var TransitionType;
(function (TransitionType) {
    // constants for serialization
    TransitionType[TransitionType["EPSILON"] = 1] = "EPSILON";
    TransitionType[TransitionType["RANGE"] = 2] = "RANGE";
    TransitionType[TransitionType["RULE"] = 3] = "RULE";
    TransitionType[TransitionType["PREDICATE"] = 4] = "PREDICATE";
    TransitionType[TransitionType["ATOM"] = 5] = "ATOM";
    TransitionType[TransitionType["ACTION"] = 6] = "ACTION";
    TransitionType[TransitionType["SET"] = 7] = "SET";
    TransitionType[TransitionType["NOT_SET"] = 8] = "NOT_SET";
    TransitionType[TransitionType["WILDCARD"] = 9] = "WILDCARD";
    TransitionType[TransitionType["PRECEDENCE"] = 10] = "PRECEDENCE";
})(TransitionType = exports.TransitionType || (exports.TransitionType = {}));
//# sourceMappingURL=TransitionType.js.map