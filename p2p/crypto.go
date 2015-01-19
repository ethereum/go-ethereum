package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/obscuren/ecies"
	"github.com/obscuren/secp256k1-go"
)

var (
	sskLen int = 16                    // ecies.MaxSharedKeyLength(pubKey) / 2
	sigLen int = 65                    // elliptic S256
	keyLen int = 32                    // ECDSA
	msgLen int = sigLen + 3*keyLen + 1 // 162
	resLen int = 65                    //
)

// aesSecret, macSecret, egressMac, ingress
type secretRW struct {
	aesSecret, macSecret, egressMac, ingressMac []byte
}

type cryptoId struct {
	prvKey    *ecdsa.PrivateKey
	pubKey    *ecdsa.PublicKey
	pubKeyDER []byte
}

func newCryptoId(id ClientIdentity) (self *cryptoId, err error) {
	// will be at server  init
	var prvKeyDER []byte = id.PrivKey()
	if prvKeyDER == nil {
		err = fmt.Errorf("no private key for client")
		return
	}
	// initialise ecies private key via importing DER encoded keys (known via our own clientIdentity)
	var prvKey = crypto.ToECDSA(prvKeyDER)
	if prvKey == nil {
		err = fmt.Errorf("invalid private key for client")
		return
	}
	self = &cryptoId{
		prvKey: prvKey,
		// initialise public key from the imported private key
		pubKey: &prvKey.PublicKey,
		// to be created at server init shared between peers and sessions
		// for reuse, call wth ReadAt, no reset seek needed
	}
	self.pubKeyDER = id.Pubkey()
	return
}

/* startHandshake is called by peer if it initiated the connection.
 By protocol spec, the party who initiates the connection (initiator) will send an 'auth' packet
New: authInitiator -> E(remote-pubk, S(ecdhe-random, ecdh-shared-secret^nonce) || H(ecdhe-random-pubk) || pubk || nonce || 0x0)
     authRecipient -> E(remote-pubk, ecdhe-random-pubk || nonce || 0x0)

Known: authInitiator = E(remote-pubk, S(ecdhe-random, token^nonce) || H(ecdhe-random-pubk) || pubk || nonce || 0x1)
       authRecipient = E(remote-pubk, ecdhe-random-pubk || nonce || 0x1) // token found
       authRecipient = E(remote-pubk, ecdhe-random-pubk || nonce || 0x0) // token not found

The caller provides the public key of the peer as conjuctured from lookup based on IP:port, given as user input or proven by signatures. The caller must have access to persistant information about the peers, and pass the previous session token as an argument to cryptoId.

The handshake is the process by which the peers establish their connection for a session.

*/

