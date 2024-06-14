"use strict";
var __awaiter = (this && this.__awaiter) || function (thisArg, _arguments, P, generator) {
    function adopt(value) { return value instanceof P ? value : new P(function (resolve) { resolve(value); }); }
    return new (P || (P = Promise))(function (resolve, reject) {
        function fulfilled(value) { try { step(generator.next(value)); } catch (e) { reject(e); } }
        function rejected(value) { try { step(generator["throw"](value)); } catch (e) { reject(e); } }
        function step(result) { result.done ? resolve(result.value) : adopt(result.value).then(fulfilled, rejected); }
        step((generator = generator.apply(thisArg, _arguments || [])).next());
    });
};
var __generator = (this && this.__generator) || function (thisArg, body) {
    var _ = { label: 0, sent: function() { if (t[0] & 1) throw t[1]; return t[1]; }, trys: [], ops: [] }, f, y, t, g;
    return g = { next: verb(0), "throw": verb(1), "return": verb(2) }, typeof Symbol === "function" && (g[Symbol.iterator] = function() { return this; }), g;
    function verb(n) { return function (v) { return step([n, v]); }; }
    function step(op) {
        if (f) throw new TypeError("Generator is already executing.");
        while (_) try {
            if (f = 1, y && (t = op[0] & 2 ? y["return"] : op[0] ? y["throw"] || ((t = y["return"]) && t.call(y), 0) : y.next) && !(t = t.call(y, op[1])).done) return t;
            if (y = 0, t) op = [op[0] & 2, t.value];
            switch (op[0]) {
                case 0: case 1: t = op; break;
                case 4: _.label++; return { value: op[1], done: false };
                case 5: _.label++; y = op[1]; op = [0]; continue;
                case 7: op = _.ops.pop(); _.trys.pop(); continue;
                default:
                    if (!(t = _.trys, t = t.length > 0 && t[t.length - 1]) && (op[0] === 6 || op[0] === 2)) { _ = 0; continue; }
                    if (op[0] === 3 && (!t || (op[1] > t[0] && op[1] < t[3]))) { _.label = op[1]; break; }
                    if (op[0] === 6 && _.label < t[1]) { _.label = t[1]; t = op; break; }
                    if (t && _.label < t[2]) { _.label = t[2]; _.ops.push(op); break; }
                    if (t[2]) _.ops.pop();
                    _.trys.pop(); continue;
            }
            op = body.call(thisArg, _);
        } catch (e) { op = [6, e]; y = 0; } finally { f = t = 0; }
        if (op[0] & 5) throw op[1]; return { value: op[0] ? op[1] : void 0, done: true };
    }
};
Object.defineProperty(exports, "__esModule", { value: true });
exports.TypedDataEncoder = void 0;
var address_1 = require("@ethersproject/address");
var bignumber_1 = require("@ethersproject/bignumber");
var bytes_1 = require("@ethersproject/bytes");
var keccak256_1 = require("@ethersproject/keccak256");
var properties_1 = require("@ethersproject/properties");
var logger_1 = require("@ethersproject/logger");
var _version_1 = require("./_version");
var logger = new logger_1.Logger(_version_1.version);
var id_1 = require("./id");
var padding = new Uint8Array(32);
padding.fill(0);
var NegativeOne = bignumber_1.BigNumber.from(-1);
var Zero = bignumber_1.BigNumber.from(0);
var One = bignumber_1.BigNumber.from(1);
var MaxUint256 = bignumber_1.BigNumber.from("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff");
function hexPadRight(value) {
    var bytes = (0, bytes_1.arrayify)(value);
    var padOffset = bytes.length % 32;
    if (padOffset) {
        return (0, bytes_1.hexConcat)([bytes, padding.slice(padOffset)]);
    }
    return (0, bytes_1.hexlify)(bytes);
}
var hexTrue = (0, bytes_1.hexZeroPad)(One.toHexString(), 32);
var hexFalse = (0, bytes_1.hexZeroPad)(Zero.toHexString(), 32);
var domainFieldTypes = {
    name: "string",
    version: "string",
    chainId: "uint256",
    verifyingContract: "address",
    salt: "bytes32"
};
var domainFieldNames = [
    "name", "version", "chainId", "verifyingContract", "salt"
];
function checkString(key) {
    return function (value) {
        if (typeof (value) !== "string") {
            logger.throwArgumentError("invalid domain value for " + JSON.stringify(key), "domain." + key, value);
        }
        return value;
    };
}
var domainChecks = {
    name: checkString("name"),
    version: checkString("version"),
    chainId: function (value) {
        try {
            return bignumber_1.BigNumber.from(value).toString();
        }
        catch (error) { }
        return logger.throwArgumentError("invalid domain value for \"chainId\"", "domain.chainId", value);
    },
    verifyingContract: function (value) {
        try {
            return (0, address_1.getAddress)(value).toLowerCase();
        }
        catch (error) { }
        return logger.throwArgumentError("invalid domain value \"verifyingContract\"", "domain.verifyingContract", value);
    },
    salt: function (value) {
        try {
            var bytes = (0, bytes_1.arrayify)(value);
            if (bytes.length !== 32) {
                throw new Error("bad length");
            }
            return (0, bytes_1.hexlify)(bytes);
        }
        catch (error) { }
        return logger.throwArgumentError("invalid domain value \"salt\"", "domain.salt", value);
    }
};
function getBaseEncoder(type) {
    // intXX and uintXX
    {
        var match = type.match(/^(u?)int(\d*)$/);
        if (match) {
            var signed = (match[1] === "");
            var width = parseInt(match[2] || "256");
            if (width % 8 !== 0 || width > 256 || (match[2] && match[2] !== String(width))) {
                logger.throwArgumentError("invalid numeric width", "type", type);
            }
            var boundsUpper_1 = MaxUint256.mask(signed ? (width - 1) : width);
            var boundsLower_1 = signed ? boundsUpper_1.add(One).mul(NegativeOne) : Zero;
            return function (value) {
                var v = bignumber_1.BigNumber.from(value);
                if (v.lt(boundsLower_1) || v.gt(boundsUpper_1)) {
                    logger.throwArgumentError("value out-of-bounds for " + type, "value", value);
                }
                return (0, bytes_1.hexZeroPad)(v.toTwos(256).toHexString(), 32);
            };
        }
    }
    // bytesXX
    {
        var match = type.match(/^bytes(\d+)$/);
        if (match) {
            var width_1 = parseInt(match[1]);
            if (width_1 === 0 || width_1 > 32 || match[1] !== String(width_1)) {
                logger.throwArgumentError("invalid bytes width", "type", type);
            }
            return function (value) {
                var bytes = (0, bytes_1.arrayify)(value);
                if (bytes.length !== width_1) {
                    logger.throwArgumentError("invalid length for " + type, "value", value);
                }
                return hexPadRight(value);
            };
        }
    }
    switch (type) {
        case "address": return function (value) {
            return (0, bytes_1.hexZeroPad)((0, address_1.getAddress)(value), 32);
        };
        case "bool": return function (value) {
            return ((!value) ? hexFalse : hexTrue);
        };
        case "bytes": return function (value) {
            return (0, keccak256_1.keccak256)(value);
        };
        case "string": return function (value) {
            return (0, id_1.id)(value);
        };
    }
    return null;
}
function encodeType(name, fields) {
    return name + "(" + fields.map(function (_a) {
        var name = _a.name, type = _a.type;
        return (type + " " + name);
    }).join(",") + ")";
}
var TypedDataEncoder = /** @class */ (function () {
    function TypedDataEncoder(types) {
        (0, properties_1.defineReadOnly)(this, "types", Object.freeze((0, properties_1.deepCopy)(types)));
        (0, properties_1.defineReadOnly)(this, "_encoderCache", {});
        (0, properties_1.defineReadOnly)(this, "_types", {});
        // Link struct types to their direct child structs
        var links = {};
        // Link structs to structs which contain them as a child
        var parents = {};
        // Link all subtypes within a given struct
        var subtypes = {};
        Object.keys(types).forEach(function (type) {
            links[type] = {};
            parents[type] = [];
            subtypes[type] = {};
        });
        var _loop_1 = function (name_1) {
            var uniqueNames = {};
            types[name_1].forEach(function (field) {
                // Check each field has a unique name
                if (uniqueNames[field.name]) {
                    logger.throwArgumentError("duplicate variable name " + JSON.stringify(field.name) + " in " + JSON.stringify(name_1), "types", types);
                }
                uniqueNames[field.name] = true;
                // Get the base type (drop any array specifiers)
                var baseType = field.type.match(/^([^\x5b]*)(\x5b|$)/)[1];
                if (baseType === name_1) {
                    logger.throwArgumentError("circular type reference to " + JSON.stringify(baseType), "types", types);
                }
                // Is this a base encoding type?
                var encoder = getBaseEncoder(baseType);
                if (encoder) {
                    return;
                }
                if (!parents[baseType]) {
                    logger.throwArgumentError("unknown type " + JSON.stringify(baseType), "types", types);
                }
                // Add linkage
                parents[baseType].push(name_1);
                links[name_1][baseType] = true;
            });
        };
        for (var name_1 in types) {
            _loop_1(name_1);
        }
        // Deduce the primary type
        var primaryTypes = Object.keys(parents).filter(function (n) { return (parents[n].length === 0); });
        if (primaryTypes.length === 0) {
            logger.throwArgumentError("missing primary type", "types", types);
        }
        else if (primaryTypes.length > 1) {
            logger.throwArgumentError("ambiguous primary types or unused types: " + primaryTypes.map(function (t) { return (JSON.stringify(t)); }).join(", "), "types", types);
        }
        (0, properties_1.defineReadOnly)(this, "primaryType", primaryTypes[0]);
        // Check for circular type references
        function checkCircular(type, found) {
            if (found[type]) {
                logger.throwArgumentError("circular type reference to " + JSON.stringify(type), "types", types);
            }
            found[type] = true;
            Object.keys(links[type]).forEach(function (child) {
                if (!parents[child]) {
                    return;
                }
                // Recursively check children
                checkCircular(child, found);
                // Mark all ancestors as having this decendant
                Object.keys(found).forEach(function (subtype) {
                    subtypes[subtype][child] = true;
                });
            });
            delete found[type];
        }
        checkCircular(this.primaryType, {});
        // Compute each fully describe type
        for (var name_2 in subtypes) {
            var st = Object.keys(subtypes[name_2]);
            st.sort();
            this._types[name_2] = encodeType(name_2, types[name_2]) + st.map(function (t) { return encodeType(t, types[t]); }).join("");
        }
    }
    TypedDataEncoder.prototype.getEncoder = function (type) {
        var encoder = this._encoderCache[type];
        if (!encoder) {
            encoder = this._encoderCache[type] = this._getEncoder(type);
        }
        return encoder;
    };
    TypedDataEncoder.prototype._getEncoder = function (type) {
        var _this = this;
        // Basic encoder type (address, bool, uint256, etc)
        {
            var encoder = getBaseEncoder(type);
            if (encoder) {
                return encoder;
            }
        }
        // Array
        var match = type.match(/^(.*)(\x5b(\d*)\x5d)$/);
        if (match) {
            var subtype_1 = match[1];
            var subEncoder_1 = this.getEncoder(subtype_1);
            var length_1 = parseInt(match[3]);
            return function (value) {
                if (length_1 >= 0 && value.length !== length_1) {
                    logger.throwArgumentError("array length mismatch; expected length ${ arrayLength }", "value", value);
                }
                var result = value.map(subEncoder_1);
                if (_this._types[subtype_1]) {
                    result = result.map(keccak256_1.keccak256);
                }
                return (0, keccak256_1.keccak256)((0, bytes_1.hexConcat)(result));
            };
        }
        // Struct
        var fields = this.types[type];
        if (fields) {
            var encodedType_1 = (0, id_1.id)(this._types[type]);
            return function (value) {
                var values = fields.map(function (_a) {
                    var name = _a.name, type = _a.type;
                    var result = _this.getEncoder(type)(value[name]);
                    if (_this._types[type]) {
                        return (0, keccak256_1.keccak256)(result);
                    }
                    return result;
                });
                values.unshift(encodedType_1);
                return (0, bytes_1.hexConcat)(values);
            };
        }
        return logger.throwArgumentError("unknown type: " + type, "type", type);
    };
    TypedDataEncoder.prototype.encodeType = function (name) {
        var result = this._types[name];
        if (!result) {
            logger.throwArgumentError("unknown type: " + JSON.stringify(name), "name", name);
        }
        return result;
    };
    TypedDataEncoder.prototype.encodeData = function (type, value) {
        return this.getEncoder(type)(value);
    };
    TypedDataEncoder.prototype.hashStruct = function (name, value) {
        return (0, keccak256_1.keccak256)(this.encodeData(name, value));
    };
    TypedDataEncoder.prototype.encode = function (value) {
        return this.encodeData(this.primaryType, value);
    };
    TypedDataEncoder.prototype.hash = function (value) {
        return this.hashStruct(this.primaryType, value);
    };
    TypedDataEncoder.prototype._visit = function (type, value, callback) {
        var _this = this;
        // Basic encoder type (address, bool, uint256, etc)
        {
            var encoder = getBaseEncoder(type);
            if (encoder) {
                return callback(type, value);
            }
        }
        // Array
        var match = type.match(/^(.*)(\x5b(\d*)\x5d)$/);
        if (match) {
            var subtype_2 = match[1];
            var length_2 = parseInt(match[3]);
            if (length_2 >= 0 && value.length !== length_2) {
                logger.throwArgumentError("array length mismatch; expected length ${ arrayLength }", "value", value);
            }
            return value.map(function (v) { return _this._visit(subtype_2, v, callback); });
        }
        // Struct
        var fields = this.types[type];
        if (fields) {
            return fields.reduce(function (accum, _a) {
                var name = _a.name, type = _a.type;
                accum[name] = _this._visit(type, value[name], callback);
                return accum;
            }, {});
        }
        return logger.throwArgumentError("unknown type: " + type, "type", type);
    };
    TypedDataEncoder.prototype.visit = function (value, callback) {
        return this._visit(this.primaryType, value, callback);
    };
    TypedDataEncoder.from = function (types) {
        return new TypedDataEncoder(types);
    };
    TypedDataEncoder.getPrimaryType = function (types) {
        return TypedDataEncoder.from(types).primaryType;
    };
    TypedDataEncoder.hashStruct = function (name, types, value) {
        return TypedDataEncoder.from(types).hashStruct(name, value);
    };
    TypedDataEncoder.hashDomain = function (domain) {
        var domainFields = [];
        for (var name_3 in domain) {
            var type = domainFieldTypes[name_3];
            if (!type) {
                logger.throwArgumentError("invalid typed-data domain key: " + JSON.stringify(name_3), "domain", domain);
            }
            domainFields.push({ name: name_3, type: type });
        }
        domainFields.sort(function (a, b) {
            return domainFieldNames.indexOf(a.name) - domainFieldNames.indexOf(b.name);
        });
        return TypedDataEncoder.hashStruct("EIP712Domain", { EIP712Domain: domainFields }, domain);
    };
    TypedDataEncoder.encode = function (domain, types, value) {
        return (0, bytes_1.hexConcat)([
            "0x1901",
            TypedDataEncoder.hashDomain(domain),
            TypedDataEncoder.from(types).hash(value)
        ]);
    };
    TypedDataEncoder.hash = function (domain, types, value) {
        return (0, keccak256_1.keccak256)(TypedDataEncoder.encode(domain, types, value));
    };
    // Replaces all address types with ENS names with their looked up address
    TypedDataEncoder.resolveNames = function (domain, types, value, resolveName) {
        return __awaiter(this, void 0, void 0, function () {
            var ensCache, encoder, _a, _b, _i, name_4, _c, _d;
            return __generator(this, function (_e) {
                switch (_e.label) {
                    case 0:
                        // Make a copy to isolate it from the object passed in
                        domain = (0, properties_1.shallowCopy)(domain);
                        ensCache = {};
                        // Do we need to look up the domain's verifyingContract?
                        if (domain.verifyingContract && !(0, bytes_1.isHexString)(domain.verifyingContract, 20)) {
                            ensCache[domain.verifyingContract] = "0x";
                        }
                        encoder = TypedDataEncoder.from(types);
                        // Get a list of all the addresses
                        encoder.visit(value, function (type, value) {
                            if (type === "address" && !(0, bytes_1.isHexString)(value, 20)) {
                                ensCache[value] = "0x";
                            }
                            return value;
                        });
                        _a = [];
                        for (_b in ensCache)
                            _a.push(_b);
                        _i = 0;
                        _e.label = 1;
                    case 1:
                        if (!(_i < _a.length)) return [3 /*break*/, 4];
                        name_4 = _a[_i];
                        _c = ensCache;
                        _d = name_4;
                        return [4 /*yield*/, resolveName(name_4)];
                    case 2:
                        _c[_d] = _e.sent();
                        _e.label = 3;
                    case 3:
                        _i++;
                        return [3 /*break*/, 1];
                    case 4:
                        // Replace the domain verifyingContract if needed
                        if (domain.verifyingContract && ensCache[domain.verifyingContract]) {
                            domain.verifyingContract = ensCache[domain.verifyingContract];
                        }
                        // Replace all ENS names with their address
                        value = encoder.visit(value, function (type, value) {
                            if (type === "address" && ensCache[value]) {
                                return ensCache[value];
                            }
                            return value;
                        });
                        return [2 /*return*/, { domain: domain, value: value }];
                }
            });
        });
    };
    TypedDataEncoder.getPayload = function (domain, types, value) {
        // Validate the domain fields
        TypedDataEncoder.hashDomain(domain);
        // Derive the EIP712Domain Struct reference type
        var domainValues = {};
        var domainTypes = [];
        domainFieldNames.forEach(function (name) {
            var value = domain[name];
            if (value == null) {
                return;
            }
            domainValues[name] = domainChecks[name](value);
            domainTypes.push({ name: name, type: domainFieldTypes[name] });
        });
        var encoder = TypedDataEncoder.from(types);
        var typesWithDomain = (0, properties_1.shallowCopy)(types);
        if (typesWithDomain.EIP712Domain) {
            logger.throwArgumentError("types must not contain EIP712Domain type", "types.EIP712Domain", types);
        }
        else {
            typesWithDomain.EIP712Domain = domainTypes;
        }
        // Validate the data structures and types
        encoder.encode(value);
        return {
            types: typesWithDomain,
            domain: domainValues,
            primaryType: encoder.primaryType,
            message: encoder.visit(value, function (type, value) {
                // bytes
                if (type.match(/^bytes(\d*)/)) {
                    return (0, bytes_1.hexlify)((0, bytes_1.arrayify)(value));
                }
                // uint or int
                if (type.match(/^u?int/)) {
                    return bignumber_1.BigNumber.from(value).toString();
                }
                switch (type) {
                    case "address":
                        return value.toLowerCase();
                    case "bool":
                        return !!value;
                    case "string":
                        if (typeof (value) !== "string") {
                            logger.throwArgumentError("invalid string", "value", value);
                        }
                        return value;
                }
                return logger.throwArgumentError("unsupported type", "type", type);
            })
        };
    };
    return TypedDataEncoder;
}());
exports.TypedDataEncoder = TypedDataEncoder;
//# sourceMappingURL=typed-data.js.map