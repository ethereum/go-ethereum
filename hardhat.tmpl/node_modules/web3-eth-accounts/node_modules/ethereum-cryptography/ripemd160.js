"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ripemd160 = void 0;
const ripemd160_1 = require("@noble/hashes/ripemd160");
const utils_js_1 = require("./utils.js");
exports.ripemd160 = (0, utils_js_1.wrapHash)(ripemd160_1.ripemd160);
