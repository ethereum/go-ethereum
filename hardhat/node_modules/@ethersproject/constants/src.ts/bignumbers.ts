import { BigNumber } from "@ethersproject/bignumber";

const NegativeOne: BigNumber = (/*#__PURE__*/BigNumber.from(-1));
const Zero: BigNumber = (/*#__PURE__*/BigNumber.from(0));
const One: BigNumber = (/*#__PURE__*/BigNumber.from(1));
const Two: BigNumber = (/*#__PURE__*/BigNumber.from(2));
const WeiPerEther: BigNumber = (/*#__PURE__*/BigNumber.from("1000000000000000000"));
const MaxUint256: BigNumber = (/*#__PURE__*/BigNumber.from("0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"));

const MinInt256: BigNumber = (/*#__PURE__*/BigNumber.from("-0x8000000000000000000000000000000000000000000000000000000000000000"));
const MaxInt256: BigNumber = (/*#__PURE__*/BigNumber.from("0x7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"));

export {
    NegativeOne,
    Zero,
    One,
    Two,
    WeiPerEther,
    MaxUint256,
    MinInt256,
    MaxInt256,
};
