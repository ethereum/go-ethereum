package p2p

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	ethlogger "github.com/ethereum/go-ethereum/logger"
	"github.com/obscuren/ecies"
)

var clogger = ethlogger.NewLogger("CRYPTOID")

const (
	sskLen int = 16  // ecies.MaxSharedKeyLength(pubKey) / 2
	sigLen int = 65  // elliptic S256
	pubLen int = 64  // 512 bit pubkey in uncompressed representation without format byte
	shaLen int = 32  // hash length (for nonce etc)
	msgLen int = 194 // sigLen + shaLen + pubLen + shaLen + 1 = 194
	resLen int = 97  // pubLen + shaLen + 1
	iHSLen int = 307 // size of the final ECIES payload sent as initiator's handshake
	rHSLen int = 210 // size of the final ECIES payload sent as receiver's handshake
)

// secretRW implements a message read writer with encryption and authentication
// it is initialised by cryptoId.Run() after a successful crypto handshake
// aesSecret, macSecret, egressMac, ingress
type secretRW struct {
	aesSecret, macSecret, egressMac, ingressMac []byte
}

type hexkey []byte

func (self hexkey) String() string {
	return fmt.Sprintf("(%d) %x", len(self), []byte(self))
}

/*
NewSecureSession(connection, privateKey, remotePublicKey, sessionToken, initiator) is called when the peer connection starts to set up a secure session by performing a crypto handshake.

 connection is (a buffered) network connection.

 privateKey is the local client's private key (*ecdsa.PrivateKey)

 remotePublicKey is the remote peer's node Id ([]byte)

 sessionToken is the token from the previous session with this same peer. Nil if no token is found.

 initiator is a boolean flag. True if the node is the initiator of the connection (ie., remote is an outbound peer reached by dialing out). False if the connection was established by accepting a call from the remote peer via a listener.

 It returns a secretRW which implements the MsgReadWriter interface.
*/

func NewSecureSession(conn io.ReadWriter, prvKey *ecdsa.PrivateKey, remotePubKeyS []byte, sessionToken []byte, initiator bool) (token []byte, rw *secretRW, err error) {
	var auth, initNonce, recNonce []byte
	var read int
	var randomPrivKey *ecdsa.PrivateKey
	var remoteRandomPubKey *ecdsa.PublicKey
	clogger.Debugf("attempting session with %v", hexkey(remotePubKeyS))
	if initiator {
		if auth, initNonce, randomPrivKey, _, err = startHandshake(prvKey, remotePubKeyS, sessionToken); err != nil {
			return
		}
		if sessionToken != nil {
			clogger.Debugf("session-token: %v", hexkey(sessionToken))
		}
		clogger.Debugf("initiator-nonce: %v", hexkey(initNonce))
		clogger.Debugf("initiator-random-private-key: %v", hexkey(crypto.FromECDSA(randomPrivKey)))
		randomPublicKeyS, _ := ExportPublicKey(&randomPrivKey.PublicKey)
		clogger.Debugf("initiator-random-public-key: %v", hexkey(randomPublicKeyS))

		if _, err = conn.Write(auth); err != nil {
			return
		}
		clogger.Debugf("initiator handshake (sent to %v):\n%v", hexkey(remotePubKeyS), hexkey(auth))
		var response []byte = make([]byte, rHSLen)
		if read, err = conn.Read(response); err != nil || read == 0 {
			return
		}
		if read != rHSLen {
			err = fmt.Errorf("remote receiver's handshake has invalid length. expect %v, got %v", rHSLen, read)
			return
		}
		// write out auth message
		// wait for response, then call complete
		if recNonce, remoteRandomPubKey, _, err = completeHandshake(response, prvKey); err != nil {
			return
		}
		clogger.Debugf("receiver-nonce: %v", hexkey(recNonce))
		remoteRandomPubKeyS, _ := ExportPublicKey(remoteRandomPubKey)
		clogger.Debugf("receiver-random-public-key: %v", hexkey(remoteRandomPubKeyS))

	} else {
		auth = make([]byte, iHSLen)
		clogger.Debugf("waiting for initiator handshake (from %v)", hexkey(remotePubKeyS))
		if read, err = conn.Read(auth); err != nil {
			return
		}
		if read != iHSLen {
			err = fmt.Errorf("remote initiator's handshake has invalid length. expect %v, got %v", iHSLen, read)
			return
		}
		clogger.Debugf("received initiator handshake (from %v):\n%v", hexkey(remotePubKeyS), hexkey(auth))
		// we are listening connection. we are responders in the handshake.
		// Extract info from the authentication. The initiator starts by sending us a handshake that we need to respond to.
		// so we read auth message first, then respond
		var response []byte
		if response, recNonce, initNonce, randomPrivKey, remoteRandomPubKey, err = respondToHandshake(auth, prvKey, remotePubKeyS, sessionToken); err != nil {
			return
		}
		clogger.Debugf("receiver-nonce: %v", hexkey(recNonce))
		clogger.Debugf("receiver-random-priv-key: %v", hexkey(crypto.FromECDSA(randomPrivKey)))
		if _, err = conn.Write(response); err != nil {
			return
		}
		clogger.Debugf("receiver handshake (sent to %v):\n%v", hexkey(remotePubKeyS), hexkey(response))
	}
	return newSession(initiator, initNonce, recNonce, auth, randomPrivKey, remoteRandomPubKey)
}

