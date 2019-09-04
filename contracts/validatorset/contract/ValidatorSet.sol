pragma solidity 0.5.9;

interface ValidatorSet {
	/// Get initial validator set
	function getInitialValidators()
		external
		view
		returns (address[] memory, uint256[] memory);

	/// Get current validator set (last enacted or initial if no changes ever made) with current stake.
	function getValidators()
		external
		view
		returns (address[] memory, uint256[] memory);
}