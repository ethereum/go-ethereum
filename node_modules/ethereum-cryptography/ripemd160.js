"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
exports.ripemd160 = void 0;
const ripemd160_1 = require("@noble/hashes/ripemd160");
const utils_1 = require("./utils");
exports.ripemd160 = (0, utils_1.wrapHash)(ripemd160_1.ripemd160);
