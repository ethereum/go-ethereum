# Using the smartcard wallet

## Requirements

  * A USB smartcard reader
  * A keycard that supports the status app

## Preparing the smartcard

  You can use status' [keycard-cli](https://github.com/status-im/keycard-cli) and you should get version 2.1 (**NOT** 2.1.1, this will be supported in a later update) of their [smartcard application](https://github.com/status-im/status-keycard/releases/download/2.1/keycard_v2.1.cap)

  You also need to make sure that the PCSC daemon is running on your system.

  Then, you can install the application to the card by typing:

  ```
  keycard install -a keycard_v2.1.cap
  ```

  Then you can initialize the application by typing:

  ```
  keycard init
  ```


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

  * Starting geth with a valid card seems to make firefox crash.