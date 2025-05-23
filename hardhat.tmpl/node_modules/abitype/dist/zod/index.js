"use strict";Object.defineProperty(exports, "__esModule", {value: true});


var _chunkO6V2CMEFjs = require('../chunk-O6V2CMEF.js');
require('../chunk-XXPGZHWZ.js');

// src/zod/zod.ts
var _zod = require('zod');
var SolidityAddress = _zod.z.literal("address");
var SolidityBool = _zod.z.literal("bool");
var SolidityBytes = _zod.z.string().regex(_chunkO6V2CMEFjs.bytesRegex);
var SolidityFunction = _zod.z.literal("function");
var SolidityString = _zod.z.literal("string");
var SolidityTuple = _zod.z.literal("tuple");
var SolidityInt = _zod.z.string().regex(_chunkO6V2CMEFjs.integerRegex);
var SolidityArrayWithoutTuple = _zod.z.string().regex(
  /^(address|bool|function|string|bytes([1-9]|1[0-9]|2[0-9]|3[0-2])?|u?int(8|16|24|32|40|48|56|64|72|80|88|96|104|112|120|128|136|144|152|160|168|176|184|192|200|208|216|224|232|240|248|256)?)(\[[0-9]{0,}\])+$/
);
var SolidityArrayWithTuple = _zod.z.string().regex(/^tuple(\[[0-9]{0,}\])+$/);
var SolidityArray = _zod.z.union([
  SolidityArrayWithTuple,
  SolidityArrayWithoutTuple
]);
var AbiParameter = _zod.z.lazy(
  () => _zod.z.intersection(
    _zod.z.object({
      name: _zod.z.string().optional(),
      /** Representation used by Solidity compiler */
      internalType: _zod.z.string().optional()
    }),
    _zod.z.union([
      _zod.z.object({
        type: _zod.z.union([
          SolidityAddress,
          SolidityBool,
          SolidityBytes,
          SolidityFunction,
          SolidityString,
          SolidityInt,
          SolidityArrayWithoutTuple
        ])
      }),
      _zod.z.object({
        type: _zod.z.union([SolidityTuple, SolidityArrayWithTuple]),
        components: _zod.z.array(AbiParameter)
      })
    ])
  )
);
var AbiStateMutability = _zod.z.union([
  _zod.z.literal("pure"),
  _zod.z.literal("view"),
  _zod.z.literal("nonpayable"),
  _zod.z.literal("payable")
]);
var AbiFunction = _zod.z.preprocess(
  (val) => {
    const abiFunction = val;
    if (abiFunction.stateMutability === void 0) {
      if (abiFunction.constant)
        abiFunction.stateMutability = "view";
      else if (abiFunction.payable)
        abiFunction.stateMutability = "payable";
      else
        abiFunction.stateMutability = "nonpayable";
    }
    return val;
  },
  _zod.z.object({
    type: _zod.z.literal("function"),
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    constant: _zod.z.boolean().optional(),
    /**
     * @deprecated Vyper used to provide gas estimates
     * https://github.com/vyperlang/vyper/issues/2151
     */
    gas: _zod.z.number().optional(),
    inputs: _zod.z.array(AbiParameter),
    name: _zod.z.string(),
    outputs: _zod.z.array(AbiParameter),
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: _zod.z.boolean().optional(),
    stateMutability: AbiStateMutability
  })
);
var AbiConstructor = _zod.z.preprocess(
  (val) => {
    const abiFunction = val;
    if (abiFunction.stateMutability === void 0) {
      if (abiFunction.payable)
        abiFunction.stateMutability = "payable";
      else
        abiFunction.stateMutability = "nonpayable";
    }
    return val;
  },
  _zod.z.object({
    type: _zod.z.literal("constructor"),
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    inputs: _zod.z.array(AbiParameter),
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: _zod.z.boolean().optional(),
    stateMutability: _zod.z.union([_zod.z.literal("nonpayable"), _zod.z.literal("payable")])
  })
);
var AbiFallback = _zod.z.preprocess(
  (val) => {
    const abiFunction = val;
    if (abiFunction.stateMutability === void 0) {
      if (abiFunction.payable)
        abiFunction.stateMutability = "payable";
      else
        abiFunction.stateMutability = "nonpayable";
    }
    return val;
  },
  _zod.z.object({
    type: _zod.z.literal("fallback"),
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    inputs: _zod.z.tuple([]).optional(),
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: _zod.z.boolean().optional(),
    stateMutability: _zod.z.union([_zod.z.literal("nonpayable"), _zod.z.literal("payable")])
  })
);
var AbiReceive = _zod.z.object({
  type: _zod.z.literal("receive"),
  stateMutability: _zod.z.literal("payable")
});
var AbiEvent = _zod.z.object({
  type: _zod.z.literal("event"),
  anonymous: _zod.z.boolean().optional(),
  inputs: _zod.z.array(
    _zod.z.intersection(AbiParameter, _zod.z.object({ indexed: _zod.z.boolean().optional() }))
  ),
  name: _zod.z.string()
});
var AbiError = _zod.z.object({
  type: _zod.z.literal("error"),
  inputs: _zod.z.array(AbiParameter),
  name: _zod.z.string()
});
var AbiItemType = _zod.z.union([
  _zod.z.literal("constructor"),
  _zod.z.literal("event"),
  _zod.z.literal("error"),
  _zod.z.literal("fallback"),
  _zod.z.literal("function"),
  _zod.z.literal("receive")
]);
var Abi = _zod.z.array(
  _zod.z.union([
    AbiError,
    AbiEvent,
    // TODO: Replace code below to `z.switch` (https://github.com/colinhacks/zod/issues/2106)
    // Need to redefine `AbiFunction | AbiConstructor | AbiFallback | AbiReceive` since `z.discriminate` doesn't support `z.preprocess` on `options`
    // https://github.com/colinhacks/zod/issues/1490
    _zod.z.preprocess(
      (val) => {
        const abiItem = val;
        if (abiItem.type === "receive")
          return abiItem;
        if (val.stateMutability === void 0) {
          if (abiItem.type === "function" && abiItem.constant)
            abiItem.stateMutability = "view";
          else if (abiItem.payable)
            abiItem.stateMutability = "payable";
          else
            abiItem.stateMutability = "nonpayable";
        }
        return val;
      },
      _zod.z.intersection(
        _zod.z.object({
          /**
           * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
           * https://github.com/ethereum/solidity/issues/992
           */
          constant: _zod.z.boolean().optional(),
          /**
           * @deprecated Vyper used to provide gas estimates
           * https://github.com/vyperlang/vyper/issues/2151
           */
          gas: _zod.z.number().optional(),
          /**
           * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
           * https://github.com/ethereum/solidity/issues/992
           */
          payable: _zod.z.boolean().optional(),
          stateMutability: AbiStateMutability
        }),
        _zod.z.discriminatedUnion("type", [
          _zod.z.object({
            type: _zod.z.literal("function"),
            inputs: _zod.z.array(AbiParameter),
            name: _zod.z.string(),
            outputs: _zod.z.array(AbiParameter)
          }),
          _zod.z.object({
            type: _zod.z.literal("constructor"),
            inputs: _zod.z.array(AbiParameter)
          }),
          _zod.z.object({
            type: _zod.z.literal("fallback"),
            inputs: _zod.z.tuple([]).optional()
          }),
          _zod.z.object({
            type: _zod.z.literal("receive"),
            stateMutability: _zod.z.literal("payable")
          })
        ])
      )
    )
  ])
);





















exports.Abi = Abi; exports.AbiConstructor = AbiConstructor; exports.AbiError = AbiError; exports.AbiEvent = AbiEvent; exports.AbiFallback = AbiFallback; exports.AbiFunction = AbiFunction; exports.AbiItemType = AbiItemType; exports.AbiParameter = AbiParameter; exports.AbiReceive = AbiReceive; exports.AbiStateMutability = AbiStateMutability; exports.SolidityAddress = SolidityAddress; exports.SolidityArray = SolidityArray; exports.SolidityArrayWithTuple = SolidityArrayWithTuple; exports.SolidityArrayWithoutTuple = SolidityArrayWithoutTuple; exports.SolidityBool = SolidityBool; exports.SolidityBytes = SolidityBytes; exports.SolidityFunction = SolidityFunction; exports.SolidityInt = SolidityInt; exports.SolidityString = SolidityString; exports.SolidityTuple = SolidityTuple;
