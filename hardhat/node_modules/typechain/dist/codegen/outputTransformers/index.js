"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.outputTransformers = void 0;
const preamble_1 = require("./preamble");
const prettier_1 = require("./prettier");
exports.outputTransformers = [preamble_1.addPreambleOutputTransformer, prettier_1.prettierOutputTransformer];
//# sourceMappingURL=index.js.map