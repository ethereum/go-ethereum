import * as P from 'micro-packed';
import * as abi from "./abi/decoder.js";
import * as typed from "./typed-data.js";
// Should not be included in npm package, just for test of typescript compilation
const assertType = (_value) => { };
const BytesVal = new Uint8Array();
const BigIntVal = BigInt(0);
const StringVal = 'string';
StringVal;
const _a = Uint8Array.from([]);
_a;
// IsEmptyArray
const isEmpty = (a) => a;
assertType(isEmpty([]));
assertType(isEmpty([1]));
assertType(isEmpty(['a', 2]));
assertType(isEmpty(['a']));
assertType(isEmpty([]));
assertType(isEmpty([]));
assertType(isEmpty([]));
assertType(isEmpty([]));
assertType(isEmpty([]));
assertType(isEmpty([]));
assertType(isEmpty([]));
assertType(isEmpty(undefined));
const t = [
    {
        type: 'constructor',
        inputs: [{ name: 'a', type: 'uint256' }],
        stateMutability: 'nonpayable',
    },
];
assertType(isEmpty(t));
// Tests
assertType(abi.mapComponent({ type: 'string' }));
assertType(abi.mapComponent({ type: 'string[]' }));
assertType(abi.mapComponent({ type: 'bytes' }));
assertType(abi.mapComponent({ type: 'bytes[]' }));
assertType(abi.mapComponent({ type: 'address' }));
assertType(abi.mapComponent({ type: 'address[]' }));
assertType(abi.mapComponent({ type: 'bool' }));
assertType(abi.mapComponent({ type: 'bool[]' }));
assertType(abi.mapComponent({ type: 'uint16' }));
assertType(abi.mapComponent({ type: 'uint16[]' }));
assertType(abi.mapComponent({ type: 'int' }));
assertType(abi.mapComponent({ type: 'int[]' }));
assertType(abi.mapComponent({ type: 'int24' }));
assertType(abi.mapComponent({ type: 'int24[]' }));
assertType(abi.mapComponent({ type: 'bytes1' }));
assertType(abi.mapComponent({ type: 'bytes1[]' }));
assertType(abi.mapComponent({ type: 'bytes15' }));
assertType(abi.mapComponent({ type: 'bytes15[]' }));
// Tuples
assertType(abi.mapComponent({
    type: 'tuple',
    components: [
        { type: 'uint16', name: 'lol' },
        { type: 'string', name: 'wut' },
    ],
}));
assertType(abi.mapComponent({
    type: 'tuple',
    components: [{ type: 'uint16', name: 'lol' }, { type: 'string' }],
}));
//
assertType(abi.mapComponent({ type: 'tuple' }));
assertType(abi.mapComponent({ type: 'int25' }));
assertType(abi.mapComponent({ type: 'bytes0' }));
// Args
// If single arg -- use as is
assertType(BytesVal);
// no names -> tuple
assertType([BytesVal, BigIntVal]);
// has names -> struct
assertType({
    lol: BytesVal,
    wut: BigIntVal,
});
// WHY?!
assertType(abi.mapArgs([{ type: 'string' }]));
assertType(abi.mapArgs([{ type: 'bytes1' }]));
assertType(abi.mapArgs([{ type: 'string' }, { type: 'uint' }]));
assertType(abi.mapArgs([
    { type: 'string', name: 'lol' },
    { type: 'uint', name: 'wut' },
]));
// Without const
assertType(abi.mapArgs([
    { type: 'string', name: 'lol' },
    { type: 'uint', name: 'wut' },
]));
assertType(abi.mapArgs([{ type: 'string' }, { type: 'uint' }]));
// unfortunately, typescript cannot detect single value arr on non-const data
assertType(abi.mapArgs([{ type: 'bytes1' }]));
assertType(abi.createContract([
    {
        name: 'lol',
        type: 'function',
        inputs: [{ type: 'uint' }, { type: 'string' }],
        outputs: [{ type: 'bytes' }, { type: 'address' }],
    },
]));
assertType(abi.createContract([
    {
        name: 'lol',
        type: 'function',
        outputs: [{ type: 'bytes' }, { type: 'address' }],
    },
]));
assertType(abi.createContract([
    {
        name: 'lol',
        type: 'function',
        inputs: [],
        outputs: [{ type: 'bytes' }, { type: 'address' }],
    },
]));
assertType(abi.createContract([
    {
        name: 'lol',
        type: 'function',
        inputs: [{ type: 'uint' }, { type: 'string' }],
        outputs: [{ type: 'bytes' }, { type: 'address' }],
    },
], 1));
// Without const there is not much can be derived from abi
assertType(abi.createContract([
    {
        name: 'lol',
        type: 'function',
        inputs: [{ type: 'uint' }, { type: 'string' }],
        outputs: [{ type: 'bytes' }, { type: 'address' }],
    },
]));
const PAIR_CONTRACT = [
    {
        type: 'function',
        name: 'getReserves',
        outputs: [
            { name: 'reserve0', type: 'uint112' },
            { name: 'reserve1', type: 'uint112' },
            { name: 'blockTimestampLast', type: 'uint32' },
        ],
    },
];
assertType(abi.createContract(PAIR_CONTRACT));
const TRANSFER_EVENT = [
    {
        anonymous: false,
        inputs: [
            { indexed: true, name: 'from', type: 'address' },
            { indexed: true, name: 'to', type: 'address' },
            { indexed: false, name: 'value', type: 'uint256' },
        ],
        name: 'Transfer',
        type: 'event',
    },
];
assertType(abi.events(TRANSFER_EVENT));
// Typed data
const types = {
    Person: [
        { name: 'name', type: 'string' },
        { name: 'wallet', type: 'address' },
    ],
    Mail: [
        { name: 'from', type: 'Person' },
        { name: 'to', type: 'Person' },
        { name: 'contents', type: 'string' },
    ],
    Group: [
        { name: 'members', type: 'Person[]' },
        { name: 'owner', type: 'Person' },
    ],
    Complex0: [
        { name: 'data', type: 'string[][]' }, // Complex array type
        { name: 'info', type: 'Mail' },
    ],
    Complex1: [
        { name: 'data', type: 'string[][][]' }, // Complex array type
        { name: 'info', type: 'Mail' },
    ],
    Complex: [
        { name: 'data', type: 'string[][3][]' }, // Complex array type
        { name: 'info', type: 'Mail' },
    ],
};
assertType(1);
assertType(1);
assertType(1);
assertType(1);
assertType(1);
assertType(1);
const recursiveTypes = {
    Node: [
        { name: 'value', type: 'string' },
        { name: 'children', type: 'Node[]' },
    ],
};
assertType(1);
// const e = typed.encoder(types);
// e.encodeData('Person', { name: 'test', wallet: 'x' });
// e.sign({ primaryType: 'Person', message: { name: 'test', wallet: 'x' }, domain: {} }, '');
// e.encodeData('Person', { name: 'test', wallet: 1n }); // should fail
// e.sign({ primaryType: 'Person', message: {name: 'test'}, domain: {} }, ''); // should fail
// e.sign({ primaryType: 'Person', message: {name: 'test', wallet: '', s: 3}, domain: {} }, ''); // should fail
// constructor
abi.deployContract([{ type: 'constructor', inputs: [], stateMutability: 'nonpayable' }], '0x00');
abi.deployContract([{ type: 'constructor', stateMutability: 'nonpayable' }], '0x00');
// abi.deployContract(
//   [{ type: 'constructor', stateMutability: 'nonpayable' }] as const,
//   '0x00',
//   undefined
// ); // should fail!
abi.deployContract([{ type: 'constructor', stateMutability: 'nonpayable' }], '0x00', undefined); // if we cannot infer type - it will be 'unknown' (and user forced to provide any argument, undefined is ok)
abi.deployContract([
    {
        type: 'constructor',
        inputs: [{ name: 'a', type: 'uint256' }],
        stateMutability: 'nonpayable',
    },
], '0x00', BigInt(100));
abi.deployContract([
    {
        type: 'constructor',
        inputs: [{ name: 'a', type: 'uint256' }],
        stateMutability: 'nonpayable',
    },
], '0x00', BigInt(100));
//# sourceMappingURL=_type_test.js.map