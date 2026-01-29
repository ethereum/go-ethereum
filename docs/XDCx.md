# XDCx - Decentralized Exchange Protocol

## Overview

XDCx is a decentralized exchange (DEX) protocol built into the XDC Network at the protocol level. It enables trustless trading of XRC20 tokens with on-chain order matching and settlement.

## Key Features

- **On-chain Order Book**: All orders stored on blockchain
- **Atomic Swaps**: Instant settlement without counterparty risk
- **Low Fees**: Transaction costs only
- **Relayer Network**: Decentralized order relay
- **Cross-chain Ready**: Bridge support for multi-chain assets

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                        XDCx                             │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │   Order     │  │   Trading   │  │   Matcher   │     │
│  │ Processor   │  │   State     │  │             │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐     │
│  │  Relayer    │  │    API      │  │  Events     │     │
│  │  Registry   │  │   Layer     │  │  Emitter    │     │
│  └─────────────┘  └─────────────┘  └─────────────┘     │
└─────────────────────────────────────────────────────────┘
```

## Order Types

### Limit Orders
```json
{
  "baseToken": "0x...",
  "quoteToken": "0x...",
  "side": "buy",
  "price": "1000000000000000000",
  "quantity": "5000000000000000000"
}
```

### Market Orders
- Execute at best available price
- Fill-or-kill semantics

## Order Lifecycle

1. **Creation**: User creates and signs order
2. **Submission**: Order submitted to relayer
3. **Matching**: Engine matches against order book
4. **Settlement**: Tokens transferred atomically
5. **Confirmation**: Trade recorded on chain

## Trading Pairs

### Adding a Trading Pair
1. Register through XDCx listing contract
2. Provide initial liquidity
3. Set trading parameters

### Pair Requirements
- Both tokens must be XRC20 compliant
- Minimum liquidity thresholds
- Registration fee

## Relayer System

### Becoming a Relayer
1. Stake required XDC amount
2. Deploy relayer infrastructure
3. Register in relayer contract
4. Start accepting orders

### Relayer Fees
- Configurable by relayer
- Typical: 0.1% maker, 0.2% taker
- Fee split with protocol

## API Reference

### Submit Order
```javascript
xdcx.sendOrder({
  baseToken: "0x...",
  quoteToken: "0x...",
  side: "buy",
  type: "limit",
  price: "1000000000000000000",
  quantity: "5000000000000000000",
  nonce: 1,
  signature: "0x..."
})
```

### Cancel Order
```javascript
xdcx.cancelOrder(orderId)
```

### Get Order Book
```javascript
xdcx.getOrderBook(baseToken, quoteToken)
```

### Get Trades
```javascript
xdcx.getTrades(baseToken, quoteToken, limit)
```

## Smart Contracts

### XDCx Listing
- Token pair registration
- Listing fee management
- Parameter configuration

### Relayer Registration
- Relayer stake management
- Fee configuration
- Status tracking

## Configuration

### Enable XDCx
```bash
./XDC --xdcx --xdcx.datadir /path/to/xdcx/data
```

### Node Configuration
```toml
[XDCx]
Enabled = true
DataDir = "/data/xdcx"
```

## Events

### Order Events
- `OrderCreated`
- `OrderMatched`
- `OrderCancelled`
- `OrderFilled`

### Trade Events
- `TradeExecuted`
- `TradeSettled`

## Security

1. **Signature Verification**: All orders cryptographically signed
2. **Replay Protection**: Nonce-based order uniqueness
3. **Balance Checks**: Real-time balance verification
4. **Rate Limiting**: Prevent spam attacks

## Best Practices

### For Traders
- Use limit orders for better prices
- Monitor order book depth
- Keep private keys secure

### For Relayers
- Maintain high uptime
- Competitive fee structure
- Fast order matching

## Troubleshooting

### Order Not Matching
- Check balance sufficient
- Verify price format
- Confirm token approval

### Settlement Failed
- Insufficient gas
- Token transfer restriction
- Price moved significantly

## Further Reading

- [XDCx Technical Specification](https://docs.xinfin.org/xdcx)
- [Relayer Setup Guide](https://docs.xinfin.org/relayer)
- [API Documentation](https://docs.xinfin.org/api/xdcx)
