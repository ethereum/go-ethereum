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
exports.nullable = exports.optional = exports.optionalOrNullable = void 0;
const t = __importStar(require("io-ts"));
function optionalOrNullable(codec, name = `${codec.name} | undefined`) {
    return new t.Type(name, (u) => u === undefined || codec.is(u), (i, c) => {
        if (i === undefined || i === null) {
            return t.success(undefined);
        }
        return codec.validate(i, c);
    }, (a) => {
        if (a === undefined) {
            return undefined;
        }
        return codec.encode(a);
    });
}
exports.optionalOrNullable = optionalOrNullable;
function optional(codec, name = `${codec.name} | undefined`) {
    return new t.Type(name, (u) => u === undefined || codec.is(u), (u, c) => (u === undefined ? t.success(undefined) : codec.validate(u, c)), (a) => (a === undefined ? undefined : codec.encode(a)));
}
exports.optional = optional;
function nullable(codec, name = `${codec.name} | null`) {
    return new t.Type(name, (u) => u === null || codec.is(u), (u, c) => (u === null ? t.success(null) : codec.validate(u, c)), (a) => (a === null ? null : codec.encode(a)));
}
exports.nullable = nullable;
//# sourceMappingURL=io-ts.js.map