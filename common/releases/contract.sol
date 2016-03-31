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


// ReleaseOracle is an Ethereum contract to store the current and previous
// versions of the go-ethereum implementation. Its goal is to allow Geth to
// check for new releases automatically without the need to consult a central
// repository.
//
// The contract takes a vote based approach on both assigning authorized signers
// as well as signing off on new Geth releases.
contract ReleaseOracle {
  // Votes is an internal data structure to count votes on a specific proposal
  struct Votes {
    address[] pass; // List of signers voting to pass a proposal
    address[] fail; // List of signers voting to fail a proposal
  }

  // Oracle authorization details
  mapping(address => bool) authorized; // Set of accounts allowed to vote on updating the contract
  address[]                signers;    // List of addresses currently accepted as signers

  // Various proposals being voted on
  mapping(address => Votes) authProps; // Currently running user authorization proposals
  address[]                 authPend;  // List of addresses being voted on (map indexes)

  // isSigner is a modifier to authorize contract transactions.
  modifier isSigner() {
    if (authorized[msg.sender]) {
      _
    }
  }

  // Constructor to assign the creator as the sole valid signer.
  function ReleaseOracle() {
    authorized[msg.sender] = true;
    signers.push(msg.sender);
  }

  // Signers is an accessor method to retrieve all te signers (public accessor
  // generates an indexed one, not a retreive-all version).
  function Signers() constant returns(address[]) {
    return signers;
  }

  // AuthProposals retrieves the list of addresses that authorization proposals
  // are currently being voted on.
  function AuthProposals() constant returns(address[]) {
    return authPend;
  }

  // AuthVotes retrieves the current authorization votes for a particular user
  // to promote him into the list of signers, or demote him from there.
  function AuthVotes(address user) constant returns(address[] promote, address[] demote) {
    return (authProps[user].pass, authProps[user].fail);
  }

  // Promote pitches in on a voting campaign to promote a new user to a signer
  // position.
  function Promote(address user) {
    updateStatus(user, true);
  }

  // Demote pitches in on a voting campaign to demote an authorized user from
  // its signer position.
  function Demote(address user) {
    updateStatus(user, false);
  }

  // updateStatus marks a vote for changing the status of an Ethereum user,
  // either for or against the user being an authorized signer.
  function updateStatus(address user, bool authorize) isSigner {
    // Gather the current votes and ensure we don't double vote
    Votes votes = authProps[user];
    for (uint i = 0; i < votes.pass.length; i++) {
      if (votes.pass[i] == msg.sender) {
        return;
      }
    }
    for (i = 0; i < votes.fail.length; i++) {
      if (votes.fail[i] == msg.sender) {
        return;
      }
    }
    // If no authorization proposal is open, add the user to the index for later lookups
    if (votes.pass.length == 0 && votes.fail.length == 0) {
      authPend.push(user);
    }
    // Cast the vote and return if the proposal cannot be resolved yet
    if (authorize) {
      votes.pass.push(msg.sender);
      if (votes.pass.length <= signers.length / 2) {
        return;
      }
    } else {
      votes.fail.push(msg.sender);
      if (votes.fail.length <= signers.length / 2) {
        return;
      }
    }
    // Proposal resolved in our favor, execute whatever we voted on
    if (authorize && !authorized[user]) {
      authorized[user] = true;
      signers.push(user);
    } else if (!authorize && authorized[user]) {
      authorized[user] = false;

      for (i = 0; i < signers.length; i++) {
        if (signers[i] == user) {
          signers[i] = signers[signers.length - 1];
          signers.length--;
          break;
        }
      }
    }
    // Finally delete the resolved proposal, index and garbage collect
    delete authProps[user];

    for (i = 0; i < authPend.length; i++) {
      if (authPend[i] == user) {
        authPend[i] = authPend[authPend.length - 1];
        authPend.length--;
        break;
      }
    }
  }
/*
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
    }*/
}
