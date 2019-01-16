ethkey
======

ethkey is a simple command-line tool for working with Ethereum keyfiles.


# Usage

### `ethkey generate`

Generate a new keyfile.
If you want to use an existing private key to use in the keyfile, it can be 
specified by setting `--privatekey` with the location of the file containing the 
private key.


### `ethkey inspect <keyfile>`

Print various information about the keyfile.
Private key information can be printed by using the `--private` flag;
make sure to use this feature with great caution!


### `ethkey signmessage <keyfile> <message/file>`

Sign the message with a keyfile.
It is possible to refer to a file containing the message.
To sign a message contained in a file, use the `--msgfile` flag.


### `ethkey verifymessage <address> <signature> <message/file>`

Verify the signature of the message.
It is possible to refer to a file containing the message.
To sign a message contained in a file, use the --msgfile flag.


### `ethkey changepassphrase <keyfile>`

Change the passphrase of a keyfile.
use the `--newpasswordfile` to point to the new password file.


## Passphrases

For every command that uses a keyfile, you will be prompted to provide the 
passphrase for decrypting the keyfile.  To avoid this message, it is possible
to pass the passphrase by using the `--passwordfile` flag pointing to a file that
contains the passphrase.

## JSON

In case you need to output the result in a JSON format, you shall by using the `--json` flag.
