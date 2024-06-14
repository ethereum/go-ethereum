/*
  This file is part of The Colony Network.
  The Colony Network is free software: you can redistribute it and/or modify
  it under the terms of the GNU General Public License as published by
  the Free Software Foundation, either version 3 of the License, or
  (at your option) any later version.
  The Colony Network is distributed in the hope that it will be useful,
  but WITHOUT ANY WARRANTY; without even the implied warranty of
  MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
  GNU General Public License for more details.
  You should have received a copy of the GNU General Public License
  along with The Colony Network. If not, see <http://www.gnu.org/licenses/>.
*/

pragma solidity ^0.5.0;

import "./Resolver.sol";

contract EtherRouter {
  Resolver public resolver;

  function() external payable {
    if (msg.sig == 0) {
      return;
    }
    // Contracts that want to receive Ether with a plain "send" have to implement
    // a fallback function with the payable modifier. Contracts now throw if no payable
    // fallback function is defined and no function matches the signature.
    // However, 'send' only provides 2300 gas, which is not enough for EtherRouter
    // so we shortcut it here.
    //
    // Note that this means we can never have a fallback function that 'does' stuff.
    // but those only really seem to be ICOs, to date. To be explicit, there is a hard
    // decision to be made here. Either:
    // 1. Contracts that use 'send' or 'transfer' cannot send money to Colonies/ColonyNetwork
    // 2. We commit to never using a fallback function that does anything.
    //
    // We have decided on option 2 here. In the future, if we wish to have such a fallback function
    // for a Colony, it could be in a separate extension contract.

    // Get routing information for the called function
    address destination = resolver.lookup(msg.sig);

    // Make the call
    assembly {
      let size := extcodesize(destination)
      if eq(size, 0) { revert(0,0) }

      calldatacopy(mload(0x40), 0, calldatasize)
      let result := delegatecall(gas, destination, mload(0x40), calldatasize, mload(0x40), 0) // ignore-swc-112 calls are only to trusted contracts
      // as their addresses are controlled by the Resolver which we trust
      returndatacopy(mload(0x40), 0, returndatasize)
      switch result
      case 1 { return(mload(0x40), returndatasize) } // ignore-swc-113
      default { revert(mload(0x40), returndatasize) }
    }
  }

  function setResolver(address _resolver) public
  {
    resolver = Resolver(_resolver);
  }
}

