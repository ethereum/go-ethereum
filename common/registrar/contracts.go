// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package registrar

const ( // built-in contracts address source code and evm code
	UrlHintCode = "0x60c180600c6000396000f30060003560e060020a90048063300a3bbf14601557005b6024600435602435604435602a565b60006000f35b6000600084815260200190815260200160002054600160a060020a0316600014806078575033600160a060020a03166000600085815260200190815260200160002054600160a060020a0316145b607f5760bc565b336000600085815260200190815260200160002081905550806001600085815260200190815260200160002083610100811060b657005b01819055505b50505056"

	UrlHintSrc = `
contract URLhint {
	function register(uint256 _hash, uint8 idx, uint256 _url) {
		if (owner[_hash] == 0 || owner[_hash] == msg.sender) {
			owner[_hash] = msg.sender;
			url[_hash][idx] = _url;
		}
	}
	mapping (uint256 => address) owner;
	  (uint256 => uint256[256]) url;
}
	`

	HashRegCode = "0x609880600c6000396000f30060003560e060020a9004806331e12c2014601f578063d66d6c1014602b57005b6025603d565b60006000f35b6037600435602435605d565b60006000f35b600054600160a060020a0316600014605357605b565b336000819055505b565b600054600160a060020a031633600160a060020a031614607b576094565b8060016000848152602001908152602001600020819055505b505056"

	HashRegSrc = `
contract HashReg {
	function setowner() {
		if (owner == 0) {
			owner = msg.sender;
		}
	}
	function register(uint256 _key, uint256 _content) {
		if (msg.sender == owner) {
			content[_key] = _content;
  	}
	}
	address owner;
	mapping (uint256 => uint256) content;
}
`

	GlobalRegistrarSrc = `
//sol

import "owned";

contract NameRegister {
  function addr(bytes32 _name) constant returns (address o_owner) {}
  function name(address _owner) constant returns (bytes32 o_name) {}
}

contract Registrar is NameRegister {
  event Changed(bytes32 indexed name);
  event PrimaryChanged(bytes32 indexed name, address indexed addr);

  function owner(bytes32 _name) constant returns (address o_owner) {}
  function addr(bytes32 _name) constant returns (address o_address) {}
  function subRegistrar(bytes32 _name) constant returns (address o_subRegistrar) {}
  function content(bytes32 _name) constant returns (bytes32 o_content) {}

  function name(address _owner) constant returns (bytes32 o_name) {}
}

contract GlobalRegistrar is Registrar {
  struct Record {
    address owner;
    address primary;
    address subRegistrar;
    bytes32 content;
    uint value;
    uint renewalDate;
  }

  function Registrar() {
    // TODO: Populate with hall-of-fame.
  }

  function reserve(bytes32 _name) {
    // Don't allow the same name to be overwritten.
    // TODO: bidding mechanism
    if (m_toRecord[_name].owner == 0) {
      m_toRecord[_name].owner = msg.sender;
      Changed(_name);
    }
  }

  /*
  TODO
  > 12 chars: free
  <= 12 chars: auction:
  1. new names are auctioned
  - 7 day period to collect all bid bytes32es + deposits
  - 1 day period to collect all bids to be considered (validity requires associated deposit to be >10% of bid)
  - all valid bids are burnt except highest - difference between that and second highest is returned to winner
  2. remember when last auctioned/renewed
  3. anyone can force renewal process:
  - 7 day period to collect all bid bytes32es + deposits
  - 1 day period to collect all bids & full amounts - bids only uncovered if sufficiently high.
  - 1% of winner burnt; original owner paid rest.
  */

  modifier onlyrecordowner(bytes32 _name) { if (m_toRecord[_name].owner == msg.sender) _ }

  function transfer(bytes32 _name, address _newOwner) onlyrecordowner(_name) {
    m_toRecord[_name].owner = _newOwner;
    Changed(_name);
  }

  function disown(bytes32 _name) onlyrecordowner(_name) {
    if (m_toName[m_toRecord[_name].primary] == _name)
    {
      PrimaryChanged(_name, m_toRecord[_name].primary);
      m_toName[m_toRecord[_name].primary] = "";
    }
    delete m_toRecord[_name];
    Changed(_name);
  }

  function setAddress(bytes32 _name, address _a, bool _primary) onlyrecordowner(_name) {
    m_toRecord[_name].primary = _a;
    if (_primary)
    {
      PrimaryChanged(_name, _a);
      m_toName[_a] = _name;
    }
    Changed(_name);
  }
  function setSubRegistrar(bytes32 _name, address _registrar) onlyrecordowner(_name) {
    m_toRecord[_name].subRegistrar = _registrar;
    Changed(_name);
  }
  function setContent(bytes32 _name, bytes32 _content) onlyrecordowner(_name) {
    m_toRecord[_name].content = _content;
    Changed(_name);
  }

  function owner(bytes32 _name) constant returns (address) { return m_toRecord[_name].owner; }
  function addr(bytes32 _name) constant returns (address) { return m_toRecord[_name].primary; }
//  function subRegistrar(bytes32 _name) constant returns (address) { return m_toRecord[_name].subRegistrar; } // TODO: bring in on next iteration.
  function register(bytes32 _name) constant returns (address) { return m_toRecord[_name].subRegistrar; }  // only possible for now
  function content(bytes32 _name) constant returns (bytes32) { return m_toRecord[_name].content; }
  function name(address _owner) constant returns (bytes32 o_name) { return m_toName[_owner]; }

  mapping (address => bytes32) m_toName;
  mapping (bytes32 => Record) m_toRecord;
}
`
	GlobalRegistrarAbi = `[{"constant":true,"inputs":[{"name":"_owner","type":"address"}],"name":"name","outputs":[{"name":"o_name","type":"bytes32"}],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"owner","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"content","outputs":[{"name":"","type":"bytes32"}],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"addr","outputs":[{"name":"","type":"address"}],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"}],"name":"reserve","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"subRegistrar","outputs":[{"name":"o_subRegistrar","type":"address"}],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"},{"name":"_newOwner","type":"address"}],"name":"transfer","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"},{"name":"_registrar","type":"address"}],"name":"setSubRegistrar","outputs":[],"type":"function"},{"constant":false,"inputs":[],"name":"Registrar","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"},{"name":"_a","type":"address"},{"name":"_primary","type":"bool"}],"name":"setAddress","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"},{"name":"_content","type":"bytes32"}],"name":"setContent","outputs":[],"type":"function"},{"constant":false,"inputs":[{"name":"_name","type":"bytes32"}],"name":"disown","outputs":[],"type":"function"},{"constant":true,"inputs":[{"name":"_name","type":"bytes32"}],"name":"register","outputs":[{"name":"","type":"address"}],"type":"function"},{"anonymous":false,"inputs":[{"indexed":true,"name":"name","type":"bytes32"}],"name":"Changed","type":"event"},{"anonymous":false,"inputs":[{"indexed":true,"name":"name","type":"bytes32"},{"indexed":true,"name":"addr","type":"address"}],"name":"PrimaryChanged","type":"event"}]`

	GlobalRegistrarCode = `0x610b408061000e6000396000f3006000357c01000000000000000000000000000000000000000000000000000000009004806301984892146100b357806302571be3146100ce5780632dff6941146100ff5780633b3b57de1461011a578063432ced041461014b5780635a3a05bd1461016257806379ce9fac1461019357806389a69c0e146101b0578063b9f37c86146101cd578063be99a980146101de578063c3d014d614610201578063d93e75731461021e578063e1fa8e841461023557005b6100c4600480359060200150610b02565b8060005260206000f35b6100df6004803590602001506109f3565b8073ffffffffffffffffffffffffffffffffffffffff1660005260206000f35b610110600480359060200150610ad4565b8060005260206000f35b61012b600480359060200150610a3e565b8073ffffffffffffffffffffffffffffffffffffffff1660005260206000f35b61015c600480359060200150610271565b60006000f35b610173600480359060200150610266565b8073ffffffffffffffffffffffffffffffffffffffff1660005260206000f35b6101aa600480359060200180359060200150610341565b60006000f35b6101c7600480359060200180359060200150610844565b60006000f35b6101d860045061026e565b60006000f35b6101fb6004803590602001803590602001803590602001506106de565b60006000f35b61021860048035906020018035906020015061092c565b60006000f35b61022f600480359060200150610429565b60006000f35b610246600480359060200150610a89565b8073ffffffffffffffffffffffffffffffffffffffff1660005260206000f35b60005b919050565b5b565b60006001600050600083815260200190815260200160002060005060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141561033d57336001600050600083815260200190815260200160002060005060000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff02191690830217905550807fa6697e974e6a320f454390be03f74955e8978f1a6971ea6730542e37b66179bc6040604090036040a25b5b50565b813373ffffffffffffffffffffffffffffffffffffffff166001600050600083815260200190815260200160002060005060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141561042357816001600050600085815260200190815260200160002060005060000160006101000a81548173ffffffffffffffffffffffffffffffffffffffff02191690830217905550827fa6697e974e6a320f454390be03f74955e8978f1a6971ea6730542e37b66179bc6040604090036040a25b505b5050565b803373ffffffffffffffffffffffffffffffffffffffff166001600050600083815260200190815260200160002060005060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614156106d95781600060005060006001600050600086815260200190815260200160002060005060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1681526020019081526020016000206000505414156105fd576001600050600083815260200190815260200160002060005060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16827ff63780e752c6a54a94fc52715dbc5518a3b4c3c2833d301a204226548a2a85456040604090036040a36000600060005060006001600050600086815260200190815260200160002060005060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600050819055505b6001600050600083815260200190815260200160002060006000820160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690556001820160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690556002820160006101000a81549073ffffffffffffffffffffffffffffffffffffffff02191690556003820160005060009055600482016000506000905560058201600050600090555050817fa6697e974e6a320f454390be03f74955e8978f1a6971ea6730542e37b66179bc6040604090036040a25b505b50565b823373ffffffffffffffffffffffffffffffffffffffff166001600050600083815260200190815260200160002060005060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141561083d57826001600050600086815260200190815260200160002060005060010160006101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908302179055508115610811578273ffffffffffffffffffffffffffffffffffffffff16847ff63780e752c6a54a94fc52715dbc5518a3b4c3c2833d301a204226548a2a85456040604090036040a383600060005060008573ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600050819055505b837fa6697e974e6a320f454390be03f74955e8978f1a6971ea6730542e37b66179bc6040604090036040a25b505b505050565b813373ffffffffffffffffffffffffffffffffffffffff166001600050600083815260200190815260200160002060005060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff16141561092657816001600050600085815260200190815260200160002060005060020160006101000a81548173ffffffffffffffffffffffffffffffffffffffff02191690830217905550827fa6697e974e6a320f454390be03f74955e8978f1a6971ea6730542e37b66179bc6040604090036040a25b505b5050565b813373ffffffffffffffffffffffffffffffffffffffff166001600050600083815260200190815260200160002060005060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff1673ffffffffffffffffffffffffffffffffffffffff1614156109ed57816001600050600085815260200190815260200160002060005060030160005081905550827fa6697e974e6a320f454390be03f74955e8978f1a6971ea6730542e37b66179bc6040604090036040a25b505b5050565b60006001600050600083815260200190815260200160002060005060000160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050610a39565b919050565b60006001600050600083815260200190815260200160002060005060010160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050610a84565b919050565b60006001600050600083815260200190815260200160002060005060020160009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050610acf565b919050565b600060016000506000838152602001908152602001600020600050600301600050549050610afd565b919050565b6000600060005060008373ffffffffffffffffffffffffffffffffffffffff168152602001908152602001600020600050549050610b3b565b91905056`
)
