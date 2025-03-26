// SPDX-License-Identifier: LGPL-3.0-only
pragma solidity >=0.8.0;

struct Supply {
    uint256 cap;
    uint256 initialSupply;
    uint256 supply;
}

/**
 * @title WrappedA0GIBase is a precompile for wrapped a0gi(wA0GI), it enables wA0GI mint/burn native 0g token directly.
 */
interface IWrappedA0GIBase {
    /**
     * @dev set the wA0GI address.
     * It is designed to be called by governance module only so it's not implemented at EVM precompile side.
     * @param addr address of wA0GI
     */
    // function setWA0GI(address addr) external;

    /**
     * @dev get the wA0GI address.
     */
    function getWA0GI() external view returns (address);

    /**
     * @dev set the cap and initial supply for a minter.
     * It is designed to be called by governance module only so it's not implemented at EVM precompile side.
     * @param minter minter address
     * @param cap mint cap
     * @param initialSupply initial mint supply
     */
    // function setMinterCap(address minter, uint256 cap, uint256 initialSupply) external;

    /**
     * @dev get the mint supply of given address
     * @param minter minter address
     */
    function minterSupply(address minter) external view returns (Supply memory);

    /**
     * @dev mint a0gi to this precompile, add corresponding amount to minter's mint supply.
     * If sender's final mint supply exceeds its mint cap, the transaction will revert.
     * Can only be called by WA0GI.
     * @param minter minter address
     * @param amount amount to mint
     */
    function mint(address minter, uint256 amount) external;

    /**
     * @dev burn given amount of a0gi on behalf of minter, reduce corresponding amount from sender's mint supply.
     * Can only be called by WA0GI.
     * @param minter minter address
     * @param amount amount to burn
     */
    function burn(address minter, uint256 amount) external;
}
