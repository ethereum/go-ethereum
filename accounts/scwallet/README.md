# Using the smartcard wallet

## Requirements

  * A USB smartcard reader
  * A keycard that supports the status app

## Usage

  1. Start `geth` with the `console` command
  2. Check the card's URL by checking `personal.listWallets`:

```
  listWallets: [{
      status: "Online, can derive public keys",
      url: "pcsc://a4d73015"
  }]
```

  3. Open the wallet, you will be prompted for your pairing password, then PIN:

```
personal.openWallet("pcsc://a4d73015")
```

  4. Check that creation was successful by typing e.g. `personal`. Then use it like a regular wallet.

## Known issues

  * Starting geth with a valid card seems to make firefox crash