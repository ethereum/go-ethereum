pragma solidity ^0.4.21;

import "./libs/SafeMath.sol";

contract BlockSigner {
    using SafeMath for uint256;

    event Sign(address _signer, uint256 _blockNumber, bytes32 _blockHash);

    mapping(bytes32 => address[]) blockSigners;
    mapping(uint256 => bytes32[]) blocks;
    uint256 public epochNumber;

    function BlockSigner(uint256 _epochNumber) public {
        epochNumber = _epochNumber;
    }

    function sign(uint256 _blockNumber, bytes32 _blockHash) external {
        // consensus should validate all senders are validators, gas = 0
        require(block.number >= _blockNumber);
        require(block.number <= _blockNumber.add(epochNumber * 2));
        blocks[_blockNumber].push(_blockHash);
        blockSigners[_blockHash].push(msg.sender);

        emit Sign(msg.sender, _blockNumber, _blockHash);
    }

    function getSigners(bytes32 _blockHash) public view returns(address[]) {
        return blockSigners[_blockHash];
    }
}
