pragma solidity ^0.4.19;

contract Transfers2 {
	function transfer(address _addr) public payable {
		_addr.transfer(msg.value);
	}
}