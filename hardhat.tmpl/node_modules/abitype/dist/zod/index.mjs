import {
  bytesRegex,
  integerRegex
} from "../chunk-WP7KDV47.mjs";
import "../chunk-NHABU752.mjs";

// src/zod/zod.ts
import { z } from "zod";
var SolidityAddress = z.literal("address");
var SolidityBool = z.literal("bool");
var SolidityBytes = z.string().regex(bytesRegex);
var SolidityFunction = z.literal("function");
var SolidityString = z.literal("string");
var SolidityTuple = z.literal("tuple");
var SolidityInt = z.string().regex(integerRegex);
var SolidityArrayWithoutTuple = z.string().regex(
  /^(address|bool|function|string|bytes([1-9]|1[0-9]|2[0-9]|3[0-2])?|u?int(8|16|24|32|40|48|56|64|72|80|88|96|104|112|120|128|136|144|152|160|168|176|184|192|200|208|216|224|232|240|248|256)?)(\[[0-9]{0,}\])+$/
);
var SolidityArrayWithTuple = z.string().regex(/^tuple(\[[0-9]{0,}\])+$/);
var SolidityArray = z.union([
  SolidityArrayWithTuple,
  SolidityArrayWithoutTuple
]);
var AbiParameter = z.lazy(
  () => z.intersection(
    z.object({
      name: z.string().optional(),
      /** Representation used by Solidity compiler */
      internalType: z.string().optional()
    }),
    z.union([
      z.object({
        type: z.union([
          SolidityAddress,
          SolidityBool,
          SolidityBytes,
          SolidityFunction,
          SolidityString,
          SolidityInt,
          SolidityArrayWithoutTuple
        ])
      }),
      z.object({
        type: z.union([SolidityTuple, SolidityArrayWithTuple]),
        components: z.array(AbiParameter)
      })
    ])
  )
);
var AbiStateMutability = z.union([
  z.literal("pure"),
  z.literal("view"),
  z.literal("nonpayable"),
  z.literal("payable")
]);
var AbiFunction = z.preprocess(
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
  z.object({
    type: z.literal("function"),
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    constant: z.boolean().optional(),
    /**
     * @deprecated Vyper used to provide gas estimates
     * https://github.com/vyperlang/vyper/issues/2151
     */
    gas: z.number().optional(),
    inputs: z.array(AbiParameter),
    name: z.string(),
    outputs: z.array(AbiParameter),
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: z.boolean().optional(),
    stateMutability: AbiStateMutability
  })
);
var AbiConstructor = z.preprocess(
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
  z.object({
    type: z.literal("constructor"),
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    inputs: z.array(AbiParameter),
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: z.boolean().optional(),
    stateMutability: z.union([z.literal("nonpayable"), z.literal("payable")])
  })
);
var AbiFallback = z.preprocess(
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
  z.object({
    type: z.literal("fallback"),
    /**
     * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    inputs: z.tuple([]).optional(),
    /**
     * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
     * https://github.com/ethereum/solidity/issues/992
     */
    payable: z.boolean().optional(),
    stateMutability: z.union([z.literal("nonpayable"), z.literal("payable")])
  })
);
var AbiReceive = z.object({
  type: z.literal("receive"),
  stateMutability: z.literal("payable")
});
var AbiEvent = z.object({
  type: z.literal("event"),
  anonymous: z.boolean().optional(),
  inputs: z.array(
    z.intersection(AbiParameter, z.object({ indexed: z.boolean().optional() }))
  ),
  name: z.string()
});
var AbiError = z.object({
  type: z.literal("error"),
  inputs: z.array(AbiParameter),
  name: z.string()
});
var AbiItemType = z.union([
  z.literal("constructor"),
  z.literal("event"),
  z.literal("error"),
  z.literal("fallback"),
  z.literal("function"),
  z.literal("receive")
]);
var Abi = z.array(
  z.union([
    AbiError,
    AbiEvent,
    // TODO: Replace code below to `z.switch` (https://github.com/colinhacks/zod/issues/2106)
    // Need to redefine `AbiFunction | AbiConstructor | AbiFallback | AbiReceive` since `z.discriminate` doesn't support `z.preprocess` on `options`
    // https://github.com/colinhacks/zod/issues/1490
    z.preprocess(
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
      z.intersection(
        z.object({
          /**
           * @deprecated use `pure` or `view` from {@link AbiStateMutability} instead
           * https://github.com/ethereum/solidity/issues/992
           */
          constant: z.boolean().optional(),
          /**
           * @deprecated Vyper used to provide gas estimates
           * https://github.com/vyperlang/vyper/issues/2151
           */
          gas: z.number().optional(),
          /**
           * @deprecated use `payable` or `nonpayable` from {@link AbiStateMutability} instead
           * https://github.com/ethereum/solidity/issues/992
           */
          payable: z.boolean().optional(),
          stateMutability: AbiStateMutability
        }),
        z.discriminatedUnion("type", [
          z.object({
            type: z.literal("function"),
            inputs: z.array(AbiParameter),
            name: z.string(),
            outputs: z.array(AbiParameter)
          }),
          z.object({
            type: z.literal("constructor"),
            inputs: z.array(AbiParameter)
          }),
          z.object({
            type: z.literal("fallback"),
            inputs: z.tuple([]).optional()
          }),
          z.object({
            type: z.literal("receive"),
            stateMutability: z.literal("payable")
          })
        ])
      )
    )
  ])
);
export {
  Abi,
  AbiConstructor,
  AbiError,
  AbiEvent,
  AbiFallback,
  AbiFunction,
  AbiItemType,
  AbiParameter,
  AbiReceive,
  AbiStateMutability,
  SolidityAddress,
  SolidityArray,
  SolidityArrayWithTuple,
  SolidityArrayWithoutTuple,
  SolidityBool,
  SolidityBytes,
  SolidityFunction,
  SolidityInt,
  SolidityString,
  SolidityTuple
};
