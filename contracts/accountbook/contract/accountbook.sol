pragma solidity ^0.5.12;

/**
 * @title AccountBook
 * @author Gary Rong<garyrong@ethereum.org>
 * @dev Implementation of the account book which les server can use to
 * @dev process micropayments from customers.
 */
contract AccountBook {
    /*
        Events
    */

    // withdrawEvent is emitted if customer opens a request to withdraw
    // deposit.
    event withdrawEvent(address indexed addr, uint256 amount);

    // balanceChangedEvent is emitted when the deposit balance of customer
    // is changed.
    event balanceChangedEvent(address indexed addr, uint256 oldBalance, uint256 newBalance);

    /*
        Definitions
    */

    // withdrawRequest defines all necessary fields of
    // the withdraw request from customers
    struct withdrawRequest {
        uint128 amount;    // the amount requested to withdraw
        uint128 createdAt; // the created block number
    }

    /*
        Modifier
    */
    modifier onlyOwner() {
        require(msg.sender == owner);
        _;
    }

    /*
        Public Functions
    */
    constructor(uint64 _challengeTimeWindow) public {
        owner = msg.sender;

        // challengeTimeWindow is decided by owner itself. In theory
        // the longer challenge time window, the safer for owner. But
        // in the mean time, it will take longer for customers to withdraw
        // money. The inconvenience may lead to lose potentional customers.
        challengeTimeWindow = _challengeTimeWindow;
    }

    // deposit adds the amount of money into the account book
    function deposit() payable public {
        uint256 balance = deposits[msg.sender];
        require(balance + msg.value > balance, "addition overflow or zero deposit");
        deposits[msg.sender] += msg.value;

        emit balanceChangedEvent(msg.sender, balance, balance+msg.value);
    }

    // withdraw opens a request to withdraw the deposit from the
    // account book. Caller has to wait challengeTimeWindow blocks
    // time which leaves enough time window for owner to challenge
    // the withdraw amount.
    // @amount: the amount of deposit to withdraw
    function withdraw(uint256 amount) public {
        // Ensure it's a meaningful withdraw request
        if (amount == 0) {
            return;
        }
        // Ensure the customer has enough deposit to withdraw
        if (deposits[msg.sender] < amount) {
            return;
        }
        // Account book can only process one withdraw request
        // at the same time.
        if (withdrawRequests[msg.sender].amount > 0) {
            return;
        }
        // Convert the amount and block number into uin128 so that
        // only 1 slot is necessary(we can save 20,000 gas cost).
        withdrawRequests[msg.sender] = withdrawRequest({amount: uint128(amount), createdAt: uint128(block.number)});
        emit withdrawEvent(msg.sender, amount);
    }

    // claim withdraws the deposit from the account book which
    // has passed the challenge period.
    function claim() public {
        uint128 amount = withdrawRequests[msg.sender].amount;

        // Short circuit if the withdrawal amount is zero.
        // There are several situations can lead to this case:
        // * there is no withdrawal request at all
        // * there is no withdrawable deposit since all of the deposit
        //   is used and cashed by owner of the account book
        if (amount == 0) {
            return;
        }
        // Ensure the request has passed the challenge period.
        if (block.number - withdrawRequests[msg.sender].createdAt < challengeTimeWindow) {
            return;
        }
        // Decrease the balance of customer before transfer.
        uint256 balance = deposits[msg.sender];
        deposits[msg.sender] -= amount;
        delete withdrawRequests[msg.sender]; // Release the withdraw lock.
        msg.sender.transfer(amount);
        emit balanceChangedEvent(msg.sender, balance, balance-amount);
    }

    // cash claims the specified amount of money from payer's deposit with
    // offchain signature.
    //
    // For cash operation, since it's called by owner itself, so that it's
    // unnecessary to leave a challenge time window.
    //
    // This function can be called in two cases:
    // * the owner of account book thinks there are too many payments made
    //   by customers
    // * if the customer opens a withdraw request which tries to withdraw
    //   some spent money, it's a way to challenge the request.
    //
    // @payer: the address of payer who has made a few micropayments off-chain.
    // @amount : the amount of money owner wants to cash
    // @sig_v : the v-value of the signature
    // @sig_r : the r-value of the signature
    // @sig_s : the s-value of the signature
    function cash(address payer, uint256 amount, uint8 sig_v, bytes32 sig_r, bytes32 sig_s) onlyOwner public {
        // In order to prevent the owner to double-cash the
        // cheque of customer, we record the cashed amount
        // in contract. Only higher signed amount can make
        // a valid cash operation.
        require(amount > paids[payer]);

        // Check the digital signature of the cheque.
        //
        // EIP 191 style signatures
        //
        // Arguments when calculating hash to validate
        // 1: byte(0x19) - the initial 0x19 byte
        // 2: byte(0) - the version byte (data with intended validator)
        // 3: this - the validator address
        // --  Application specific data
        // 4: amount the amount of paid money
        // 5: chainID(todo need istanbul fork)
        bytes32 hash = keccak256(abi.encodePacked(byte(0x19), byte(0), this, amount));
        require(payer == ecrecover(hash, sig_v, sig_r, sig_s));

        // Cash all cheques to owner's account directly.
        uint256 balance = deposits[payer];
        uint256 newBalance;
        uint256 diff = amount - paids[payer];

        // Move the money into owner's address
        if (balance >= diff) {
            // Payer has enough deposit to cover all spends
            newBalance = balance - diff;
            deposits[payer] = newBalance;
            owner.transfer(diff);
            emit balanceChangedEvent(payer, balance, newBalance);
        } else if (balance > 0) {
            // It can happen that payer doesn't have enough deposit to cover spends.
            // In theory owner should reject all "invalid" cheques off-chain. But if
            // some errors occur that owner accept the "useless" cheque, we still support
            // owner to cash all "spent" money.
            delete deposits[payer];

            // Transfer all remaing money into owner's pocket.
            owner.transfer(balance);
            emit balanceChangedEvent(payer, balance, 0);
        }
        paids[payer] = amount; // Record all cashed amount in order to prevent "double-cash"

        // If customer want to withdraw some spent money, reject
        // it by decreaing the amount or just delete the request.
        // It's the challenge action for invalid withdraw request.
        if (withdrawRequests[payer].amount > newBalance) {
            if (newBalance == 0) {
                delete withdrawRequests[payer];
            } else {
                withdrawRequests[payer].amount = uint128(newBalance);
            }
        }
    }

    /*
        Fields
    */
    // paids is the map which contains the cumulative paid amount
    // in wei from each customer
    mapping(address => uint256) public paids;

    // deposits is the map which contains the deposit of customers.
    // Customers can withdraw unused deposit back.
    mapping(address => uint256) public deposits;

    // withdrawRequests is the map which contains all withdraw
    // requests from customers, no matter to withdraw all deposit
    // or a part.
    mapping(address => withdrawRequest) public withdrawRequests;

    // owner is the address of the account book owner(the address
    // of les server).
    address payable public owner;

    // challengeTimeWindow is the maximum time that owner can perform
    // challenge when customer requests deposit withdrawal. It's count
    // by block number.
    uint64 public challengeTimeWindow;
}
