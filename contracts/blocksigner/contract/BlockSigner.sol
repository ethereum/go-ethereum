pragma solidity ^0.4.21;

import "./libs/SafeMath.sol";

contract BlockSigner {
    using SafeMath for uint256;

    event Sign(address _signer, uint256 _blockNumber);

    mapping(uint256 => address[]) blockSigners;

    function sign(uint256 _blockNumber) external {
        // consensus should validate all senders are validators, gas = 0
        require(block.number >= _blockNumber);
        require(block.number <= _blockNumber.add(990 * 2));
        blockSigners[_blockNumber].push(msg.sender);

        emit Sign(msg.sender, _blockNumber);
    }

    function getSigners(uint256 _blockNumber) public view returns(address[]) {
        return blockSigners[_blockNumber];
    }
}
