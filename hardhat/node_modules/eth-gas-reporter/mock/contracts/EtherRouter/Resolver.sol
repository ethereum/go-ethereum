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

contract Resolver {
  mapping (bytes4 => address) public pointers;

  function register(string memory signature, address destination) public
  {
    pointers[stringToSig(signature)] = destination;
  }

  function lookup(bytes4 sig) public view returns(address) {
    return pointers[sig];
  }

  function stringToSig(string memory signature) public pure returns(bytes4) {
    return bytes4(keccak256(abi.encodePacked(signature)));
  }
}