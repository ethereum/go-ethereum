// Copyright 2020 The go-ethereum Authors
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

pragma solidity ^0.6.9;

/**
 * @title LotteryBook
 * @author Gary Rong<garyrong@ethereum.org>, Felföldi Zsolt<zsfelfoldi@ethereum.org>
 * @dev Implementation of the lottery issue book which the owner can use to
 * @dev make the payment to several receivers efficiently.
 */
contract LotteryBook {
    /*
        Events
    */
    // lotteryClaimed is emitted if some active lottery is claimed by lucky winner.
    event lotteryClaimed(bytes32 indexed id);

    // lotteryCreated is emitted if someone creates a lottery.
    event lotteryCreated(address indexed creator, bytes32 id);

    /*
        Definitions
    */
    struct lottery {
        // amount is the total amount of lottery, uint64 is totally enough which can
        // represent 18ether at most.
        uint64 amount;

        // revealNumber is the block number to reveal the lottery. If current block
        // number is larger than this number, it means this lottery is expired and
        // can be claimed by the winner.
        uint64 revealNumber;

        // salt is a random number which used to calculate the id of the lottery.
        // Lottery id is derived by keccak256(tree_root+id) so that we can store
        // many lotteries with same root hash.
        uint64 salt;

        // owner is the creator of lottery, all issued cheques should be verified
        // based on the owner field.
        address payable owner;
    }

    /*
        Public Functions
    */
    // newLottery creates a new lottery with the specified probabilistic tree
    // root hash and the corresponding lottery amount.
    //
    // @id: the id of the lottery, it's calculated by keccak256(roothash + salt)
    // @blockNumber: the specified block number at which to reveal the lottery
    // @salt: the random number which used to calculate the lottery id
    //
    // The amount of lottery is set by msg.value.
    //
    // The probabilistic tree looks like:
    //
    //       srv2   srv3
    //         \     /
    //          \   /
    //           \ /
    //   srv1   hash2
    //     \     /
    //      \   /
    //       \ /
    //      hash1  srv4
    //        \     /
    //         \   /
    //          \ /
    //       root hash
    //
    // Leaves are individual receivers, encoded as sha3(keyID) in order to preserve privacy
    // of other receivers in the tree.
    //
    // Each child has half of the probabilty of its parent. Like srv4 leave has 1/2 chance
    // to win.
    //
    // When the lottery is created, owner can start to send cheques to the receivers which
    // is included in the probabilistic tree.
    //
    // The sender should send a merkle proof path, lottery salt and a signed cheque to receiver.
    // With the given merkle proof path and salt receiver can ensure it's included in the tree
    // and the relative payment amount(total_amount * winning chance).
    //
    // The signed cheque should include these following fields:
    // * the id of lottery
    // * the specified winning hash range.
    // * the signature to cover all the above fields.
    //
    // The reason to put a winning hash range in the cheque is: the expected transaction cost
    // of claiming the winnings is proportional to the cheque amount. The real transaction cost
    // only occurs when cashing the whole deposit (so their ratio can be known in advance)
    function newLottery(bytes32 id, uint64 blockNumber, uint64 salt) public payable {
        // Ensure the given reveal block number and lottery amount is reasonable.
        require(blockNumber > block.number && msg.value > 0 && msg.value <= 1 ether, "invalid lottery settings");

        // Ensure there is no duplicated lottery.
        require(lotteries[id].revealNumber == 0, "duplicated lottery");

        lotteries[id] = lottery({amount: uint64(msg.value), revealNumber: blockNumber, salt: salt, owner: msg.sender});

        emit lotteryCreated(msg.sender, id);
    }

    // resetLottery reowns the expired lottery if no one claims or this is no winner.
    // @id: the id of the stale lottery
    // @newid: the id of new lottery for replacement
    // @newRevealNumber: the specified block number at which to reveal the lottery
    // @newSalt: the random number which used to calculate the lottery id
    function resetLottery(bytes32 id, bytes32 newid, uint64 newRevealNumber, uint64 newSalt) public payable {
        // Ensure the new reveal block number is a valid future number.
        require(newRevealNumber > block.number, "invalid lottery reset operation");

        // Ensure the lottery exists and it's expired.
        uint64 oldRevealNumber = lotteries[id].revealNumber;
        require(oldRevealNumber != 0 && oldRevealNumber+visibleBlocks < block.number, "non-existent or non-expired lottery");

        // Ensure it's called by owner.
        require(lotteries[id].owner == msg.sender, "only owner is allowed to reset lottery");

        // Ensure there is no duplicated lottery.
        require(lotteries[newid].revealNumber == 0, "duplicated lottery");

        // Now we can make sure the old lottery is expired.
        // There are a few cases can lead to this situation:
        // * the winner doesn't claim the money in 256 blocks.
        // * therer is no winner of the lottery.
        // In both cases above, the lottery owner can reown the lottery
        // and reset the parameters.
        //
        // Note these following two fields must be changed. The reason to change
        // reveal number is we need to re-activate lottery. Changing the lottery
        // salt is to ensure that the new lottery won't be reclaimed by original
        // lottery payee.
        lotteries[newid].revealNumber = newRevealNumber;
        lotteries[newid].salt = newSalt;
        lotteries[newid].amount = lotteries[id].amount;
        lotteries[newid].owner = msg.sender;
        // Increase lottery amount if caller requires
        if (msg.value > 0) {
            uint64 amount = lotteries[newid].amount;
            require(amount+uint64(msg.value) > amount, "addition overflow");
            require(amount+uint64(msg.value) <= 1 ether, "exceeds maximum lottery deposit");
            lotteries[newid].amount = amount+uint64(msg.value);
        }
        delete lotteries[id];

        emit lotteryCreated(msg.sender, newid);
    }

    // destroyLottery destorys the expired lottery and reclaim
    // the deposit inside.
    function destroyLottery(bytes32 id) public {
        // Ensure the lottery exists and it's expired.
        uint64 revealNumber = lotteries[id].revealNumber;
        require(revealNumber != 0 && revealNumber+visibleBlocks < block.number, "non-existent or non-expired lottery");

        // Ensure it's called by owner.
        address payable owner = lotteries[id].owner;
        require(owner == msg.sender, "only owner is allowed to reset lottery");

        uint64 amount = lotteries[id].amount;
        delete lotteries[id];
        owner.transfer(amount);
    }

    // claim claims the lottery if the caller can prove it's the lucky winner.
    //
    // Note: since blockhash global function can only access latest 256 block hashes,
    // so that winner has to claim the money in 1 hour! Otherwise the money is gone.
    //
    // Besides chain reorg can cause block hash change at the reveal number, it would
    // be safer to wait a few block confirmations to ensure the reveal hash is stable.
    //
    // @id: the id of lottery
    // @revealRange: the promised hash range allowed for lottery redemption
    // @sig_v: the v-value of the signature
    // @sig_r: the r-value of the signature
    // @sig_s: the s-value of the signature
    // @receiver_salt: the salt for receiver
    // @proof: the merkle proof which used to prove the msg.sender has qualification to claim
    function claim(bytes32 id, bytes4 revealRange, uint8 sig_v, bytes32 sig_r, bytes32 sig_s, uint64 receiverSalt, bytes32[] memory proof) public {
        // Ensure the lottery is existent and claimable.
        uint64 revealNumber = lotteries[id].revealNumber;

        // Valid reveal block range is: [revealNumber+1, revealNumber+256]
        require(revealNumber != 0, "non-existent lottery");
        require(revealNumber < block.number && revealNumber + visibleBlocks >= block.number, "lottery isn't claimeable or it's already stale");

        // Verify the position of sender in the probabilistic tree.
        bytes32 h = keccak256(abi.encodePacked(msg.sender, receiverSalt));

        uint256 pos; // The position of msg.sender
        for (uint8 i = 0; i < proof.length; i++) {
            bytes32 elem = proof[i];
            if (h < elem) {
                h = keccak256(abi.encodePacked(h, elem));
            } else {
                pos += uint8(1)<<i;
                h = keccak256(abi.encodePacked(elem, h));
            }
        }
        // Derive the lottery id with computed merkle root and recorded lottery salt.
        h = keccak256(abi.encodePacked(h, lotteries[id].salt));
        require(h == id, "invalid position merkle proof");

        // Verified the caller's position, now ensure it's the lucky guy.
        //
        // The winning range of caller is [pos*maxWeight/dividend, revealRange]
        // Note the revealRange is encoded in big-endian format.
        require(uint32(uint256(blockhash(revealNumber))) <= uint32(revealRange), "invalid winner proof");
        require((maxWeight>>proof.length)*pos <= uint32(uint256(blockhash(revealNumber))), "invalid winner proof");

        // We also need to ensure the revealRange not larger than the upper limit, otherwhile
        // owner can always assign a very high range to itself.
        uint32 upperlimit = uint32(maxWeight>>proof.length)*uint32(pos+1);
        require(upperlimit == 0 || upperlimit > uint32(revealRange), "invalid winner proof");

        // Verify the digital signature of the cheque.
        //
        // EIP 191 style signatures
        //
        // Arguments when calculating hash to validate
        // 1: byte(0x19) - the initial 0x19 byte
        // 2: byte(0) - the version byte (data with intended validator)
        // 3: this - the validator address
        // --  Application specific data
        // 4: id - the id of lottery which is derived by keccak256(root+salt)
        // 5: range - the promised hash range allowed for lottery redemption.
        //    Once the range is verified, the corresponding receiver is verified.
        bytes32 hash = keccak256(abi.encodePacked(byte(0x19), byte(0), this, id, revealRange));
        require(lotteries[id].owner == ecrecover(hash, sig_v, sig_r, sig_s), "invalid signature");

        // Pass cheque verification, now we can ensure:
        // * the lottery is claimable
        // * the caller is included in the probabilistic tree
        // * the caller is the lucky guy to claim lottery
        // * all information provided above is signed by owner itself.
        //
        // Transfer the whole deposit to the winner.
        msg.sender.transfer(lotteries[id].amount);

        // Delete the lottery entry to prevent double claim.
        delete(lotteries[id]);

        // Emit claimed event to notify subscribers especially owner.
        emit lotteryClaimed(id);
    }

    /*
        Fields
    */
    // deposits is the map which contains all created deposit by owner.
    mapping(bytes32=>lottery) public lotteries;

    // The maximum weight which used to calculate reveal range.
    uint64 constant maxWeight = 1<<32;

    // The version of lottery contract.
    uint64 constant public version = 0;

    // visibleBlocks is the maximum visbible recent blocks in the EVM.
    //
    // Solidity documentation:
    //
    // blockhash(uint blockNumber) returns (bytes32): hash of the given block
    // - only works for 256 most recent, excluding current, blocks
    uint64 constant public visibleBlocks = 256;
}
