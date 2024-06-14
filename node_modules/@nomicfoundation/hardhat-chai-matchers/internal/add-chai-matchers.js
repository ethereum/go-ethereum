"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const chai_1 = require("chai");
const chai_as_promised_1 = __importDefault(require("chai-as-promised"));
require("../types");
const hardhatChaiMatchers_1 = require("./hardhatChaiMatchers");
(0, chai_1.use)(hardhatChaiMatchers_1.hardhatChaiMatchers);
(0, chai_1.use)(chai_as_promised_1.default);
//# sourceMappingURL=add-chai-matchers.js.map