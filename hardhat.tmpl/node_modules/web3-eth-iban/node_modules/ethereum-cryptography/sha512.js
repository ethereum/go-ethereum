"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.sha512 = void 0;
const sha512_1 = require("@noble/hashes/sha512");
const utils_js_1 = require("./utils.js");
exports.sha512 = (0, utils_js_1.wrapHash)(sha512_1.sha512);
