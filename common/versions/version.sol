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

// WARNING: WORK IN PROGRESS & UNTESTED
// 
// contract tracking versions added by designated signers.
// designed to track versions of geth (go-ethereum) recommended by the
// go-ethereum team. geth client interfaces with contract through ABI by simply
// reading the full state and then deciding on recommended version based on
// some logic (e.g. version date & number of signers).
//
// to keep things simple, the contract does not use FSM for multisig
// but rather allows any designated signer to add a version or vote for an
// existing version. this avoids need to track voting-in-progress states and
// also provides history of all past versions.
//

contract Versions {
    struct V {
        bytes32 v;
        uint64 ts;
        address[] signers;
    }

    address[] public parties; // owners/signers
    address[] public deleteAcks; // votes to suicide contract
    uint public deleteAcksReq; // number of votes needed
    V[] public versions;
 
    modifier canAccess(address addr) {
        bool access = false;
        for (uint i = 0; i < parties.length; i++) {
            if (parties[i] == addr) {
                access = true;
                break;
            }
        }
        if (access == false) {
            throw;
        }
        _
    }
	
	function Versions(address[] addrs) {
        if (addrs.length < 2) {
            throw;
        }
        
        parties = addrs;
        deleteAcksReq = (addrs.length / 2) + 1;
    }

    // TODO: use dynamic array when solidity adds proper support for returning them
    function GetVersions() returns (bytes32[10], uint64[10], uint[10]) {
        bytes32[10] memory vs;
        uint64[10] memory ts;
        uint[10] memory ss;
        for (uint i = 0; i < versions.length; i++) {
            vs[i] = versions[i].v;
            ts[i] = versions[i].ts;
            ss[i] = versions[i].signers.length;
        }
        return (vs, ts, ss);
    }

    // either submit a new version or acknowledge an existing one
    function AckVersion(bytes32 ver)
        canAccess(msg.sender)
    {
        for (uint i = 0; i < versions.length; i++) {
            if (versions[i].v == ver) {
                for (uint j = 0; j < versions[i].signers.length; j++) {
                    if (versions[i].signers[j] == msg.sender) {
                        // already signed
                        throw;
                    }
                }
                // add sender as signer of existing version
                versions[i].signers.push(msg.sender);
                return;
            }
        }
     
        // version is new, add it
        // due to dynamic array, push it first then set values
        V memory v;
        versions.push(v);
        versions[versions.length - 1].v = ver;
        // signers is dynamic array; have to extend size manually
        versions[versions.length - 1].signers.length++;
        versions[versions.length - 1].signers[0] = msg.sender;
        versions[versions.length - 1].ts = uint64(block.timestamp);
    }
    
     // remove vote for a version, if present
    function NackVersion(bytes32 ver)
        canAccess(msg.sender)
    {
        for (uint i = 0; i < versions.length; i++) {
            if (versions[i].v == ver) {
                for (uint j = 0; j < versions[i].signers.length; j++) {
                    if (versions[i].signers[j] == msg.sender) {
                        delete versions[i].signers[j];
                    }
                }
            }
        }
    }
    
    // delete-this-contract vote, suicide if enough votes
    function AckDelete()
        canAccess(msg.sender)
    {
        for (uint i = 0; i < deleteAcks.length; i++) {
            if (deleteAcks[i] == msg.sender) {
                throw; // already acked delete
            }
        }
        deleteAcks.push(msg.sender);
        if (deleteAcks.length >= deleteAcksReq) {
            suicide(msg.sender);
        }
    }
    
    // remove sender's delete-this-contract vote, if present
    function NackDelete()
        canAccess(msg.sender)
    {
        uint len = deleteAcks.length;
        for (uint i = 0; i < len; i++) {
            if (deleteAcks[i] == msg.sender) {
                if (len > 1) {
                    deleteAcks[i] = deleteAcks[len-1];
                }
                deleteAcks.length -= 1;
            }
        }
    }
}
