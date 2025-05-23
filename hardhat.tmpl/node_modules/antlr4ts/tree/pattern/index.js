"use strict";
/*!
 * Copyright 2016 The ANTLR Project. All rights reserved.
 * Licensed under the BSD-3-Clause license. See LICENSE file in the project root for license information.
 */
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    Object.defineProperty(o, k2, { enumerable: true, get: function() { return m[k]; } });
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __exportStar = (this && this.__exportStar) || function(m, exports) {
    for (var p in m) if (p !== "default" && !Object.prototype.hasOwnProperty.call(exports, p)) __createBinding(exports, m, p);
};
Object.defineProperty(exports, "__esModule", { value: true });
__exportStar(require("./ParseTreeMatch"), exports);
__exportStar(require("./ParseTreePattern"), exports);
__exportStar(require("./ParseTreePatternMatcher"), exports);
__exportStar(require("./RuleTagToken"), exports);
__exportStar(require("./TokenTagToken"), exports);
// The following are "package-private modules" - exported individually but don't need to be part of the public API
// exposed by this file.
//
// export * from "./Chunk";
// export * from "./TagChunk";
// export * from "./TextChunk";
//# sourceMappingURL=index.js.map