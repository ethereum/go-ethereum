declare const _default: {
    name: string;
    comment: string;
    url: string;
    status: string;
    gasConfig: {
        minGasLimit: {
            v: number;
            d: string;
        };
        gasLimitBoundDivisor: {
            v: number;
            d: string;
        };
        maxRefundQuotient: {
            v: number;
            d: string;
        };
    };
    gasPrices: {
        base: {
            v: number;
            d: string;
        };
        tierStep: {
            v: number[];
            d: string;
        };
        exp: {
            v: number;
            d: string;
        };
        expByte: {
            v: number;
            d: string;
        };
        sha3: {
            v: number;
            d: string;
        };
        sha3Word: {
            v: number;
            d: string;
        };
        sload: {
            v: number;
            d: string;
        };
        sstoreSet: {
            v: number;
            d: string;
        };
        sstoreReset: {
            v: number;
            d: string;
        };
        sstoreRefund: {
            v: number;
            d: string;
        };
        jumpdest: {
            v: number;
            d: string;
        };
        log: {
            v: number;
            d: string;
        };
        logData: {
            v: number;
            d: string;
        };
        logTopic: {
            v: number;
            d: string;
        };
        create: {
            v: number;
            d: string;
        };
        call: {
            v: number;
            d: string;
        };
        callStipend: {
            v: number;
            d: string;
        };
        callValueTransfer: {
            v: number;
            d: string;
        };
        callNewAccount: {
            v: number;
            d: string;
        };
        selfdestructRefund: {
            v: number;
            d: string;
        };
        memory: {
            v: number;
            d: string;
        };
        quadCoeffDiv: {
            v: number;
            d: string;
        };
        createData: {
            v: number;
            d: string;
        };
        tx: {
            v: number;
            d: string;
        };
        txCreation: {
            v: number;
            d: string;
        };
        txDataZero: {
            v: number;
            d: string;
        };
        txDataNonZero: {
            v: number;
            d: string;
        };
        copy: {
            v: number;
            d: string;
        };
        ecRecover: {
            v: number;
            d: string;
        };
        sha256: {
            v: number;
            d: string;
        };
        sha256Word: {
            v: number;
            d: string;
        };
        ripemd160: {
            v: number;
            d: string;
        };
        ripemd160Word: {
            v: number;
            d: string;
        };
        identity: {
            v: number;
            d: string;
        };
        identityWord: {
            v: number;
            d: string;
        };
        stop: {
            v: number;
            d: string;
        };
        add: {
            v: number;
            d: string;
        };
        mul: {
            v: number;
            d: string;
        };
        sub: {
            v: number;
            d: string;
        };
        div: {
            v: number;
            d: string;
        };
        sdiv: {
            v: number;
            d: string;
        };
        mod: {
            v: number;
            d: string;
        };
        smod: {
            v: number;
            d: string;
        };
        addmod: {
            v: number;
            d: string;
        };
        mulmod: {
            v: number;
            d: string;
        };
        signextend: {
            v: number;
            d: string;
        };
        lt: {
            v: number;
            d: string;
        };
        gt: {
            v: number;
            d: string;
        };
        slt: {
            v: number;
            d: string;
        };
        sgt: {
            v: number;
            d: string;
        };
        eq: {
            v: number;
            d: string;
        };
        iszero: {
            v: number;
            d: string;
        };
        and: {
            v: number;
            d: string;
        };
        or: {
            v: number;
            d: string;
        };
        xor: {
            v: number;
            d: string;
        };
        not: {
            v: number;
            d: string;
        };
        byte: {
            v: number;
            d: string;
        };
        address: {
            v: number;
            d: string;
        };
        balance: {
            v: number;
            d: string;
        };
        origin: {
            v: number;
            d: string;
        };
        caller: {
            v: number;
            d: string;
        };
        callvalue: {
            v: number;
            d: string;
        };
        calldataload: {
            v: number;
            d: string;
        };
        calldatasize: {
            v: number;
            d: string;
        };
        calldatacopy: {
            v: number;
            d: string;
        };
        codesize: {
            v: number;
            d: string;
        };
        codecopy: {
            v: number;
            d: string;
        };
        gasprice: {
            v: number;
            d: string;
        };
        extcodesize: {
            v: number;
            d: string;
        };
        extcodecopy: {
            v: number;
            d: string;
        };
        blockhash: {
            v: number;
            d: string;
        };
        coinbase: {
            v: number;
            d: string;
        };
        timestamp: {
            v: number;
            d: string;
        };
        number: {
            v: number;
            d: string;
        };
        difficulty: {
            v: number;
            d: string;
        };
        gaslimit: {
            v: number;
            d: string;
        };
        pop: {
            v: number;
            d: string;
        };
        mload: {
            v: number;
            d: string;
        };
        mstore: {
            v: number;
            d: string;
        };
        mstore8: {
            v: number;
            d: string;
        };
        sstore: {
            v: number;
            d: string;
        };
        jump: {
            v: number;
            d: string;
        };
        jumpi: {
            v: number;
            d: string;
        };
        pc: {
            v: number;
            d: string;
        };
        msize: {
            v: number;
            d: string;
        };
        gas: {
            v: number;
            d: string;
        };
        push: {
            v: number;
            d: string;
        };
        dup: {
            v: number;
            d: string;
        };
        swap: {
            v: number;
            d: string;
        };
        callcode: {
            v: number;
            d: string;
        };
        return: {
            v: number;
            d: string;
        };
        invalid: {
            v: number;
            d: string;
        };
        selfdestruct: {
            v: number;
            d: string;
        };
    };
    vm: {
        stackLimit: {
            v: number;
            d: string;
        };
        callCreateDepth: {
            v: number;
            d: string;
        };
        maxExtraDataSize: {
            v: number;
            d: string;
        };
    };
    pow: {
        minimumDifficulty: {
            v: number;
            d: string;
        };
        difficultyBoundDivisor: {
            v: number;
            d: string;
        };
        durationLimit: {
            v: number;
            d: string;
        };
        epochDuration: {
            v: number;
            d: string;
        };
        timebombPeriod: {
            v: number;
            d: string;
        };
        minerReward: {
            v: string;
            d: string;
        };
        difficultyBombDelay: {
            v: number;
            d: string;
        };
    };
};
export default _default;
