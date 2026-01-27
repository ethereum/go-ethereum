// SPDX-License-Identifier: GPL-3.0
pragma solidity 0.8.33;

contract TicketAllocator {
    address constant SYSTEM_ADDRESS = 0xffffFFFfFFffffffffffffffFfFFFfffFFFfFFfE;

    error NotEnoughTickets();
    error NotSystemContract();

    struct Ticket {
        uint16 amount;
        uint256 blockNumber;
    }

    mapping(address => Ticket[]) public queue;
    mapping(address => uint256) public head;
    mapping(address => uint16) public balance;
    
    address[] public senders; // list of senders with >0 balance

    uint16 public constant TOTAL_TICKETS = 21;
    uint16 public constant TIMEOUT = 2;
    
    uint256 private leftTickets;
    uint256 private lastRequestBlock;

    function requestTickets(address sender, uint16 numTickets) external {
        uint256 currentBlock = block.number;

        if (currentBlock != lastRequestBlock) {
            leftTickets = TOTAL_TICKETS;
            lastRequestBlock = currentBlock;
        }

        if (numTickets > leftTickets) {
            revert NotEnoughTickets();
        }

        if (balance[sender] == 0) {
            senders.push(sender);
        }

        queue[sender].push(Ticket({
            amount: numTickets,
            blockNumber: currentBlock
        }));

        balance[sender] += numTickets;
        leftTickets -= numTickets;
    }

    function checkBalance(address sender) public view returns(uint) {
        return balance[sender];
    }

    // System call:
    // 1. remove expired and used tickets
    //    - for used tickets, the oldest ticket is removed first
    // 3. return the overall sender balances
    fallback(bytes calldata input) external returns (bytes memory) {
        if (msg.sender != SYSTEM_ADDRESS) revert NotSystemContract();

        address[] memory usedSenders;
        uint16[] memory usedAmounts;
        
        if (input.length > 0) {
            (usedSenders, usedAmounts) = abi.decode(input, (address[], uint16[]));
        }

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

 
                if (ticket.blockNumber + TIMEOUT < block.number) {
                    // remove expired tickets (comsume used tickets if possible)
                    balance[sender] -= ticket.amount;
                    if (used > 0) {
                        // used = max(0, used - t.amount)
                        uint16 deduct = used > ticket.amount ? ticket.amount : used;
                        used -= deduct;
                    }
                    ticket.amount = 0;
                    senderHead++;
                } else if (used > 0) {
                    // remove used ticekts (for now we don't revert when the ticket amount is less than used tickets)
                    uint16 deduct = used > ticket.amount ? ticket.amount : used;
                    ticket.amount -= uint16(deduct);
                    balance[sender] -= deduct;
                    used -= deduct;
                    
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

        // return the active senders and their balances
        address[] memory activeSenders = new address[](senders.length);
        uint16[] memory activeBalances = new uint16[](senders.length);

        for (uint256 i = 0; i < senders.length; i++) {
            activeSenders[i] = senders[i];
            activeBalances[i] = balance[senders[i]];
        }

        return abi.encode(activeSenders, activeBalances);
    }
}
