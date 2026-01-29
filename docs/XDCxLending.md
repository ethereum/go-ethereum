# XDCxLending - Decentralized Lending Protocol

## Overview

XDCxLending is a decentralized lending protocol built into the XDC Network. It enables peer-to-peer lending and borrowing of XRC20 tokens with collateralized loans.

## Key Features

- **Collateralized Loans**: Secure lending with over-collateralization
- **Variable Interest Rates**: Market-driven rates
- **Automatic Liquidation**: Protect lenders from defaults
- **Multiple Terms**: Flexible loan durations
- **On-chain Settlement**: Trustless execution

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    XDCxLending                          │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Order     │  │  Lending    │  │  Liquidator │     │
│  │ Processor   │  │   State     │  │             │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Loan      │  │  Interest   │  │  Collateral │     │
│  │  Manager    │  │  Calculator │  │   Manager   │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
```

## Loan Types

### Borrow Order
Request to borrow tokens at a specific interest rate.

```json
{
  "lendingToken": "0x...",
  "collateralToken": "0x...",
  "side": "borrow",
  "interestRate": "500",
  "term": "2592000",
  "quantity": "1000000000000000000"
}
```

### Lend Order
Offer to lend tokens at a specific interest rate.

```json
{
  "lendingToken": "0x...",
  "collateralToken": "0x...",
  "side": "lend",
  "interestRate": "300",
  "term": "2592000",
  "quantity": "10000000000000000000"
}
```

## Loan Lifecycle

1. **Order Creation**: User creates borrow/lend order
2. **Matching**: Orders matched by interest rate
3. **Collateral Lock**: Borrower's collateral locked
4. **Disbursement**: Lending tokens transferred
5. **Active Loan**: Interest accrues
6. **Repayment**: Borrower repays principal + interest
7. **Collateral Release**: Collateral returned

## Collateral Management

### Collateral Ratio
- Minimum: 150%
- Liquidation: 110%

### Topup
Borrowers can add collateral to avoid liquidation.

### Liquidation
When collateral ratio falls below threshold:
1. Loan marked for liquidation
2. Collateral sold to repay lender
3. Penalty applied to borrower

## Interest Calculation

### Simple Interest
```
Interest = Principal × Rate × Time / (365 × 100)
```

### Example
- Principal: 1000 XDC
- Rate: 5% annual
- Term: 30 days
- Interest: 1000 × 5 × 30 / (365 × 100) = 4.11 XDC

## API Reference

### Create Lending Order
```javascript
xdcxlending.sendLendingOrder({
  lendingToken: "0x...",
  collateralToken: "0x...",
  side: "borrow",
  type: "limit",
  interestRate: "500",
  term: 2592000,
  quantity: "1000000000000000000",
  nonce: 1,
  signature: "0x..."
})
```

### Topup Collateral
```javascript
xdcxlending.topup(loanId, amount)
```

### Repay Loan
```javascript
xdcxlending.repay(loanId)
```

### Get Loan
```javascript
xdcxlending.getLoan(loanId)
```

### Get Order Book
```javascript
xdcxlending.getLendingOrderBook(lendingToken, collateralToken, term)
```

## Smart Contracts

### Lending Registration
- Relayer registration
- Fee configuration
- Term management

### Collateral Contract
- Collateral deposit/withdrawal
- Ratio calculation
- Liquidation trigger

## Configuration

### Enable XDCxLending
```bash
./XDC --xdcxlending --xdcxlending.datadir /path/to/lending/data
```

### Parameters
```go
type Config struct {
    DefaultTerm     uint64   // 30 days
    MinCollateral   *big.Int // 150%
    LiquidationRate *big.Int // 110%
}
```

## Events

### Lending Events
- `LendingOrderCreated`
- `LendingOrderMatched`
- `LendingOrderCancelled`

### Loan Events
- `LoanCreated`
- `LoanTopup`
- `LoanRepaid`
- `LoanLiquidated`

## Risk Management

### For Lenders
- Collateral protects against default
- Automatic liquidation mechanism
- Interest rate based on risk

### For Borrowers
- Monitor collateral ratio
- Topup before liquidation
- Choose appropriate term

## Security Considerations

1. **Price Oracle**: Collateral valuation
2. **Liquidation Delay**: Time buffer
3. **Rate Limits**: Prevent manipulation
4. **Emergency Pause**: Protocol safety

## Best Practices

### For Borrowers
- Maintain healthy collateral ratio
- Set alerts for low collateral
- Repay before term expiry

### For Lenders
- Diversify lending portfolio
- Choose appropriate interest rates
- Monitor loan status

## Troubleshooting

### Order Not Matching
- Check interest rate competitive
- Verify collateral sufficient
- Confirm token balances

### Liquidation Concerns
- Add collateral (topup)
- Monitor market prices
- Set up alerts

## Example Workflow

### Borrower
1. Deposit collateral (150% of loan)
2. Create borrow order
3. Receive lending tokens
4. Use tokens as needed
5. Repay before expiry
6. Receive collateral back

### Lender
1. Have lending tokens available
2. Create lend order
3. Tokens transferred on match
4. Receive interest during term
5. Get principal + interest on repay

## Further Reading

- [XDCxLending Specification](https://docs.xinfin.org/lending)
- [Risk Parameters](https://docs.xinfin.org/lending/risk)
- [API Documentation](https://docs.xinfin.org/api/lending)
