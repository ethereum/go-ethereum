// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/**
 * @title SimpleDEX
 * @dev A simple decentralized exchange for AI agent trading
 * Supports ETH to ERC20 token swaps with a fixed rate (upgradeable via oracle)
 */

interface IERC20 {
    function totalSupply() external view returns (uint256);
    function balanceOf(address account) external view returns (uint256);
    function transfer(address recipient, uint256 amount) external returns (bool);
    function allowance(address owner, address spender) external view returns (uint256);
    function approve(address spender, uint256 amount) external returns (bool);
    function transferFrom(address sender, address recipient, uint256 amount) external returns (bool);

    event Transfer(address indexed from, address indexed to, uint256 value);
    event Approval(address indexed owner, address indexed spender, uint256 value);
}

contract SimpleDEX {
    IERC20 public token;
    address public owner;
    uint256 public ethToTokenRate; // How many tokens per 1 ETH (scaled by 1e18)

    // Security parameters
    uint256 public minSwapAmount = 0.001 ether;
    uint256 public maxSwapAmount = 10 ether;
    uint256 public dailySwapLimit = 50 ether;

    // Trading statistics
    uint256 public totalSwapsExecuted;
    uint256 public totalEthSwapped;
    uint256 public totalTokensSwapped;

    // Daily limit tracking
    mapping(uint256 => uint256) public dailySwapVolume;

    event SwapExecuted(
        address indexed trader,
        uint256 ethAmount,
        uint256 tokenAmount,
        uint256 rate,
        uint256 timestamp
    );
    event RateUpdated(uint256 oldRate, uint256 newRate, uint256 timestamp);
    event LiquidityAdded(uint256 tokenAmount, uint256 timestamp);
    event LiquidityRemoved(uint256 tokenAmount, uint256 timestamp);
    event EmergencyWithdraw(address indexed recipient, uint256 ethAmount, uint256 tokenAmount);

    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner can call this");
        _;
    }

    modifier validSwapAmount(uint256 amount) {
        require(amount >= minSwapAmount, "Swap amount too small");
        require(amount <= maxSwapAmount, "Swap amount too large");
        _;
    }

    constructor(address _token, uint256 _initialRate) {
        require(_token != address(0), "Invalid token address");
        require(_initialRate > 0, "Rate must be positive");

        token = IERC20(_token);
        owner = msg.sender;
        ethToTokenRate = _initialRate;
    }

    /**
     * @dev Main swap function - ETH to Token
     * AI agent calls this to execute trades
     */
    function swapETHForToken() external payable validSwapAmount(msg.value) {
        uint256 currentDay = block.timestamp / 1 days;

        // Check daily limit
        require(
            dailySwapVolume[currentDay] + msg.value <= dailySwapLimit,
            "Daily swap limit exceeded"
        );

        // Calculate token amount with current rate
        uint256 tokenAmount = (msg.value * ethToTokenRate) / 1e18;

        // Check liquidity
        uint256 dexBalance = token.balanceOf(address(this));
        require(dexBalance >= tokenAmount, "Insufficient DEX liquidity");

        // Update tracking
        dailySwapVolume[currentDay] += msg.value;
        totalSwapsExecuted++;
        totalEthSwapped += msg.value;
        totalTokensSwapped += tokenAmount;

        // Execute swap
        require(token.transfer(msg.sender, tokenAmount), "Token transfer failed");

        emit SwapExecuted(msg.sender, msg.value, tokenAmount, ethToTokenRate, block.timestamp);
    }

    /**
     * @dev Reverse swap - Token to ETH (for completeness)
     */
    function swapTokenForETH(uint256 tokenAmount) external {
        require(tokenAmount > 0, "Amount must be positive");

        uint256 ethAmount = (tokenAmount * 1e18) / ethToTokenRate;
        require(ethAmount >= minSwapAmount, "Swap amount too small");
        require(address(this).balance >= ethAmount, "Insufficient ETH liquidity");

        uint256 currentDay = block.timestamp / 1 days;
        require(
            dailySwapVolume[currentDay] + ethAmount <= dailySwapLimit,
            "Daily swap limit exceeded"
        );

        // Transfer tokens from user
        require(token.transferFrom(msg.sender, address(this), tokenAmount), "Token transfer failed");

        // Update tracking
        dailySwapVolume[currentDay] += ethAmount;
        totalSwapsExecuted++;
        totalEthSwapped += ethAmount;
        totalTokensSwapped += tokenAmount;

        // Send ETH to user
        (bool success, ) = msg.sender.call{value: ethAmount}("");
        require(success, "ETH transfer failed");

        emit SwapExecuted(msg.sender, ethAmount, tokenAmount, ethToTokenRate, block.timestamp);
    }

    /**
     * @dev Get current swap rate
     */
    function getCurrentRate() external view returns (uint256) {
        return ethToTokenRate;
    }

    /**
     * @dev Calculate token amount for given ETH
     */
    function calculateTokenAmount(uint256 ethAmount) external view returns (uint256) {
        return (ethAmount * ethToTokenRate) / 1e18;
    }

    /**
     * @dev Calculate ETH amount for given tokens
     */
    function calculateETHAmount(uint256 tokenAmount) external view returns (uint256) {
        return (tokenAmount * 1e18) / ethToTokenRate;
    }

    /**
     * @dev Update exchange rate (simulates oracle update)
     * In production: integrate with Chainlink or similar oracle
     */
    function updateRate(uint256 newRate) external onlyOwner {
        require(newRate > 0, "Rate must be positive");

        // Sanity check: prevent rate manipulation (max 20% change per update)
        uint256 maxChange = (ethToTokenRate * 20) / 100;
        require(
            newRate >= ethToTokenRate - maxChange && newRate <= ethToTokenRate + maxChange,
            "Rate change too large"
        );

        uint256 oldRate = ethToTokenRate;
        ethToTokenRate = newRate;

        emit RateUpdated(oldRate, newRate, block.timestamp);
    }

    /**
     * @dev Add liquidity to the DEX (tokens)
     */
    function addLiquidity(uint256 tokenAmount) external onlyOwner {
        require(tokenAmount > 0, "Amount must be positive");
        require(token.transferFrom(msg.sender, address(this), tokenAmount), "Transfer failed");

        emit LiquidityAdded(tokenAmount, block.timestamp);
    }

    /**
     * @dev Remove liquidity from the DEX
     */
    function removeLiquidity(uint256 tokenAmount) external onlyOwner {
        require(tokenAmount > 0, "Amount must be positive");
        require(token.balanceOf(address(this)) >= tokenAmount, "Insufficient balance");
        require(token.transfer(owner, tokenAmount), "Transfer failed");

        emit LiquidityRemoved(tokenAmount, block.timestamp);
    }

    /**
     * @dev Update security parameters
     */
    function updateLimits(
        uint256 _minSwap,
        uint256 _maxSwap,
        uint256 _dailyLimit
    ) external onlyOwner {
        require(_minSwap < _maxSwap, "Invalid limits");
        require(_maxSwap <= _dailyLimit, "Max swap exceeds daily limit");

        minSwapAmount = _minSwap;
        maxSwapAmount = _maxSwap;
        dailySwapLimit = _dailyLimit;
    }

    /**
     * @dev Get DEX statistics
     */
    function getStats() external view returns (
        uint256 totalSwaps,
        uint256 totalEth,
        uint256 totalTokens,
        uint256 tokenBalance,
        uint256 ethBalance
    ) {
        return (
            totalSwapsExecuted,
            totalEthSwapped,
            totalTokensSwapped,
            token.balanceOf(address(this)),
            address(this).balance
        );
    }

    /**
     * @dev Get remaining daily swap capacity
     */
    function getRemainingDailyCapacity() external view returns (uint256) {
        uint256 currentDay = block.timestamp / 1 days;
        uint256 used = dailySwapVolume[currentDay];
        return used >= dailySwapLimit ? 0 : dailySwapLimit - used;
    }

    /**
     * @dev Emergency withdraw (only owner, for safety)
     */
    function emergencyWithdraw() external onlyOwner {
        uint256 tokenBalance = token.balanceOf(address(this));
        uint256 ethBalance = address(this).balance;

        if (tokenBalance > 0) {
            require(token.transfer(owner, tokenBalance), "Token transfer failed");
        }

        if (ethBalance > 0) {
            (bool success, ) = owner.call{value: ethBalance}("");
            require(success, "ETH transfer failed");
        }

        emit EmergencyWithdraw(owner, ethBalance, tokenBalance);
    }

    /**
     * @dev Accept ETH deposits for liquidity
     */
    receive() external payable {
        // Allow ETH deposits for liquidity
    }
}
