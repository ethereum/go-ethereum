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
const semver = __importStar(require("semver"));
function update(compilerVersion, abi) {
    let hasConstructor = false;
    let hasFallback = false;
    for (let i = 0; i < abi.length; i++) {
        const item = abi[i];
        if (item.type === 'constructor') {
            hasConstructor = true;
            // <0.4.5 assumed every constructor to be payable
            if (semver.lt(compilerVersion, '0.4.5')) {
                item.payable = true;
            }
        }
        else if (item.type === 'fallback') {
            hasFallback = true;
        }
        if (item.type !== 'event') {
            // add 'payable' to everything, except constant functions
            if (!item.constant && semver.lt(compilerVersion, '0.4.0')) {
                item.payable = true;
            }
            // add stateMutability field
            if (semver.lt(compilerVersion, '0.4.16')) {
                if (item.payable) {
                    item.stateMutability = 'payable';
                }
                else if (item.constant) {
                    item.stateMutability = 'view';
                }
                else {
                    item.stateMutability = 'nonpayable';
                }
            }
        }
    }
    // 0.1.2 from Aug 2015 had it. The code has it since May 2015 (e7931ade)
    if (!hasConstructor && semver.lt(compilerVersion, '0.1.2')) {
        abi.push({
            type: 'constructor',
            payable: true,
            stateMutability: 'payable',
            inputs: []
        });
    }
    if (!hasFallback && semver.lt(compilerVersion, '0.4.0')) {
        abi.push({
            type: 'fallback',
            payable: true,
            stateMutability: 'payable'
        });
    }
    return abi;
}
module.exports = {
    update
};
