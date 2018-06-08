pragma solidity ^0.4.19;

import "./Transfers2.sol";

contract Transfers {
	mapping (address => uint) public balance;
	Transfers2 t;
	address t_addr;

	constructor() public {
		t = new Transfers2();
		t_addr = address(t);
	}

	function transfer(address _addr, uint _value) public {
		require(balance[msg.sender] >= _value);
		bytes4 sig = bytes4(keccak256("transfer(address)"));
		t_addr.call.value(_value)(sig, _addr);		
	}

	function myBalance() public returns (uint) {
		return balance[msg.sender];
	}

	function deposit() public payable returns (uint) {
		balance[msg.sender] += msg.value;
		return balance[msg.sender];
	}

	function withdraw(address _addr, uint _value) public {
		require(balance[msg.sender] >= _value);
		balance[msg.sender] -= _value;
		_addr.transfer(_value);
	}

	function withdrawMulti(address[] _addrs, uint _value) public {
		uint l = _addrs.length;
		require(balance[msg.sender] >= _value * l);
		balance[msg.sender] -= _value * l;
		for (uint i = 0; i < l; i++){
			_addrs[i].transfer(_value);
		}
	}
}