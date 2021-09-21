pragma solidity 0.4.24;

contract LAbstractRegistration {
    mapping(address => uint) public RESIGN_REQUESTS;
    function getRelayerByCoinbase(address) public view returns (uint, address, uint256, uint16, address[] memory, address[] memory);
}

contract LAbstractXDCXListing {
    function getTokenStatus(address) public view returns (bool);
}

contract LAbstractTokenTRC21 {
    function issuer() public view returns (address);
}

contract Lending {
    
    // @dev collateral = 0x0 => get collaterals from COLLATERALS
    struct LendingRelayer {
        uint16 _tradeFee;
        address[] _baseTokens;
        uint256[] _terms; // seconds
        address[] _collaterals;
    }

    struct Price {
        uint256 _price;
        uint256 _blockNumber;
    }

    struct Collateral {
        uint256 _depositRate;
        uint256 _liquidationRate;
        uint256 _recallRate;
        mapping(address => Price) _price;
    }
    
    mapping(address => LendingRelayer) public LENDINGRELAYER_LIST;

    mapping(address => Collateral) public COLLATERAL_LIST;
    address[] public COLLATERALS;
    
    address[] public BASES;
    
    uint256[] public TERMS;

    address[] public ILO_COLLATERALS;

    LAbstractRegistration public Relayer;

    address public MODERATOR;

    address constant private XDCNative = 0x0000000000000000000000000000000000000001;

    LAbstractXDCXListing public XDCXListing;

    address public ORACLE_PRICE_FEEDER;

    modifier oraclePriceFeederOnly() {
        require(msg.sender == ORACLE_PRICE_FEEDER, "Oracle price feeder only.");
        _;
    }

    modifier moderatorOnly() {
        require(msg.sender == MODERATOR, "Moderator only.");
        _;
    }

    function indexOf(address[] memory addrs, address target) internal pure returns (bool){
        for (uint i = 0; i < addrs.length; i ++) {
            if (addrs[i] == target) {
                return true;
            }
        }
        return false;
    }

    function termIndexOf(uint256[] memory terms, uint256 target) internal pure returns (bool){
        for (uint i = 0; i < terms.length; i ++) {
            if (terms[i] == target) {
                return true;
            }
        }
        return false;
    }
    
    constructor (address r, address t) public {
        Relayer = LAbstractRegistration(r);
        XDCXListing = LAbstractXDCXListing(t);
        ORACLE_PRICE_FEEDER = msg.sender;
        MODERATOR = msg.sender;
    }
    
    // change Oracle Price Feeder, to support Oracle Price Service
    function changeOraclePriceFeeder(address feeder) public oraclePriceFeederOnly {
        require(feeder != address(0));
        ORACLE_PRICE_FEEDER = feeder;
    }

    function changeModerator(address moderator) public moderatorOnly {
        require(moderator != address(0));
        MODERATOR = moderator;
    }

    // add/update depositRate liquidationRate recallRate price for collateral
    function addCollateral(address token, uint256 depositRate, uint256 liquidationRate, uint256 recallRate) public moderatorOnly {
        require(depositRate >= 100 && liquidationRate > 100, "Invalid rates");
        require(depositRate > liquidationRate , "Invalid deposit rates");
        require(recallRate > depositRate, "Invalid recall rates");

        bool b = XDCXListing.getTokenStatus(token) || (token == XDCNative);
        require(b, "Invalid collateral");

        COLLATERAL_LIST[token] = Collateral({
            _depositRate: depositRate,
            _liquidationRate: liquidationRate,
            _recallRate: recallRate
        });

        if (!indexOf(COLLATERALS, token)) {
            COLLATERALS.push(token);
        }
    }

    // update price for collateral
    function setCollateralPrice(address token, address lendingToken, uint256 price) public {

        bool b = XDCXListing.getTokenStatus(token) || (token == XDCNative);
        require(b, "Invalid collateral");

        require(indexOf(BASES, lendingToken), "Invalid lending token");

        require(COLLATERAL_LIST[token]._depositRate >= 100, "Invalid collateral");

        if (indexOf(COLLATERALS, token)) {
            require(msg.sender == ORACLE_PRICE_FEEDER, "Oracle Price Feeder required");
        } else {
            LAbstractTokenTRC21 t = LAbstractTokenTRC21(token);
            require(t.issuer() == msg.sender, "Required token issuer");
        }

        COLLATERAL_LIST[token]._price[lendingToken] = Price({
            _price: price,
            _blockNumber: block.number
        });
    }

    // add/update depositRate liquidationRate recall Rate price for ILO collateral
    // ILO token is issued by a relayer
    function addILOCollateral(address token, uint256 depositRate, uint256 liquidationRate, uint256 recallRate) public {
        require(depositRate >= 100 && liquidationRate > 100, "Invalid rates");
        require(depositRate > liquidationRate , "Invalid deposit rates");
        require(recallRate > depositRate , "Invalid recall rates");

        require(!indexOf(COLLATERALS, token) , "Invalid ILO collateral");

        bool b = XDCXListing.getTokenStatus(token);
        require(b, "Invalid collateral");

        LAbstractTokenTRC21 t = LAbstractTokenTRC21(token);
        require(t.issuer() == msg.sender, "Required token issuer");

        COLLATERAL_LIST[token] = Collateral({
            _depositRate: depositRate,
            _liquidationRate: liquidationRate,
            _recallRate: recallRate
        });

        if (!indexOf(ILO_COLLATERALS, token)) {
            ILO_COLLATERALS.push(token);
        }
    }
    
    // lending tokens
    function addBaseToken(address token) public moderatorOnly {
        bool b = XDCXListing.getTokenStatus(token) || (token == XDCNative);
        require(b, "Invalid base token");
        if (!indexOf(BASES, token)) {
            BASES.push(token);
        }
    }
    
    // period of loan
    function addTerm(uint256 term) public moderatorOnly {
        require(term >= 60, "Invalid term");

        if (!termIndexOf(TERMS, term)) {
            TERMS.push(term);
        }
    }
    
    function update(address coinbase, uint16 tradeFee, address[] memory baseTokens, uint256[] memory terms, address[] memory collaterals) public {
        (, address owner,,,,) = Relayer.getRelayerByCoinbase(coinbase);
        require(owner == msg.sender, "Relayer owner required");
        require(Relayer.RESIGN_REQUESTS(coinbase) == 0, "Relayer required to close");
        require(tradeFee >= 0 && tradeFee < 1000, "Invalid trade Fee"); // 0% -> 10%
        require(baseTokens.length == terms.length, "Not valid number of terms");
        require(baseTokens.length == collaterals.length, "Not valid number of collaterals");

        // validate baseTokens
        bool b = false;
        for (uint i = 0; i < baseTokens.length; i++) {
            b = indexOf(BASES, baseTokens[i]);
            require(b == true, "Invalid lending token");
        }

        // validate terms
        for (i = 0; i < terms.length; i++) {
            b = termIndexOf(TERMS, terms[i]);
            require(b == true, "Invalid term");
        }

        // validate collaterals
        for (i = 0; i < collaterals.length; i++) {
            if (collaterals[i] != address(0)) {
                require(indexOf(ILO_COLLATERALS, collaterals[i]), "Invalid collateral");
            }
        }
        
        LENDINGRELAYER_LIST[coinbase] = LendingRelayer({
            _tradeFee: tradeFee,
            _baseTokens: baseTokens,
            _terms: terms,
            _collaterals: collaterals
        });
    }

    function updateFee(address coinbase, uint16 tradeFee) public {
        (, address owner,,,,) = Relayer.getRelayerByCoinbase(coinbase);
        require(owner == msg.sender, "Relayer owner required");
        require(Relayer.RESIGN_REQUESTS(coinbase) == 0, "Relayer required to close");
        require(tradeFee >= 0 && tradeFee < 1000, "Invalid trade Fee"); // 0% -> 10%

        LENDINGRELAYER_LIST[coinbase]._tradeFee = tradeFee;
    }


    function getLendingRelayerByCoinbase(address coinbase) public view returns (uint16, address[] memory, uint256[] memory, address[] memory) {
        return (LENDINGRELAYER_LIST[coinbase]._tradeFee,
                LENDINGRELAYER_LIST[coinbase]._baseTokens,
                LENDINGRELAYER_LIST[coinbase]._terms,
                LENDINGRELAYER_LIST[coinbase]._collaterals);
    }

    function getCollateralPrice(address token, address lendingToken) public view returns (uint256, uint256) {
        return (COLLATERAL_LIST[token]._price[lendingToken]._price,
                COLLATERAL_LIST[token]._price[lendingToken]._blockNumber);
    }
}
