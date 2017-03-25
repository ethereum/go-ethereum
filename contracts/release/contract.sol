// Copyright 2016 The go-ethereum Authors
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
// The contract takes a vote based approach on both assigning authorised signers
// as well as signing off on new Geth releases.
//
// Note, when a signer is demoted, the currently pending release is auto-nuked.
// The reason is to prevent suprises where a demotion actually tilts the votes
// in favor of one voter party and pushing out a new release as a consequence of
// a simple demotion.
contract ReleaseOracle {
  // Votes is an internal data structure to count votes on a specific proposal
  struct Votes {
    address[] pass; // List of signers voting to pass a proposal
    address[] fail; // List of signers voting to fail a proposal
  }

  // Version is the version details of a particular Geth release
  struct Version {
    uint32  major;  // Major version component of the release
    uint32  minor;  // Minor version component of the release
    uint32  patch;  // Patch version component of the release
    bytes20 commit; // Git SHA1 commit hash of the release

    uint64  time;  // Timestamp of the release approval
    Votes   votes; // Votes that passed this release
  }

  // Oracle authorization details
  mapping(address => bool) authorised; // Set of accounts allowed to vote on updating the contract
  address[]                voters;     // List of addresses currently accepted as signers

  // Various proposals being voted on
  mapping(address => Votes) authProps; // Currently running user authorization proposals
  address[]                 authPend;  // List of addresses being voted on (map indexes)

  Version   verProp;  // Currently proposed release being voted on
  Version[] releases; // All the positively voted releases

  // isSigner is a modifier to authorize contract transactions.
  modifier isSigner() {
    if (authorised[msg.sender]) {
      _
    }
  }

  // Constructor to assign the initial set of signers.
  function ReleaseOracle(address[] signers) {
    // If no signers were specified, assign the creator as the sole signer
    if (signers.length == 0) {
      authorised[msg.sender] = true;
      voters.push(msg.sender);
      return;
    }
    // Otherwise assign the individual signers one by one
    for (uint i = 0; i < signers.length; i++) {
      authorised[signers[i]] = true;
      voters.push(signers[i]);
    }
  }

  // signers is an accessor method to retrieve all te signers (public accessor
  // generates an indexed one, not a retrieve-all version).
  function signers() constant returns(address[]) {
    return voters;
  }

  // authProposals retrieves the list of addresses that authorization proposals
  // are currently being voted on.
  function authProposals() constant returns(address[]) {
    return authPend;
  }

  // authVotes retrieves the current authorization votes for a particular user
  // to promote him into the list of signers, or demote him from there.
  function authVotes(address user) constant returns(address[] promote, address[] demote) {
    return (authProps[user].pass, authProps[user].fail);
  }

  // currentVersion retrieves the semantic version, commit hash and release time
  // of the currently votec active release.
  function currentVersion() constant returns (uint32 major, uint32 minor, uint32 patch, bytes20 commit, uint time) {
    if (releases.length == 0) {
      return (0, 0, 0, 0, 0);
    }
    var release = releases[releases.length - 1];

    return (release.major, release.minor, release.patch, release.commit, release.time);
  }

  // proposedVersion retrieves the semantic version, commit hash and the current
  // votes for the next proposed release.
  function proposedVersion() constant returns (uint32 major, uint32 minor, uint32 patch, bytes20 commit, address[] pass, address[] fail) {
    return (verProp.major, verProp.minor, verProp.patch, verProp.commit, verProp.votes.pass, verProp.votes.fail);
  }

  // promote pitches in on a voting campaign to promote a new user to a signer
  // position.
  function promote(address user) {
    updateSigner(user, true);
  }

  // demote pitches in on a voting campaign to demote an authorised user from
  // its signer position.
  function demote(address user) {
    updateSigner(user, false);
  }

  // release votes for a particular version to be included as the next release.
  function release(uint32 major, uint32 minor, uint32 patch, bytes20 commit) {
    updateRelease(major, minor, patch, commit, true);
  }

  // nuke votes for the currently proposed version to not be included as the next
  // release. Nuking doesn't require a specific version number for simplicity.
  function nuke() {
    updateRelease(0, 0, 0, 0, false);
  }

  // updateSigner marks a vote for changing the status of an Ethereum user, either
  // for or against the user being an authorised signer.
  function updateSigner(address user, bool authorize) internal isSigner {
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
      if (votes.pass.length <= voters.length / 2) {
        return;
      }
    } else {
      votes.fail.push(msg.sender);
      if (votes.fail.length <= voters.length / 2) {
        return;
      }
    }
    // Proposal resolved in our favor, execute whatever we voted on
    if (authorize && !authorised[user]) {
      authorised[user] = true;
      voters.push(user);
    } else if (!authorize && authorised[user]) {
      authorised[user] = false;

      for (i = 0; i < voters.length; i++) {
        if (voters[i] == user) {
          voters[i] = voters[voters.length - 1];
          voters.length--;

          delete verProp; // Nuke any version proposal (no surprise releases!)
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

  // updateRelease votes for a particular version to be included as the next release,
  // or for the currently proposed release to be nuked out.
  function updateRelease(uint32 major, uint32 minor, uint32 patch, bytes20 commit, bool release) internal isSigner {
    // Skip nuke votes if no proposal is pending
    if (!release && verProp.votes.pass.length == 0) {
      return;
    }
    // Mark a new release if no proposal is pending
    if (verProp.votes.pass.length == 0) {
      verProp.major  = major;
      verProp.minor  = minor;
      verProp.patch  = patch;
      verProp.commit = commit;
    }
    // Make sure positive votes match the current proposal
    if (release && (verProp.major != major || verProp.minor != minor || verProp.patch != patch || verProp.commit != commit)) {
      return;
    }
    // Gather the current votes and ensure we don't double vote
    Votes votes = verProp.votes;
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
    // Cast the vote and return if the proposal cannot be resolved yet
    if (release) {
      votes.pass.push(msg.sender);
      if (votes.pass.length <= voters.length / 2) {
        return;
      }
    } else {
      votes.fail.push(msg.sender);
      if (votes.fail.length <= voters.length / 2) {
        return;
      }
    }
    // Proposal resolved in our favor, execute whatever we voted on
    if (release) {
      verProp.time = uint64(now);
      releases.push(verProp);
      delete verProp;
    } else {
      delete verProp;
    }
  }
}