func (self *cryptoId) startHandshake(remotePubKeyDER, sessionToken []byte) (auth []byte, initNonce []byte, randomPrvKey *ecdsa.PrivateKey, randomPubKey *ecdsa.PublicKey, err error) {
	// session init, common to both parties
	remotePubKey := crypto.ToECDSAPub(remotePubKeyDER)
	if remotePubKey == nil {
		err = fmt.Errorf("invalid remote public key")
		return
	}

	var tokenFlag byte
	if sessionToken == nil {
		// no session token found means we need to generate shared secret.
		// ecies shared secret is used as initial session token for new peers
		// generate shared key from prv and remote pubkey
		if sessionToken, err = ecies.ImportECDSA(self.prvKey).GenerateShared(ecies.ImportECDSAPublic(remotePubKey), sskLen, sskLen); err != nil {
			return
		}
		// this will not stay here ;)
		fmt.Printf("secret generated: %v %x", len(sessionToken), sessionToken)
		// tokenFlag = 0x00 // redundant
	} else {
		// for known peers, we use stored token from the previous session
		tokenFlag = 0x01
	}

	//E(remote-pubk, S(ecdhe-random, ecdh-shared-secret^nonce) || H(ecdhe-random-pubk) || pubk || nonce || 0x0)
	// E(remote-pubk, S(ecdhe-random, token^nonce) || H(ecdhe-random-pubk) || pubk || nonce || 0x1)
	// allocate msgLen long message,
	var msg []byte = make([]byte, msgLen)
	// generate sskLen long nonce
	initNonce = msg[msgLen-keyLen-1 : msgLen-1]
	// nonce = msg[msgLen-sskLen-1 : msgLen-1]
	if _, err = rand.Read(initNonce); err != nil {
		return
	}
	// create known message
	// ecdh-shared-secret^nonce for new peers
	// token^nonce for old peers
	var sharedSecret = Xor(sessionToken, initNonce)

	// generate random keypair to use for signing
	if randomPrvKey, err = crypto.GenerateKey(); err != nil {
		return
	}
	// sign shared secret (message known to both parties): shared-secret
	var signature []byte
	// signature = sign(ecdhe-random, shared-secret)
	// uses secp256k1.Sign
	if signature, err = crypto.Sign(sharedSecret, randomPrvKey); err != nil {
		return
	}
	fmt.Printf("signature generated: %v %x", len(signature), signature)

	// message
	// signed-shared-secret || H(ecdhe-random-pubk) || pubk || nonce || 0x0
	copy(msg, signature) // copy signed-shared-secret
	// H(ecdhe-random-pubk)
	copy(msg[sigLen:sigLen+keyLen], crypto.Sha3(crypto.FromECDSAPub(&randomPrvKey.PublicKey)))
	// pubkey copied to the correct segment.
	copy(msg[sigLen+keyLen:sigLen+2*keyLen], self.pubKeyDER)
	// nonce is already in the slice
	// stick tokenFlag byte to the end
	msg[msgLen-1] = tokenFlag

	fmt.Printf("plaintext message generated: %v %x", len(msg), msg)

	// encrypt using remote-pubk
	// auth = eciesEncrypt(remote-pubk, msg)

	if auth, err = crypto.Encrypt(remotePubKey, msg); err != nil {
		return
	}
	fmt.Printf("encrypted message generated: %v %x\n used pubkey: %x\n", len(auth), auth, crypto.FromECDSAPub(remotePubKey))

	return
}

// verifyAuth is called by peer if it accepted (but not initiated) the connection
func (self *cryptoId) respondToHandshake(auth, sessionToken []byte, remotePubKey *ecdsa.PublicKey) (authResp []byte, respNonce []byte, initNonce []byte, randomPrvKey *ecdsa.PrivateKey, err error) {
	var msg []byte
	fmt.Printf("encrypted message received: %v %x\n used pubkey: %x\n", len(auth), auth, crypto.FromECDSAPub(self.pubKey))
	// they prove that msg is meant for me,
	// I prove I possess private key if i can read it
	if msg, err = crypto.Decrypt(self.prvKey, auth); err != nil {
		return
	}
	fmt.Printf("\nplaintext message retrieved: %v %x\n", len(msg), msg)

	var tokenFlag byte
	if sessionToken == nil {
		// no session token found means we need to generate shared secret.
		// ecies shared secret is used as initial session token for new peers
		// generate shared key from prv and remote pubkey
		if sessionToken, err = ecies.ImportECDSA(self.prvKey).GenerateShared(ecies.ImportECDSAPublic(remotePubKey), sskLen, sskLen); err != nil {
			return
		}
		fmt.Printf("secret generated: %v %x", len(sessionToken), sessionToken)
		// tokenFlag = 0x00 // redundant
	} else {
		// for known peers, we use stored token from the previous session
		tokenFlag = 0x01
	}

	// the initiator nonce is read off the end of the message
	initNonce = msg[msgLen-keyLen-1 : msgLen-1]
	// I prove that i own prv key (to derive shared secret, and read nonce off encrypted msg) and that I own shared secret
	// they prove they own the private key belonging to ecdhe-random-pubk
	// we can now reconstruct the signed message and recover the peers pubkey
	var signedMsg = Xor(sessionToken, initNonce)
	var remoteRandomPubKeyDER []byte
	if remoteRandomPubKeyDER, err = secp256k1.RecoverPubkey(signedMsg, msg[:sigLen]); err != nil {
		return
	}
	// convert to ECDSA standard
	remoteRandomPubKey := crypto.ToECDSAPub(remoteRandomPubKeyDER)
	if remoteRandomPubKey == nil {
		err = fmt.Errorf("invalid remote public key")
		return
	}

	// now we find ourselves a long task too, fill it random
	var resp = make([]byte, resLen)
	// generate keyLen long nonce
	respNonce = msg[resLen-keyLen-1 : msgLen-1]
	if _, err = rand.Read(respNonce); err != nil {
		return
	}
	// generate random keypair for session
	if randomPrvKey, err = crypto.GenerateKey(); err != nil {
		return
	}
	// responder auth message
	// E(remote-pubk, ecdhe-random-pubk || nonce || 0x0)
	copy(resp[:keyLen], crypto.FromECDSAPub(&randomPrvKey.PublicKey))
	// nonce is already in the slice
	resp[resLen-1] = tokenFlag

	// encrypt using remote-pubk
	// auth = eciesEncrypt(remote-pubk, msg)
	// why not encrypt with ecdhe-random-remote
	if authResp, err = crypto.Encrypt(remotePubKey, resp); err != nil {
		return
	}
	return
}

