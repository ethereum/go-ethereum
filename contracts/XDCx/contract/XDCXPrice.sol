pragma solidity 0.4.24;

contract XDCXPrice {
    function GetLastPrice(address base, address quote) public view returns(uint256) {
        address XDCX_LAST_PRICE_PRECOMPILED_CONTRACT = 0x0000000000000000000000000000000000000029;
        uint256[1] memory result;
        address[2] memory input;
        input[0] = base;
        input[1] = quote;
        assembly {
            // GetLastPrice precompile!
            if iszero(staticcall(not(0), XDCX_LAST_PRICE_PRECOMPILED_CONTRACT, input, 0x40, result, 0x20)) {
                revert(0, 0)
            }
        }
        return result[0];
    }

    function GetEpochPrice(address base, address quote) public view returns(uint256) {
        address XDCX_EPOCH_PRICE_PRECOMPILED_CONTRACT = 0x000000000000000000000000000000000000002A;
        uint256[1] memory result;
        address[2] memory input;
        input[0] = base;
        input[1] = quote;
        assembly {
            // GetEpochPrice precompile!
            if iszero(staticcall(not(0), XDCX_EPOCH_PRICE_PRECOMPILED_CONTRACT, input, 0x40, result, 0x20)) {
                revert(0, 0)
            }
        }
        return result[0];
    }
}


