"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
const wrapper_1 = __importDefault(require("./wrapper"));
const soljson = require('./soljson.js');
module.exports = (0, wrapper_1.default)(soljson);
