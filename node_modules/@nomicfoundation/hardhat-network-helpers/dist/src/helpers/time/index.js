"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.setNextBlockTimestamp = exports.latestBlock = exports.latest = exports.increaseTo = exports.increase = exports.advanceBlockTo = exports.advanceBlock = exports.duration = void 0;
exports.duration = __importStar(require("./duration"));
var advanceBlock_1 = require("./advanceBlock");
Object.defineProperty(exports, "advanceBlock", { enumerable: true, get: function () { return advanceBlock_1.advanceBlock; } });
var advanceBlockTo_1 = require("./advanceBlockTo");
Object.defineProperty(exports, "advanceBlockTo", { enumerable: true, get: function () { return advanceBlockTo_1.advanceBlockTo; } });
var increase_1 = require("./increase");
Object.defineProperty(exports, "increase", { enumerable: true, get: function () { return increase_1.increase; } });
var increaseTo_1 = require("./increaseTo");
Object.defineProperty(exports, "increaseTo", { enumerable: true, get: function () { return increaseTo_1.increaseTo; } });
var latest_1 = require("./latest");
Object.defineProperty(exports, "latest", { enumerable: true, get: function () { return latest_1.latest; } });
var latestBlock_1 = require("./latestBlock");
Object.defineProperty(exports, "latestBlock", { enumerable: true, get: function () { return latestBlock_1.latestBlock; } });
var setNextBlockTimestamp_1 = require("./setNextBlockTimestamp");
Object.defineProperty(exports, "setNextBlockTimestamp", { enumerable: true, get: function () { return setNextBlockTimestamp_1.setNextBlockTimestamp; } });
//# sourceMappingURL=index.js.map