/*
ImportPublicKey creates a 512 bit *ecsda.PublicKey from a byte slice. It accepts the simple 64 byte uncompressed format or the 65 byte format given by calling elliptic.Marshal on the EC point represented by the key. Any other length will result in an invalid public key error.
*/
func ImportPublicKey(pubKey []byte) (pubKeyEC *ecdsa.PublicKey, err error) {
	var pubKey65 []byte
	switch len(pubKey) {
	case 64:
		pubKey65 = append([]byte{0x04}, pubKey...)
	case 65:
		pubKey65 = pubKey
	default:
		return nil, fmt.Errorf("invalid public key length %v (expect 64/65)", len(pubKey))
	}
	return crypto.ToECDSAPub(pubKey65), nil
}

/*
ExportPublicKey exports a *ecdsa.PublicKey into a byte slice using a simple 64-byte format. and is used for simple serialisation in network communication
*/
func ExportPublicKey(pubKeyEC *ecdsa.PublicKey) (pubKey []byte, err error) {
	if pubKeyEC == nil {
		return nil, fmt.Errorf("no ECDSA public key given")
	}
	return crypto.FromECDSAPub(pubKeyEC)[1:], nil
}

/* startHandshake is called by if the node is the initiator of the connection.

The caller provides the public key of the peer as conjuctured from lookup based on IP:port, given as user input or proven by signatures. The caller must have access to persistant information about the peers, and pass the previous session token as an argument to cryptoId.

The first return value is the auth message that is to be sent out to the remote receiver.
*/
func startHandshake(prvKey *ecdsa.PrivateKey, remotePubKeyS, sessionToken []byte) (auth []byte, initNonce []byte, randomPrvKey *ecdsa.PrivateKey, remotePubKey *ecdsa.PublicKey, err error) {
	// session init, common to both parties
	if remotePubKey, err = ImportPublicKey(remotePubKeyS); err != nil {
		return
	}

	var tokenFlag byte // = 0x00
	if sessionToken == nil {
		// no session token found means we need to generate shared secret.
		// ecies shared secret is used as initial session token for new peers
		// generate shared key from prv and remote pubkey
		if sessionToken, err = ecies.ImportECDSA(prvKey).GenerateShared(ecies.ImportECDSAPublic(remotePubKey), sskLen, sskLen); err != nil {
			return
		}
		// tokenFlag = 0x00 // redundant
	} else {
		// for known peers, we use stored token from the previous session
		tokenFlag = 0x01
	}

	//E(remote-pubk, S(ecdhe-random, ecdh-shared-secret^nonce) || H(ecdhe-random-pubk) || pubk || nonce || 0x0)
	// E(remote-pubk, S(ecdhe-random, token^nonce) || H(ecdhe-random-pubk) || pubk || nonce || 0x1)
	// allocate msgLen long message,
	var msg []byte = make([]byte, msgLen)
	initNonce = msg[msgLen-shaLen-1 : msgLen-1]
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

	// message
	// signed-shared-secret || H(ecdhe-random-pubk) || pubk || nonce || 0x0
	copy(msg, signature) // copy signed-shared-secret
	// H(ecdhe-random-pubk)
	var randomPubKey64 []byte
	if randomPubKey64, err = ExportPublicKey(&randomPrvKey.PublicKey); err != nil {
		return
	}
	var pubKey64 []byte
	if pubKey64, err = ExportPublicKey(&prvKey.PublicKey); err != nil {
		return
	}
	copy(msg[sigLen:sigLen+shaLen], crypto.Sha3(randomPubKey64))
	// pubkey copied to the correct segment.
	copy(msg[sigLen+shaLen:sigLen+shaLen+pubLen], pubKey64)
	// nonce is already in the slice
	// stick tokenFlag byte to the end
	msg[msgLen-1] = tokenFlag

	// encrypt using remote-pubk
	// auth = eciesEncrypt(remote-pubk, msg)

	if auth, err = crypto.Encrypt(remotePubKey, msg); err != nil {
		return
	}

	return
}