func (self *cryptoId) completeHandshake(auth []byte) (respNonce []byte, remoteRandomPubKey *ecdsa.PublicKey, tokenFlag bool, err error) {
	var msg []byte
	// they prove that msg is meant for me,
	// I prove I possess private key if i can read it
	if msg, err = crypto.Decrypt(self.prvKey, auth); err != nil {
		return
	}

	respNonce = msg[resLen-keyLen-1 : resLen-1]
	var remoteRandomPubKeyDER = msg[:keyLen]
	remoteRandomPubKey = crypto.ToECDSAPub(remoteRandomPubKeyDER)
	if remoteRandomPubKey == nil {
		err = fmt.Errorf("invalid ecdh random remote public key")
		return
	}
	if msg[resLen-1] == 0x01 {
		tokenFlag = true
	}
	return
}

func (self *cryptoId) newSession(initNonce, respNonce, auth []byte, privKey *ecdsa.PrivateKey, remoteRandomPubKey *ecdsa.PublicKey) (sessionToken []byte, rw *secretRW, err error) {
	// 3) Now we can trust ecdhe-random-pubk to derive new keys
	//ecdhe-shared-secret = ecdh.agree(ecdhe-random, remote-ecdhe-random-pubk)
	var dhSharedSecret []byte
	pubKey := ecies.ImportECDSAPublic(remoteRandomPubKey)
	if dhSharedSecret, err = ecies.ImportECDSA(privKey).GenerateShared(pubKey, sskLen, sskLen); err != nil {
		return
	}
	// shared-secret = crypto.Sha3(ecdhe-shared-secret || crypto.Sha3(nonce || initiator-nonce))
	var sharedSecret = crypto.Sha3(append(dhSharedSecret, crypto.Sha3(append(respNonce, initNonce...))...))
	// token = crypto.Sha3(shared-secret)
	sessionToken = crypto.Sha3(sharedSecret)
	// aes-secret = crypto.Sha3(ecdhe-shared-secret || shared-secret)
	var aesSecret = crypto.Sha3(append(dhSharedSecret, sharedSecret...))
	// # destroy shared-secret
	// mac-secret = crypto.Sha3(ecdhe-shared-secret || aes-secret)
	var macSecret = crypto.Sha3(append(dhSharedSecret, aesSecret...))
	// # destroy ecdhe-shared-secret
	// egress-mac = crypto.Sha3(mac-secret^nonce || auth)
	var egressMac = crypto.Sha3(append(Xor(macSecret, respNonce), auth...))
	// # destroy nonce
	// ingress-mac = crypto.Sha3(mac-secret^initiator-nonce || auth),
	var ingressMac = crypto.Sha3(append(Xor(macSecret, initNonce), auth...))
	// # destroy remote-nonce
	rw = &secretRW{
		aesSecret:  aesSecret,
		macSecret:  macSecret,
		egressMac:  egressMac,
		ingressMac: ingressMac,
	}
	return
}

// should use cipher.xorBytes from crypto/cipher/xor.go for fast xor
func Xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return
}
