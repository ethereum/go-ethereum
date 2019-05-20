package p2p

import (
	// "binary"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	ethlogger "github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/obscuren/ecies"
)

var clogger = ethlogger.NewLogger("CRYPTOID")

const (
	sskLen = 16 // ecies.MaxSharedKeyLength(pubKey) / 2
	sigLen = 65 // elliptic S256
	pubLen = 64 // 512 bit pubkey in uncompressed representation without format byte
	shaLen = 32 // hash length (for nonce etc)

	authMsgLen  = sigLen + shaLen + pubLen + shaLen + 1
	authRespLen = pubLen + shaLen + 1

	eciesBytes = 65 + 16 + 32
	iHSLen     = authMsgLen + eciesBytes  // size of the final ECIES payload sent as initiator's handshake
	rHSLen     = authRespLen + eciesBytes // size of the final ECIES payload sent as receiver's handshake
)

type hexkey []byte

func (self hexkey) String() string {
	return fmt.Sprintf("(%d) %x", len(self), []byte(self))
}

func encHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, dial *discover.Node) (
	remoteID discover.NodeID,
	sessionToken []byte,
	err error,
) {
	if dial == nil {
		var remotePubkey []byte
		sessionToken, remotePubkey, err = inboundEncHandshake(conn, prv, nil)
		copy(remoteID[:], remotePubkey)
	} else {
		remoteID = dial.ID
		sessionToken, err = outboundEncHandshake(conn, prv, remoteID[:], nil)
	}
	return remoteID, sessionToken, err
}

// outboundEncHandshake negotiates a session token on conn.
// it should be called on the dialing side of the connection.
//
// privateKey is the local client's private key
// remotePublicKey is the remote peer's node ID
// sessionToken is the token from a previous session with this node.
func outboundEncHandshake(conn io.ReadWriter, prvKey *ecdsa.PrivateKey, remotePublicKey []byte, sessionToken []byte) (
	newSessionToken []byte,
	err error,
) {
	auth, initNonce, randomPrivKey, err := authMsg(prvKey, remotePublicKey, sessionToken)
	if err != nil {
		return nil, err
	}
	if sessionToken != nil {
		clogger.Debugf("session-token: %v", hexkey(sessionToken))
	}

	clogger.Debugf("initiator-nonce: %v", hexkey(initNonce))
	clogger.Debugf("initiator-random-private-key: %v", hexkey(crypto.FromECDSA(randomPrivKey)))
	randomPublicKeyS, _ := exportPublicKey(&randomPrivKey.PublicKey)
	clogger.Debugf("initiator-random-public-key: %v", hexkey(randomPublicKeyS))
	if _, err = conn.Write(auth); err != nil {
		return nil, err
	}
	clogger.Debugf("initiator handshake: %v", hexkey(auth))

	response := make([]byte, rHSLen)
	if _, err = io.ReadFull(conn, response); err != nil {
		return nil, err
	}
	recNonce, remoteRandomPubKey, _, err := completeHandshake(response, prvKey)
	if err != nil {
		return nil, err
	}

	clogger.Debugf("receiver-nonce: %v", hexkey(recNonce))
	remoteRandomPubKeyS, _ := exportPublicKey(remoteRandomPubKey)
	clogger.Debugf("receiver-random-public-key: %v", hexkey(remoteRandomPubKeyS))
	return newSession(initNonce, recNonce, randomPrivKey, remoteRandomPubKey)
}

