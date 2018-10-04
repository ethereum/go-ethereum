pragma solidity ^0.4;

contract ExpPerformanceTester {

    function testExp(int exponent, int seed, uint n) external returns (int) {
        int e = seed;
        for (uint i = 0; i < n; i += 1) {
            e = e ** exponent;
        }
        return e;
    }

    function testExpUnroll16(int exponent, int seed, uint n) external returns (int) {
        int e = seed;
        for (uint i = 0; i < n; i += 16) {
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
            e = e ** exponent;
        }
        return e;
    }

    function testNop(int exponent, int seed, uint n) external returns (int) {
        for (uint i = 0; i < n; i += 1) {}
        return seed;
    }

    function testNopUnroll16(int exponent, int seed, uint n) external returns (int) {
        for (uint i = 0; i < n; i += 16) {}
        return seed;
    }
}
