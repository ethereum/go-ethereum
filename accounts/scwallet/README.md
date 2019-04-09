# Using the smartcard wallet

## Requirements

  * A USB smartcard reader
  * A keycard that supports the status app
  * PCSCD version 4.3 running on your system **Only version 4.3 is currently supported**

## Preparing the smartcard

  **WARNING: FOILLOWING THESE INSTRUCTIONS WILL DESTROY THE MASTER KEY ON YOUR CARD. ONLY PROCEED IF NO FUNDS ARE ASSOCIATED WITH THESE ACCOUNTS**

  You can use status' [keycard-cli](https://github.com/status-im/keycard-cli) and you should get version 2.1.1 of their [smartcard application](https://github.com/status-im/status-keycard/releases/download/2.1.1/keycard_v2.1.1.cap)

  You also need to make sure that the PCSC daemon is running on your system.

  Then, you can install the application to the card by typing:

  ```
  keycard install -a keycard_v2.1.cap
  ```

  Then you can initialize the application by typing:

  ```
  keycard init
  ```

  Then the card needs to be paired:

  ```
  keycard pair
  ```

  Finally, you need to have the card generate a new master key:

  ```
  keycard shell <<END
  keycard-select
  keycard-set-pairing PAIRING_KEY PAIRING_INDEX
  keycard-open-secure-channel
  keycard-verify-pin CARD_PIN
  keycard-generate-key
  END
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