// authMsg creates the initiator handshake.
func authMsg(prvKey *ecdsa.PrivateKey, remotePubKeyS, sessionToken []byte) (
	auth, initNonce []byte,
	randomPrvKey *ecdsa.PrivateKey,
	err error,
) {
	// session init, common to both parties
	remotePubKey, err := importPublicKey(remotePubKeyS)
	if err != nil {
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
	var msg []byte = make([]byte, authMsgLen)
	initNonce = msg[authMsgLen-shaLen-1 : authMsgLen-1]
	if _, err = rand.Read(initNonce); err != nil {
		return
	}
	// create known message
	// ecdh-shared-secret^nonce for new peers
	// token^nonce for old peers
	var sharedSecret = xor(sessionToken, initNonce)

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
	if randomPubKey64, err = exportPublicKey(&randomPrvKey.PublicKey); err != nil {
		return
	}
	var pubKey64 []byte
	if pubKey64, err = exportPublicKey(&prvKey.PublicKey); err != nil {
		return
	}
	copy(msg[sigLen:sigLen+shaLen], crypto.Sha3(randomPubKey64))
	// pubkey copied to the correct segment.
	copy(msg[sigLen+shaLen:sigLen+shaLen+pubLen], pubKey64)
	// nonce is already in the slice
	// stick tokenFlag byte to the end
	msg[authMsgLen-1] = tokenFlag

	// encrypt using remote-pubk
	// auth = eciesEncrypt(remote-pubk, msg)
	if auth, err = crypto.Encrypt(remotePubKey, msg); err != nil {
		return
	}
	return
}

// completeHandshake is called when the initiator receives an
// authentication response (aka receiver handshake). It completes the
// handshake by reading off parameters the remote peer provides needed
// to set up the secure session.
func completeHandshake(auth []byte, prvKey *ecdsa.PrivateKey) (
	respNonce []byte,
	remoteRandomPubKey *ecdsa.PublicKey,
	tokenFlag bool,
	err error,
) {
	var msg []byte
	// they prove that msg is meant for me,
	// I prove I possess private key if i can read it
	if msg, err = crypto.Decrypt(prvKey, auth); err != nil {
		return
	}

	respNonce = msg[pubLen : pubLen+shaLen]
	var remoteRandomPubKeyS = msg[:pubLen]
	if remoteRandomPubKey, err = importPublicKey(remoteRandomPubKeyS); err != nil {
		return
	}
	if msg[authRespLen-1] == 0x01 {
		tokenFlag = true
	}
	return
}

// inboundEncHandshake negotiates a session token on conn.
// it should be called on the listening side of the connection.
//
// privateKey is the local client's private key
// sessionToken is the token from a previous session with this node.
func inboundEncHandshake(conn io.ReadWriter, prvKey *ecdsa.PrivateKey, sessionToken []byte) (
	token, remotePubKey []byte,
	err error,
) {
	// we are listening connection. we are responders in the
	// handshake. Extract info from the authentication. The initiator
	// starts by sending us a handshake that we need to respond to. so
	// we read auth message first, then respond.
	auth := make([]byte, iHSLen)
	if _, err := io.ReadFull(conn, auth); err != nil {
		return nil, nil, err
	}
	response, recNonce, initNonce, remotePubKey, randomPrivKey, remoteRandomPubKey, err := authResp(auth, sessionToken, prvKey)
	if err != nil {
		return nil, nil, err
	}
	clogger.Debugf("receiver-nonce: %v", hexkey(recNonce))
	clogger.Debugf("receiver-random-priv-key: %v", hexkey(crypto.FromECDSA(randomPrivKey)))
	if _, err = conn.Write(response); err != nil {
		return nil, nil, err
	}
	clogger.Debugf("receiver handshake:\n%v", hexkey(response))
	token, err = newSession(initNonce, recNonce, randomPrivKey, remoteRandomPubKey)
	return token, remotePubKey, err
}

// authResp is called by peer if it accepted (but not
// initiated) the connection from the remote. It is passed the initiator
// handshake received and the session token belonging to the
// remote initiator.
//
// The first return value is the authentication response (aka receiver
// handshake) that is to be sent to the remote initiator.
func authResp(auth, sessionToken []byte, prvKey *ecdsa.PrivateKey) (
	authResp, respNonce, initNonce, remotePubKeyS []byte,
	randomPrivKey *ecdsa.PrivateKey,
	remoteRandomPubKey *ecdsa.PublicKey,
	err error,
) {
	// they prove that msg is meant for me,
	// I prove I possess private key if i can read it
	msg, err := crypto.Decrypt(prvKey, auth)
	if err != nil {
		return
	}

	remotePubKeyS = msg[sigLen+shaLen : sigLen+shaLen+pubLen]
	remotePubKey, _ := importPublicKey(remotePubKeyS)

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
	initNonce = msg[authMsgLen-shaLen-1 : authMsgLen-1]
	// I prove that i own prv key (to derive shared secret, and read
	// nonce off encrypted msg) and that I own shared secret they
	// prove they own the private key belonging to ecdhe-random-pubk
	// we can now reconstruct the signed message and recover the peers
	// pubkey
	var signedMsg = xor(sessionToken, initNonce)
	var remoteRandomPubKeyS []byte
	if remoteRandomPubKeyS, err = secp256k1.RecoverPubkey(signedMsg, msg[:sigLen]); err != nil {
		return
	}
	// convert to ECDSA standard
	if remoteRandomPubKey, err = importPublicKey(remoteRandomPubKeyS); err != nil {
		return
	}

	// now we find ourselves a long task too, fill it random
	var resp = make([]byte, authRespLen)
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
	if randomPubKeyS, err = exportPublicKey(&randomPrivKey.PublicKey); err != nil {
		return
	}
	copy(resp[:pubLen], randomPubKeyS)
	// nonce is already in the slice
	resp[authRespLen-1] = tokenFlag

	// encrypt using remote-pubk
	// auth = eciesEncrypt(remote-pubk, msg)
	// why not encrypt with ecdhe-random-remote
	if authResp, err = crypto.Encrypt(remotePubKey, resp); err != nil {
		return
	}
	return
}

// newSession is called after the handshake is completed. The
// arguments are values negotiated in the handshake. The return value
// is a new session Token to be remembered for the next time we
// connect with this peer.
func newSession(initNonce, respNonce []byte, privKey *ecdsa.PrivateKey, remoteRandomPubKey *ecdsa.PublicKey) ([]byte, error) {
	// 3) Now we can trust ecdhe-random-pubk to derive new keys
	//ecdhe-shared-secret = ecdh.agree(ecdhe-random, remote-ecdhe-random-pubk)
	pubKey := ecies.ImportECDSAPublic(remoteRandomPubKey)
	dhSharedSecret, err := ecies.ImportECDSA(privKey).GenerateShared(pubKey, sskLen, sskLen)
	if err != nil {
		return nil, err
	}
	sharedSecret := crypto.Sha3(dhSharedSecret, crypto.Sha3(respNonce, initNonce))
	sessionToken := crypto.Sha3(sharedSecret)
	return sessionToken, nil
}

// importPublicKey unmarshals 512 bit public keys.
func importPublicKey(pubKey []byte) (pubKeyEC *ecdsa.PublicKey, err error) {
	var pubKey65 []byte
	switch len(pubKey) {
	case 64:
		// add 'uncompressed key' flag
		pubKey65 = append([]byte{0x04}, pubKey...)
	case 65:
		pubKey65 = pubKey
	default:
		return nil, fmt.Errorf("invalid public key length %v (expect 64/65)", len(pubKey))
	}
	return crypto.ToECDSAPub(pubKey65), nil
}

func exportPublicKey(pubKeyEC *ecdsa.PublicKey) (pubKey []byte, err error) {
	if pubKeyEC == nil {
		return nil, fmt.Errorf("no ECDSA public key given")
	}
	return crypto.FromECDSAPub(pubKeyEC)[1:], nil
}

func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}