/*
respondToHandshake is called by peer if it accepted (but not initiated) the connection from the remote. It is passed the initiator handshake received, the public key and session token belonging to the remote initiator.

The first return value is the authentication response (aka receiver handshake) that is to be sent to the remote initiator.
*/
func respondToHandshake(auth []byte, prvKey *ecdsa.PrivateKey, remotePubKeyS, sessionToken []byte) (authResp []byte, respNonce []byte, initNonce []byte, randomPrivKey *ecdsa.PrivateKey, remoteRandomPubKey *ecdsa.PublicKey, err error) {
	var msg []byte
	var remotePubKey *ecdsa.PublicKey
	if remotePubKey, err = ImportPublicKey(remotePubKeyS); err != nil {
		return
	}

	// they prove that msg is meant for me,
	// I prove I possess private key if i can read it
	if msg, err = crypto.Decrypt(prvKey, auth); err != nil {
		return
	}

	var tokenFlag byte
	if sessionToken == nil {
		// no session token found means we need to generate shared secret.
		// ecies shared secret is used as initial session token for new peers
		// generate shared key from prv and remote pubkey
		if sessionToken, err = ecies.ImportECDSA(prvKey).GenerateShared(ecies.ImportECDSAPublic(remotePubKey), sskLen, sskLen); err != nil {
			return
		}
		// tokenFlag = 0x00 // redundant
	} else {
		// for known peers, we use stored token from the previous session
		tokenFlag = 0x01
	}

	// the initiator nonce is read off the end of the message
	initNonce = msg[msgLen-shaLen-1 : msgLen-1]
	// I prove that i own prv key (to derive shared secret, and read nonce off encrypted msg) and that I own shared secret
	// they prove they own the private key belonging to ecdhe-random-pubk
	// we can now reconstruct the signed message and recover the peers pubkey
	var signedMsg = Xor(sessionToken, initNonce)
	var remoteRandomPubKeyS []byte
	if remoteRandomPubKeyS, err = secp256k1.RecoverPubkey(signedMsg, msg[:sigLen]); err != nil {
		return
	}
	// convert to ECDSA standard
	if remoteRandomPubKey, err = ImportPublicKey(remoteRandomPubKeyS); err != nil {
		return
	}

	// now we find ourselves a long task too, fill it random
	var resp = make([]byte, resLen)
	// generate shaLen long nonce
	respNonce = resp[pubLen : pubLen+shaLen]
	if _, err = rand.Read(respNonce); err != nil {
		return
	}
	// generate random keypair for session
	if randomPrivKey, err = crypto.GenerateKey(); err != nil {
		return
	}
	// responder auth message
	// E(remote-pubk, ecdhe-random-pubk || nonce || 0x0)
	var randomPubKeyS []byte
	if randomPubKeyS, err = ExportPublicKey(&randomPrivKey.PublicKey); err != nil {
		return
	}
	copy(resp[:pubLen], randomPubKeyS)
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

/*
completeHandshake is called when the initiator receives an authentication response (aka receiver handshake). It completes the handshake by reading off parameters the remote peer provides needed to set up the secure session
*/
func completeHandshake(auth []byte, prvKey *ecdsa.PrivateKey) (respNonce []byte, remoteRandomPubKey *ecdsa.PublicKey, tokenFlag bool, err error) {
	var msg []byte
	// they prove that msg is meant for me,
	// I prove I possess private key if i can read it
	if msg, err = crypto.Decrypt(prvKey, auth); err != nil {
		return
	}

	respNonce = msg[pubLen : pubLen+shaLen]
	var remoteRandomPubKeyS = msg[:pubLen]
	if remoteRandomPubKey, err = ImportPublicKey(remoteRandomPubKeyS); err != nil {
		return
	}
	if msg[resLen-1] == 0x01 {
		tokenFlag = true
	}
	return
}

/*
newSession is called after the handshake is completed. The arguments are values negotiated in the handshake and the return value is a new session : a new session Token to be remembered for the next time we connect with this peer. And a MsgReadWriter that implements an encrypted and authenticated connection with key material obtained from the crypto handshake key exchange
*/
func newSession(initiator bool, initNonce, respNonce, auth []byte, privKey *ecdsa.PrivateKey, remoteRandomPubKey *ecdsa.PublicKey) (sessionToken []byte, rw *secretRW, err error) {
	// 3) Now we can trust ecdhe-random-pubk to derive new keys
	//ecdhe-shared-secret = ecdh.agree(ecdhe-random, remote-ecdhe-random-pubk)
	var dhSharedSecret []byte
	pubKey := ecies.ImportECDSAPublic(remoteRandomPubKey)
	if dhSharedSecret, err = ecies.ImportECDSA(privKey).GenerateShared(pubKey, sskLen, sskLen); err != nil {
		return
	}
	var sharedSecret = crypto.Sha3(append(dhSharedSecret, crypto.Sha3(append(respNonce, initNonce...))...))
	sessionToken = crypto.Sha3(sharedSecret)
	var aesSecret = crypto.Sha3(append(dhSharedSecret, sharedSecret...))
	var macSecret = crypto.Sha3(append(dhSharedSecret, aesSecret...))
	var egressMac, ingressMac []byte
	if initiator {
		egressMac = Xor(macSecret, respNonce)
		ingressMac = Xor(macSecret, initNonce)
	} else {
		egressMac = Xor(macSecret, initNonce)
		ingressMac = Xor(macSecret, respNonce)
	}
	rw = &secretRW{
		aesSecret:  aesSecret,
		macSecret:  macSecret,
		egressMac:  egressMac,
		ingressMac: ingressMac,
	}
	clogger.Debugf("aes-secret: %v", hexkey(aesSecret))
	clogger.Debugf("mac-secret: %v", hexkey(macSecret))
	clogger.Debugf("egress-mac: %v", hexkey(egressMac))
	clogger.Debugf("ingress-mac: %v", hexkey(ingressMac))
	return
}

// TODO: optimisation
// should use cipher.xorBytes from crypto/cipher/xor.go for fast xor
func Xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return
}
