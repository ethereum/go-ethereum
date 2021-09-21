pragma solidity ^0.4.24;

import "./SafeMath.sol";

contract AbstractXDCXListing {
    function getTokenStatus(address) public view returns (bool);
}

contract RelayerRegistration {
    using SafeMath for uint;

    /// @dev constructor arguments
    address public CONTRACT_OWNER;
    uint public MaximumRelayers;
    uint public MaximumTokenList;
    address constant private XDCNative = 0x0000000000000000000000000000000000000001;

    /// @dev Data types
    struct Relayer {
        uint256 _deposit;
        uint16 _tradeFee;
        address[] _fromTokens;
        address[] _toTokens;
        uint _index;
        address _owner;
    }

    /// @DEV coinbase -> relayer
    mapping(address => Relayer) public RELAYER_LIST;
    /// @dev index -> coinbase
    mapping(uint => address) public RELAYER_COINBASES;
    /// @dev coinbase -> time
    mapping(address => uint) public RESIGN_REQUESTS;
    /// @dev coinbase -> price
    mapping(address => uint256) public RELAYER_ON_SALE_LIST;

    uint public RelayerCount;
    uint256 public MinimumDeposit;
    uint public ActiveRelayerCount;

    AbstractXDCXListing private XDCXListing;

    /// @dev Events
    /// struct-mapping -> values
    event ConfigEvent(uint max_relayer, uint max_token, uint256 min_deposit);
    event RegisterEvent(uint256 deposit, uint16 tradeFee, address[] fromTokens, address[] toTokens);
    event UpdateEvent(uint256 deposit, uint16 tradeFee, address[] fromTokens, address[] toTokens);
    event UpdateFeeEvent(address coinbase, uint16 tradeFee);
    event TransferEvent(address owner, uint256 deposit, uint16 tradeFee, address[] fromTokens, address[] toTokens);
    event ResignEvent(uint deposit_release_time, uint256 deposit_amount);
    event RefundEvent(bool success, uint remaining_time, uint256 deposit_amount);

    event SellEvent(bool is_on_sale, address coinbase, uint256 price);
    event BuyEvent(bool success, address coinbase, uint256 price);

    constructor (address XDCxListing, uint maxRelayers, uint maxTokenList, uint minDeposit) public {
        XDCXListing = AbstractXDCXListing(XDCxListing);
        RelayerCount = 0;
        ActiveRelayerCount = 0;
        MaximumRelayers = maxRelayers;
        MaximumTokenList = maxTokenList;
        MinimumDeposit = minDeposit;
        CONTRACT_OWNER = msg.sender;
    }


    /// @dev Modifier
    modifier contractOwnerOnly() {
        require(msg.sender == CONTRACT_OWNER, "Contract Owner Only.");
        _;
    }

    modifier relayerOwnerOnly(address coinbase) {
        require(msg.sender == RELAYER_LIST[coinbase]._owner, "Relayer Owner Only.");
        _;
    }

    modifier onlyActiveRelayer(address coinbase) {
        require(RESIGN_REQUESTS[coinbase] == 0, "The relayer has been requested to close.");
        _;
    }

    modifier nonZeroValue() {
        require(msg.value > 0, "Transfer value must be > 0");
        _;
    }

    modifier notForSale(address coinbase) {
        require(RELAYER_ON_SALE_LIST[coinbase] == 0, "The relayer must be not currently for Sale");
        _;
    }


    /// @dev support to move to oracle service
    function changeContractOwner(address owner) public contractOwnerOnly {
        require(owner != address(0));
        CONTRACT_OWNER = owner;
    }

    /// @dev Contract Config Modifications
    function reconfigure(uint maxRelayer, uint maxToken, uint minDeposit) public contractOwnerOnly {
        require(maxRelayer >= ActiveRelayerCount);
        require(maxToken > 4 && maxToken < 1001);
        require(minDeposit > 10000);
        MaximumRelayers = maxRelayer;
        MaximumTokenList = maxToken;
        MinimumDeposit = minDeposit;

        emit ConfigEvent(MaximumRelayers,
                         MaximumTokenList,
                         MinimumDeposit);
    }

    /// @dev State-Alter Methods
    function register(address coinbase, uint16 tradeFee, address[] memory fromTokens, address[] memory toTokens) public payable {
        require(msg.sender != CONTRACT_OWNER, "Contract Owner is forbidden to create a Relayer");
        require(msg.sender != coinbase, "Coinbase and RelayerOwner address must not be the same");
        require(coinbase != CONTRACT_OWNER, "Coinbase must not be same as CONTRACT_OWNER");
        require(msg.value >= MinimumDeposit, "Minimum deposit not satisfied.");
        /// @dev valid relayer configuration
        require(tradeFee >= 0 && tradeFee < 1000, "Invalid Maker Fee");
        require(fromTokens.length <= MaximumTokenList, "Exceeding number of trade pairs");
        require(toTokens.length == fromTokens.length, "Not valid number of Pairs");

        require(RELAYER_LIST[coinbase]._deposit == 0, "Coinbase already registered.");
        require(RESIGN_REQUESTS[coinbase] == 0, "The relayer has been requested to close.");
        require(ActiveRelayerCount < MaximumRelayers, "Maximum relayers registered");

        // check valid tokens, token must pair with XDC(x/XDC)
        require(validateTokens(fromTokens, toTokens) == true, "Invalid quote tokens");

        RELAYER_COINBASES[RelayerCount] = coinbase;
        RELAYER_LIST[coinbase] = Relayer({
            _deposit: msg.value,
            _tradeFee: tradeFee,
            _fromTokens: fromTokens,
            _toTokens: toTokens,
            _index: RelayerCount,
            _owner: msg.sender
        });

        RelayerCount++;
        ActiveRelayerCount++;

        emit RegisterEvent(RELAYER_LIST[coinbase]._deposit,
                           RELAYER_LIST[coinbase]._tradeFee,
                           RELAYER_LIST[coinbase]._fromTokens,
                           RELAYER_LIST[coinbase]._toTokens);
    }


    function update(address coinbase, uint16 tradeFee, address[] memory fromTokens, address[] memory toTokens) public relayerOwnerOnly(coinbase) onlyActiveRelayer(coinbase) notForSale(coinbase) {
        require(tradeFee >= 0 && tradeFee < 1000, "Invalid Maker Fee");
        require(fromTokens.length <= MaximumTokenList, "Exceeding number of trade pairs");
        require(toTokens.length == fromTokens.length, "Not valid number of Pairs");

        require(validateTokens(fromTokens, toTokens) == true, "Invalid quote tokens");

        RELAYER_LIST[coinbase]._tradeFee = tradeFee;
        RELAYER_LIST[coinbase]._fromTokens = fromTokens;
        RELAYER_LIST[coinbase]._toTokens = toTokens;

        emit UpdateEvent(RELAYER_LIST[coinbase]._deposit,
                         RELAYER_LIST[coinbase]._tradeFee,
                         RELAYER_LIST[coinbase]._fromTokens,
                         RELAYER_LIST[coinbase]._toTokens);
    }

    function updateFee(address coinbase, uint16 tradeFee) public relayerOwnerOnly(coinbase) onlyActiveRelayer(coinbase) notForSale(coinbase) {
        require(tradeFee >= 0 && tradeFee < 1000, "Invalid Maker Fee");
        RELAYER_LIST[coinbase]._tradeFee = tradeFee;
        emit UpdateFeeEvent(coinbase, RELAYER_LIST[coinbase]._tradeFee);
    }

    // List new tokens
    function listToken(
        address coinbase,
        address fromToken,
        address toToken
    ) public relayerOwnerOnly(coinbase) onlyActiveRelayer(coinbase) notForSale(coinbase) {
        require(RELAYER_LIST[coinbase]._fromTokens.length < MaximumTokenList, "Exceeding number of trade pairs");

        require(addingTokenValidation(coinbase, fromToken, toToken) == true, "Invalid pair");

        RELAYER_LIST[coinbase]._fromTokens.push(fromToken);
        RELAYER_LIST[coinbase]._toTokens.push(toToken);

        emit UpdateEvent(RELAYER_LIST[coinbase]._deposit,
                         RELAYER_LIST[coinbase]._tradeFee,
                         RELAYER_LIST[coinbase]._fromTokens,
                         RELAYER_LIST[coinbase]._toTokens);
    }

    // delist a token
    function deListToken(
        address coinbase,
        address fromToken,
        address toToken
    ) public relayerOwnerOnly(coinbase) onlyActiveRelayer(coinbase) notForSale(coinbase) {
        (address[] memory newFromTokens, address[] memory newToTokens) = deList(coinbase, fromToken, toToken);

        require(validateTokens(newFromTokens, newToTokens) == true, "Invalid quote tokens");

        RELAYER_LIST[coinbase]._fromTokens = newFromTokens;
        RELAYER_LIST[coinbase]._toTokens = newToTokens;

        emit UpdateEvent(RELAYER_LIST[coinbase]._deposit,
                         RELAYER_LIST[coinbase]._tradeFee,
                         RELAYER_LIST[coinbase]._fromTokens,
                         RELAYER_LIST[coinbase]._toTokens);
    }

    function transfer(address coinbase, address new_owner) public relayerOwnerOnly(coinbase) onlyActiveRelayer(coinbase) notForSale(coinbase) {
        require(new_owner != address(0) && new_owner != msg.sender);
        require(RELAYER_LIST[new_owner]._owner == address(0), "Owner address must not be currently used as relayer-coinbase");

        RELAYER_LIST[coinbase]._owner = new_owner;
        emit TransferEvent(RELAYER_LIST[coinbase]._owner,
                           RELAYER_LIST[coinbase]._deposit,
                           RELAYER_LIST[coinbase]._tradeFee,
                           RELAYER_LIST[coinbase]._fromTokens,
                           RELAYER_LIST[coinbase]._toTokens);
    }


    function depositMore(address coinbase) public payable relayerOwnerOnly(coinbase) onlyActiveRelayer(coinbase) notForSale(coinbase) nonZeroValue {
        require(msg.value >= 1 ether, "At least 1 XDC is required for a deposit request");
        RELAYER_LIST[coinbase]._deposit = RELAYER_LIST[coinbase]._deposit.add(msg.value);
        emit UpdateEvent(RELAYER_LIST[coinbase]._deposit,
                         RELAYER_LIST[coinbase]._tradeFee,
                         RELAYER_LIST[coinbase]._fromTokens,
                         RELAYER_LIST[coinbase]._toTokens);
    }


    function resign(address coinbase) public relayerOwnerOnly(coinbase) notForSale(coinbase) {
        require(RELAYER_LIST[coinbase]._deposit > 0, "No relayer associated with this address");
        require(RESIGN_REQUESTS[coinbase] == 0, "Request already received");
        RESIGN_REQUESTS[coinbase] = now + 4 weeks;
        ActiveRelayerCount--;
        emit ResignEvent(RESIGN_REQUESTS[coinbase],
                         RELAYER_LIST[coinbase]._deposit);
    }


    function refund(address coinbase) public relayerOwnerOnly(coinbase) notForSale(coinbase) {
        require(RESIGN_REQUESTS[coinbase] > 0, "Request not found");
        uint256 amount = RELAYER_LIST[coinbase]._deposit;
        uint deleting_index = RELAYER_LIST[coinbase]._index;

        if (RESIGN_REQUESTS[coinbase] < now) {
            delete RELAYER_LIST[coinbase];
            delete RESIGN_REQUESTS[coinbase];
            /// @notice swap last relayer's index with the deleting relayer's index
            address last_coinbase = RELAYER_COINBASES[RelayerCount - 1];
            delete RELAYER_COINBASES[RelayerCount - 1];

            RELAYER_COINBASES[deleting_index] = last_coinbase;
            RELAYER_LIST[last_coinbase]._index = deleting_index;

            RelayerCount--;
            msg.sender.transfer(amount);
            emit RefundEvent(true,
                             0,
                             amount);
        } else {
            /// Not yet, remind user about the remaining time...
            emit RefundEvent(false,
                             RESIGN_REQUESTS[coinbase] - now,
                             amount);
        }
    }

    /// @dev SELLING/BUYING RELAYERS
    // NOTE: Putting a relayer on sale will immediately halt all the state-changing actions on this relayer...
    // ...including Update, Deposit, Resign, Refund and Transfer
    function sellRelayer(address coinbase, uint256 price) public relayerOwnerOnly(coinbase) onlyActiveRelayer(coinbase) {
        // NOTE: calling this function more than once will act like changing Price-tag
        require(price > 0, "Price tag must be different than Zero(0)");
        RELAYER_ON_SALE_LIST[coinbase] = price;
        emit SellEvent(true, coinbase, price);
    }

    function cancelSelling(address coinbase) public relayerOwnerOnly(coinbase) onlyActiveRelayer(coinbase) {
        require(RELAYER_ON_SALE_LIST[coinbase] > 0, "Relayer is not currently for sale");
        delete RELAYER_ON_SALE_LIST[coinbase];
        emit SellEvent(false, coinbase, 0);
    }

    function buyRelayer(address coinbase) public payable onlyActiveRelayer(coinbase) {
        uint256 price = RELAYER_ON_SALE_LIST[coinbase];

        require(price > 0, "Relayer is not currently for sale");
        require(msg.value == price, "Price-tag must be matched");

        address seller = RELAYER_LIST[coinbase]._owner;
        require(msg.sender != address(0) && msg.sender != seller && seller != address(0), "Address not valid");
        RELAYER_LIST[coinbase]._owner = msg.sender;

        delete RELAYER_ON_SALE_LIST[coinbase];
        seller.transfer(price);
        emit BuyEvent(true, coinbase, msg.value);
    }


    function getRelayerByCoinbase(address coinbase) public view returns (uint, address, uint256, uint16, address[] memory, address[] memory) {
        return (RELAYER_LIST[coinbase]._index,
                RELAYER_LIST[coinbase]._owner,
                RELAYER_LIST[coinbase]._deposit,
                RELAYER_LIST[coinbase]._tradeFee,
                RELAYER_LIST[coinbase]._fromTokens,
                RELAYER_LIST[coinbase]._toTokens);
    }

    function indexOf(address[] memory XDCPair, address target) internal pure returns (bool){
        for (uint i = 0; i < XDCPair.length; i ++) {
            if (XDCPair[i] == target) {
                return true;
            }
        }
        return false;
    }

    function validateTokens(address[] memory fromTokens, address[] memory toTokens) internal returns(bool) {
        uint countPair = 0;
        uint countNonPair = 0;

        address[] memory XDCPairs = new address[](fromTokens.length);
        address[] memory nonXDCPairs = new address[](fromTokens.length);

        for (uint i = 0; i < toTokens.length; i++) {
            bool b = XDCXListing.getTokenStatus(toTokens[i]) || (toTokens[i] == XDCNative);
            b = b && (XDCXListing.getTokenStatus(fromTokens[i]) || fromTokens[i] == XDCNative);
            if (!b) {
                return false;
            }
            if (toTokens[i] == XDCNative) {
                XDCPairs[countPair] = fromTokens[i];
                countPair++;
            } else {
                if (fromTokens[i] == XDCNative) {
                    XDCPairs[countPair] = toTokens[i];
                    countPair++;
                }
                nonXDCPairs[countNonPair] = toTokens[i];
                countNonPair++;
            }
        }

        for (uint j = 0; j < countNonPair; j++) {
            if (!indexOf(XDCPairs, nonXDCPairs[j])) {
                return false;
            }
        }
        return true;
    }

    // add a token validation
    function addingTokenValidation(
        address coinbase,
        address fromToken,
        address toToken
    ) internal view returns(bool){
        uint countPair = 0;

        address[] memory XDCPairs = new address[](RELAYER_LIST[coinbase]._toTokens.length + 1);

        bool b = XDCXListing.getTokenStatus(toToken) || (toToken == XDCNative);
        b = b && (XDCXListing.getTokenStatus(fromToken) || fromToken == XDCNative);
        if (!b) {
            return false;
        }

        if (fromToken == XDCNative || toToken == XDCNative) {
            return true;
        }

        // get tokens that paired with XDC
        for (uint i = 0; i < RELAYER_LIST[coinbase]._toTokens.length; i++) {
            if (RELAYER_LIST[coinbase]._toTokens[i] == XDCNative) {
                XDCPairs[countPair] = RELAYER_LIST[coinbase]._fromTokens[i];
                countPair++;
            } else {
                if (RELAYER_LIST[coinbase]._fromTokens[i] == XDCNative) {
                    XDCPairs[countPair] = RELAYER_LIST[coinbase]._toTokens[i];
                    countPair++;
                }
            }
        }
        if (!indexOf(XDCPairs, toToken)) {
            return false;
        }
        return true;
    }

    // create new arrays of base and quote tokens
    function deList(address coinbase, address fromToken, address toToken) private view returns(address[], address[]) {
        address[] memory newFromTokens = new address[](RELAYER_LIST[coinbase]._toTokens.length);
        address[] memory newToTokens = new address[](RELAYER_LIST[coinbase]._toTokens.length);
        uint count = 0;

        for (uint i = 0; i < RELAYER_LIST[coinbase]._toTokens.length; i++) {
            if (RELAYER_LIST[coinbase]._toTokens[i] != toToken ||
                RELAYER_LIST[coinbase]._fromTokens[i] != fromToken) {
                newFromTokens[count] = RELAYER_LIST[coinbase]._fromTokens[i];
                newToTokens[count] = RELAYER_LIST[coinbase]._toTokens[i];
                count++;
            }
        }

        if (count != RELAYER_LIST[coinbase]._toTokens.length) {
            address[] memory fts = new address[](newToTokens.length-1);
            address[] memory tts = new address[](newToTokens.length-1);
            for (uint j = 0; j < newToTokens.length - 1; j++) {
                fts[j] = newFromTokens[j];
                tts[j] = newToTokens[j];
            }
            return (fts, tts);
        }

        return (newFromTokens, newToTokens);
       
    }
}
