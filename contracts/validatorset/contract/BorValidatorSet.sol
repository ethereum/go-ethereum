pragma solidity 0.5.9;

import { ValidatorSet } from "./ValidatorSet.sol";

contract BorValidatorSet is ValidatorSet {
	/// Issue this log event to signal a desired change in validator set.
	/// This will not lead to a change in active validator set until
	/// finalizeChange is called.
	///
	/// Only the last log event of any block can take effect.
	/// If a signal is issued while another is being finalized it may never
	/// take effect.
	///
	/// _parentHash here should be the parent block hash, or the
	/// signal will not be recognized.
	event InitiateChange(bytes32 indexed _parentHash, address[] _newSet);

	/// Called when an initiated change reaches finality and is activated.
	/// Only valid when msg.sender == SYSTEM (EIP96, 2**160 - 2).
	///
	/// Also called when the contract is first enabled for consensus. In this case,
	/// the "change" finalized is the activation of the initial set.
	function finalizeChange() external {
	    
	}

	/// Reports benign misbehavior of validator of the current validator set
	/// (e.g. validator offline).
	function reportBenign(address validator, uint256 blockNumber) external {
	    
	}

	/// Reports malicious misbehavior of validator of the current validator set
	/// and provides proof of that misbehavor, which varies by engine
	/// (e.g. double vote).
	function reportMalicious(address validator, uint256 blockNumber, bytes calldata proof) external {
	    
	}

	/// Get current validator set (last enacted or initial if no changes ever made) with current stake.
	function getValidators() external view returns (address[] memory, uint256[] memory) {
	    address[] memory d = new address[](4);
	    d[0] = 0x9fB29AAc15b9A4B7F17c3385939b007540f4d791;
	    d[1] = 0x96C42C56fdb78294F96B0cFa33c92bed7D75F96a;
			d[2] = 0x7D58F677794ECdB751332c9A507993dB1b008874;
	    d[3] = 0xE4F1A86989758D4aC65671855B9a29B843bb865D;
	    uint256[] memory p = new uint256[](4);
	    p[0] = 10;
	    p[1] = 20;
	    p[2] = 30;
	    p[3] = 40;
	    return (d, p);
	}
}