// Contains the message filter for fine grained subscriptions.

package whisper

import "crypto/ecdsa"

// Filter is used to subscribe to specific types of whisper messages.
type Filter struct {
	To     *ecdsa.PublicKey // Recipient of the message
	From   *ecdsa.PublicKey // Sender of the message
	Topics []Topic          // Topics to watch messages on
	Fn     func(*Message)   // Handler in case of a match
}
