// SPDX-License-Identifier: GPL-3.0
pragma solidity 0.8.33;

contract TicketAllocator {
    address constant SYSTEM_ADDRESS = 0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE;

    error NotEnoughTickets();
    error NotSystemContract();
    error InvalidPayment();

    struct Ticket {
        uint16 amount;
        uint256 blockNumber;
        uint256 bidPerTicket;
        address requestor; // address that requested the tickets
    }

    struct PendingBid {
        address sender;
        address requestor; // address that called RequestTickets
        uint16 amount;
        uint256 bidPerTicket;
    }

    mapping(address => Ticket[]) public queue;
    mapping(address => uint256) public head;
    mapping(address => uint16) public balance;
    mapping(address => uint256) public withdrawable; // amount available for withdrawal
    
    address[] public senders; // list of senders with >0 balance

    uint16 public constant TOTAL_TICKETS = 21;
    uint16 public constant TIMEOUT = 2;

    PendingBid[] public pendingBids;
    uint256 public pendingBlock;

    function GetBalance() public view returns (address[] memory, uint16[] memory){
        uint16[] memory res;
        res = new uint16[](senders.length);
        for (uint256 i = 0; i < senders.length; i++) {
            res[i] = balance[senders[i]];
        }
        return (senders, res);
    }

    function _addModified(
        address sender,
        address[] memory modified,
        uint256 modifiedCount
    ) private pure returns (uint256) {
        for (uint256 i = 0; i < modifiedCount; i++) {
            if (modified[i] == sender) {
                return modifiedCount;
            }
        }
        modified[modifiedCount] = sender;
        return modifiedCount + 1;
    }

    function RequestTickets(address sender, uint16 numTickets, uint256 bidPerTicket) external payable {
        // Check payment matches the declared bid
        if (msg.value != uint256(numTickets) * bidPerTicket) {
            revert InvalidPayment();
        }

        // Reset pending bids if we moved to a new block
        if (block.number != pendingBlock) {
            delete pendingBids;
            pendingBlock = block.number;
        }

        pendingBids.push(PendingBid({
            sender: sender,
            requestor: msg.sender,
            amount: numTickets,
            bidPerTicket: bidPerTicket
        }));
    }

    function withdraw() external {
        uint256 amount = withdrawable[msg.sender];
        if (amount == 0) {
            return;
        }
        withdrawable[msg.sender] = 0;
        (bool success, ) = payable(msg.sender).call{value: amount}("");
        require(success, "withdrawal failed");
    }
    
    // System call:
    // 1. allocate tickets for this block based on bids
    // 2. process used and expired tickets, refunding bids for used tickets
    // 3. return modified addresses and their balances
    fallback(bytes calldata input) external returns (bytes memory) {
        if (msg.sender != SYSTEM_ADDRESS) revert NotSystemContract();

        address[] memory usedSenders;
        uint16[] memory usedAmounts;

        if (input.length > 0) {
            (usedSenders, usedAmounts) = abi.decode(input, (address[], uint16[]));
        }

        // Track modified addresses for this system call
        address[] memory modified = new address[](senders.length + pendingBids.length);
        uint256 modifiedCount = 0;

        // Step 1: allocate tickets for this block based on bids
        if (pendingBlock == block.number && pendingBids.length > 0) {
            // Sort pending bids by bidPerTicket in descending order (bubble sort on storage)
            uint256 n = pendingBids.length;
            for (uint256 i = 0; i < n; i++) {
                for (uint256 j = i + 1; j < n; j++) {
                    if (pendingBids[j].bidPerTicket > pendingBids[i].bidPerTicket) {
                        PendingBid memory tmp = pendingBids[i];
                        pendingBids[i] = pendingBids[j];
                        pendingBids[j] = tmp;
                    }
                }
            }

            uint16 remaining = TOTAL_TICKETS;

            for (uint256 i = 0; i < n; i++) {
                PendingBid storage bid = pendingBids[i];
                if (bid.amount == 0) {
                    continue;
                }

                uint16 alloc = bid.amount;
                if (alloc > remaining) {
                    alloc = remaining;
                }

                if (alloc > 0) {
                    // Allocate tickets and keep the payment locked until used or expired
                    if (balance[bid.sender] == 0) {
                        senders.push(bid.sender);
                    }
                    queue[bid.sender].push(Ticket({
                        amount: alloc,
                        blockNumber: block.number,
                        bidPerTicket: bid.bidPerTicket,
                        requestor: bid.requestor
                    }));
                    balance[bid.sender] += alloc;
                    remaining -= alloc;

                    // mark modified
                    modifiedCount = _addModified(bid.sender, modified, modifiedCount);

                    // Add unallocated portion to withdrawal, if any
                    if (bid.amount > alloc) {
                        uint16 unfilled = bid.amount - alloc;
                        uint256 refund = uint256(unfilled) * bid.bidPerTicket;
                        withdrawable[bid.requestor] += refund;
                    }
                } else {
                    // No tickets left, full refund to withdrawal
                    uint256 refundAll = uint256(bid.amount) * bid.bidPerTicket;
                    withdrawable[bid.requestor] += refundAll;
                }
            }

            delete pendingBids;
            pendingBlock = 0;
        }

        // Step 2: process used and expired tickets, refunding bids for used tickets
        for (uint256 s = 0; s < senders.length; ) {
            address sender = senders[s];
            Ticket[] storage senderQueue = queue[sender];
            uint256 senderHead = head[sender];

            // find the number of used tickets of this sender
            uint16 used = 0;
            for (uint16 i = 0; i < usedSenders.length; i++) {
                if (usedSenders[i] == sender) {
                    used = usedAmounts[i];
                    break;
                }
            }

            while (senderHead < senderQueue.length) {
                Ticket storage ticket = senderQueue[senderHead];
                
                if (ticket.amount == 0) {
                    senderHead++;
                    continue;
                }

                // Expired tickets: burn the funds paid for these tickets
                if (ticket.blockNumber + TIMEOUT < block.number) {
                    balance[sender] -= ticket.amount;
                    ticket.amount = 0;
                    senderHead++;

                    // Burn by sending to address(0)
                    (bool success, ) = payable(address(0)).call{value: ticket.amount * ticket.bidPerTicket}("");
                    require(success, "burn failed");

                    // mark modified
                    modifiedCount = _addModified(sender, modified, modifiedCount);
                } else if (used > 0) {
                    // Used tickets: add refund to withdrawal for the requestor
                    uint16 deduct = used > ticket.amount ? ticket.amount : used;
                    ticket.amount -= deduct;
                    balance[sender] -= deduct;
                    used -= deduct;
                    uint256 refund = uint256(deduct) * ticket.bidPerTicket;
                    withdrawable[ticket.requestor] += refund;

                    // mark modified
                    modifiedCount = _addModified(sender, modified, modifiedCount);
                    
                    if (ticket.amount == 0) senderHead++;
                } else {
                    break;
                }
            }

            head[sender] = senderHead;

            // remove the sender from balance map if its balance became 0
            if (balance[sender] == 0) {
                senders[s] = senders[senders.length - 1];
                senders.pop();
                continue; // process swapped-in sender at same index (no s++)
            }
            s++;
        }

        // Step 3: return modified addresses and their balances
        address[] memory modifiedAddress = new address[](modifiedCount);
        uint16[] memory modifiedTickets = new uint16[](modifiedCount);
        for (uint256 i = 0; i < modifiedCount; i++) {
            modifiedAddress[i] = modified[i];
            modifiedTickets[i] = balance[modified[i]];
        }
        return abi.encode(modifiedAddress, modifiedTickets);
    }
}
