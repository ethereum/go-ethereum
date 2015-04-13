package p2p

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"errors"
	"fmt"
	"hash"
	"io"
	"net"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
	"github.com/ethereum/go-ethereum/crypto/secp256k1"
	"github.com/ethereum/go-ethereum/crypto/sha3"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	sskLen = 16 // ecies.MaxSharedKeyLength(pubKey) / 2
	sigLen = 65 // elliptic S256
	pubLen = 64 // 512 bit pubkey in uncompressed representation without format byte
	shaLen = 32 // hash length (for nonce etc)

	authMsgLen  = sigLen + shaLen + pubLen + shaLen + 1
	authRespLen = pubLen + shaLen + 1

	eciesBytes     = 65 + 16 + 32
	encAuthMsgLen  = authMsgLen + eciesBytes  // size of the final ECIES payload sent as initiator's handshake
	encAuthRespLen = authRespLen + eciesBytes // size of the final ECIES payload sent as receiver's handshake
)

// conn represents a remote connection after encryption handshake
// and protocol handshake have completed.
//
// The MsgReadWriter is usually layered as follows:
//
//     netWrapper       (I/O timeouts, thread-safe ReadMsg, WriteMsg)
//     rlpxFrameRW      (message encoding, encryption, authentication)
//     bufio.ReadWriter (buffering)
//     net.Conn         (network I/O)
//
type conn struct {
	MsgReadWriter
	*protoHandshake
}

// secrets represents the connection secrets
// which are negotiated during the encryption handshake.
type secrets struct {
	RemoteID              discover.NodeID
	AES, MAC              []byte
	EgressMAC, IngressMAC hash.Hash
	Token                 []byte
}

// protoHandshake is the RLP structure of the protocol handshake.
type protoHandshake struct {
	Version    uint64
	Name       string
	Caps       []Cap
	ListenPort uint64
	ID         discover.NodeID
}

// setupConn starts a protocol session on the given connection.
// It runs the encryption handshake and the protocol handshake.
// If dial is non-nil, the connection the local node is the initiator.
// If atcap is true, the connection will be disconnected with DiscTooManyPeers
// after the key exchange.
func setupConn(fd net.Conn, prv *ecdsa.PrivateKey, our *protoHandshake, dial *discover.Node, atcap bool) (*conn, error) {
	if dial == nil {
		return setupInboundConn(fd, prv, our, atcap)
	} else {
		return setupOutboundConn(fd, prv, our, dial, atcap)
	}
}

func setupInboundConn(fd net.Conn, prv *ecdsa.PrivateKey, our *protoHandshake, atcap bool) (*conn, error) {
	secrets, err := receiverEncHandshake(fd, prv, nil)
	if err != nil {
		return nil, fmt.Errorf("encryption handshake failed: %v", err)
	}
	rw := newRlpxFrameRW(fd, secrets)
	if atcap {
		SendItems(rw, discMsg, DiscTooManyPeers)
		return nil, errors.New("we have too many peers")
	}
	// Run the protocol handshake using authenticated messages.
	rhs, err := readProtocolHandshake(rw, secrets.RemoteID, our)
	if err != nil {
		return nil, err
	}
	if err := Send(rw, handshakeMsg, our); err != nil {
		return nil, fmt.Errorf("protocol handshake write error: %v", err)
	}
	return &conn{rw, rhs}, nil
}

func setupOutboundConn(fd net.Conn, prv *ecdsa.PrivateKey, our *protoHandshake, dial *discover.Node, atcap bool) (*conn, error) {
	secrets, err := initiatorEncHandshake(fd, prv, dial.ID, nil)
	if err != nil {
		return nil, fmt.Errorf("encryption handshake failed: %v", err)
	}
	rw := newRlpxFrameRW(fd, secrets)
	if atcap {
		SendItems(rw, discMsg, DiscTooManyPeers)
		return nil, errors.New("we have too many peers")
	}
	// Run the protocol handshake using authenticated messages.
	//
	// Note that even though writing the handshake is first, we prefer
	// returning the handshake read error. If the remote side
	// disconnects us early with a valid reason, we should return it
	// as the error so it can be tracked elsewhere.
	werr := make(chan error, 1)
	go func() { werr <- Send(rw, handshakeMsg, our) }()
	rhs, err := readProtocolHandshake(rw, secrets.RemoteID, our)
	if err != nil {
		return nil, err
	}
	if err := <-werr; err != nil {
		return nil, fmt.Errorf("protocol handshake write error: %v", err)
	}
	if rhs.ID != dial.ID {
		return nil, errors.New("dialed node id mismatch")
	}
	return &conn{rw, rhs}, nil
}

// encHandshake contains the state of the encryption handshake.
type encHandshake struct {
	initiator bool
	remoteID  discover.NodeID

	remotePub            *ecies.PublicKey  // remote-pubk
	initNonce, respNonce []byte            // nonce
	randomPrivKey        *ecies.PrivateKey // ecdhe-random
	remoteRandomPub      *ecies.PublicKey  // ecdhe-random-pubk
}

// secrets is called after the handshake is completed.
// It extracts the connection secrets from the handshake values.
func (h *encHandshake) secrets(auth, authResp []byte) (secrets, error) {
	ecdheSecret, err := h.randomPrivKey.GenerateShared(h.remoteRandomPub, sskLen, sskLen)
	if err != nil {
		return secrets{}, err
	}

	// derive base secrets from ephemeral key agreement
	sharedSecret := crypto.Sha3(ecdheSecret, crypto.Sha3(h.respNonce, h.initNonce))
	aesSecret := crypto.Sha3(ecdheSecret, sharedSecret)
	s := secrets{
		RemoteID: h.remoteID,
		AES:      aesSecret,
		MAC:      crypto.Sha3(ecdheSecret, aesSecret),
		Token:    crypto.Sha3(sharedSecret),
	}

	// setup sha3 instances for the MACs
	mac1 := sha3.NewKeccak256()
	mac1.Write(xor(s.MAC, h.respNonce))
	mac1.Write(auth)
	mac2 := sha3.NewKeccak256()
	mac2.Write(xor(s.MAC, h.initNonce))
	mac2.Write(authResp)
	if h.initiator {
		s.EgressMAC, s.IngressMAC = mac1, mac2
	} else {
		s.EgressMAC, s.IngressMAC = mac2, mac1
	}

	return s, nil
}

func (h *encHandshake) ecdhShared(prv *ecdsa.PrivateKey) ([]byte, error) {
	return ecies.ImportECDSA(prv).GenerateShared(h.remotePub, sskLen, sskLen)
}

// initiatorEncHandshake negotiates a session token on conn.
// it should be called on the dialing side of the connection.
//
// prv is the local client's private key.
// token is the token from a previous session with this node.
func initiatorEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, remoteID discover.NodeID, token []byte) (s secrets, err error) {
	h, err := newInitiatorHandshake(remoteID)
	if err != nil {
		return s, err
	}
	auth, err := h.authMsg(prv, token)
	if err != nil {
		return s, err
	}
	if _, err = conn.Write(auth); err != nil {
		return s, err
	}

	response := make([]byte, encAuthRespLen)
	if _, err = io.ReadFull(conn, response); err != nil {
		return s, err
	}
	if err := h.decodeAuthResp(response, prv); err != nil {
		return s, err
	}
	return h.secrets(auth, response)
}

func newInitiatorHandshake(remoteID discover.NodeID) (*encHandshake, error) {
	// generate random initiator nonce
	n := make([]byte, shaLen)
	if _, err := rand.Read(n); err != nil {
		return nil, err
	}
	// generate random keypair to use for signing
	randpriv, err := ecies.GenerateKey(rand.Reader, crypto.S256(), nil)
	if err != nil {
		return nil, err
	}
	rpub, err := remoteID.Pubkey()
	if err != nil {
		return nil, fmt.Errorf("bad remoteID: %v", err)
	}
	h := &encHandshake{
		initiator:     true,
		remoteID:      remoteID,
		remotePub:     ecies.ImportECDSAPublic(rpub),
		initNonce:     n,
		randomPrivKey: randpriv,
	}
	return h, nil
}

// authMsg creates an encrypted initiator handshake message.
func (h *encHandshake) authMsg(prv *ecdsa.PrivateKey, token []byte) ([]byte, error) {
	var tokenFlag byte
	if token == nil {
		// no session token found means we need to generate shared secret.
		// ecies shared secret is used as initial session token for new peers
		// generate shared key from prv and remote pubkey
		var err error
		if token, err = h.ecdhShared(prv); err != nil {
			return nil, err
		}
	} else {
		// for known peers, we use stored token from the previous session
		tokenFlag = 0x01
	}

	// sign known message:
	//   ecdh-shared-secret^nonce for new peers
	//   token^nonce for old peers
	signed := xor(token, h.initNonce)
	signature, err := crypto.Sign(signed, h.randomPrivKey.ExportECDSA())
	if err != nil {
		return nil, err
	}

	// encode auth message
	// signature || sha3(ecdhe-random-pubk) || pubk || nonce || token-flag
	msg := make([]byte, authMsgLen)
	n := copy(msg, signature)
	n += copy(msg[n:], crypto.Sha3(exportPubkey(&h.randomPrivKey.PublicKey)))
	n += copy(msg[n:], crypto.FromECDSAPub(&prv.PublicKey)[1:])
	n += copy(msg[n:], h.initNonce)
	msg[n] = tokenFlag

	// encrypt auth message using remote-pubk
	return ecies.Encrypt(rand.Reader, h.remotePub, msg, nil, nil)
}

// decodeAuthResp decode an encrypted authentication response message.
func (h *encHandshake) decodeAuthResp(auth []byte, prv *ecdsa.PrivateKey) error {
	msg, err := crypto.Decrypt(prv, auth)
	if err != nil {
		return fmt.Errorf("could not decrypt auth response (%v)", err)
	}
	h.respNonce = msg[pubLen : pubLen+shaLen]
	h.remoteRandomPub, err = importPublicKey(msg[:pubLen])
	if err != nil {
		return err
	}
	// ignore token flag for now
	return nil
}

// receiverEncHandshake negotiates a session token on conn.
// it should be called on the listening side of the connection.
//
// prv is the local client's private key.
// token is the token from a previous session with this node.
func receiverEncHandshake(conn io.ReadWriter, prv *ecdsa.PrivateKey, token []byte) (s secrets, err error) {
	// read remote auth sent by initiator.
	auth := make([]byte, encAuthMsgLen)
	if _, err := io.ReadFull(conn, auth); err != nil {
		return s, err
	}
	h, err := decodeAuthMsg(prv, token, auth)
	if err != nil {
		return s, err
	}

	// send auth response
	resp, err := h.authResp(prv, token)
	if err != nil {
		return s, err
	}
	if _, err = conn.Write(resp); err != nil {
		return s, err
	}

	return h.secrets(auth, resp)
}

func decodeAuthMsg(prv *ecdsa.PrivateKey, token []byte, auth []byte) (*encHandshake, error) {
	var err error
	h := new(encHandshake)
	// generate random keypair for session
	h.randomPrivKey, err = ecies.GenerateKey(rand.Reader, crypto.S256(), nil)
	if err != nil {
		return nil, err
	}
	// generate random nonce
	h.respNonce = make([]byte, shaLen)
	if _, err = rand.Read(h.respNonce); err != nil {
		return nil, err
	}

	msg, err := crypto.Decrypt(prv, auth)
	if err != nil {
		return nil, fmt.Errorf("could not decrypt auth message (%v)", err)
	}

	// decode message parameters
	// signature || sha3(ecdhe-random-pubk) || pubk || nonce || token-flag
	h.initNonce = msg[authMsgLen-shaLen-1 : authMsgLen-1]
	copy(h.remoteID[:], msg[sigLen+shaLen:sigLen+shaLen+pubLen])
	rpub, err := h.remoteID.Pubkey()
	if err != nil {
		return nil, fmt.Errorf("bad remoteID: %#v", err)
	}
	h.remotePub = ecies.ImportECDSAPublic(rpub)

	// recover remote random pubkey from signed message.
	if token == nil {
		// TODO: it is an error if the initiator has a token and we don't. check that.

		// no session token means we need to generate shared secret.
		// ecies shared secret is used as initial session token for new peers.
		// generate shared key from prv and remote pubkey.
		if token, err = h.ecdhShared(prv); err != nil {
			return nil, err
		}
	}
	signedMsg := xor(token, h.initNonce)
	remoteRandomPub, err := secp256k1.RecoverPubkey(signedMsg, msg[:sigLen])
	if err != nil {
		return nil, err
	}
	h.remoteRandomPub, _ = importPublicKey(remoteRandomPub)
	return h, nil
}

// authResp generates the encrypted authentication response message.
func (h *encHandshake) authResp(prv *ecdsa.PrivateKey, token []byte) ([]byte, error) {
	// responder auth message
	// E(remote-pubk, ecdhe-random-pubk || nonce || 0x0)
	resp := make([]byte, authRespLen)
	n := copy(resp, exportPubkey(&h.randomPrivKey.PublicKey))
	n += copy(resp[n:], h.respNonce)
	if token == nil {
		resp[n] = 0
	} else {
		resp[n] = 1
	}
	// encrypt using remote-pubk
	return ecies.Encrypt(rand.Reader, h.remotePub, resp, nil, nil)
}

// importPublicKey unmarshals 512 bit public keys.
func importPublicKey(pubKey []byte) (*ecies.PublicKey, error) {
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
	// TODO: fewer pointless conversions
	return ecies.ImportECDSAPublic(crypto.ToECDSAPub(pubKey65)), nil
}

func exportPubkey(pub *ecies.PublicKey) []byte {
	if pub == nil {
		panic("nil pubkey")
	}
	return elliptic.Marshal(pub.Curve, pub.X, pub.Y)[1:]
}

func xor(one, other []byte) (xor []byte) {
	xor = make([]byte, len(one))
	for i := 0; i < len(one); i++ {
		xor[i] = one[i] ^ other[i]
	}
	return xor
}

func readProtocolHandshake(rw MsgReadWriter, wantID discover.NodeID, our *protoHandshake) (*protoHandshake, error) {
	msg, err := rw.ReadMsg()
	if err != nil {
		return nil, err
	}
	if msg.Code == discMsg {
		// disconnect before protocol handshake is valid according to the
		// spec and we send it ourself if Server.addPeer fails.
		var reason [1]DiscReason
		rlp.Decode(msg.Payload, &reason)
		return nil, reason[0]
	}
	if msg.Code != handshakeMsg {
		return nil, fmt.Errorf("expected handshake, got %x", msg.Code)
	}
	if msg.Size > baseProtocolMaxMsgSize {
		return nil, fmt.Errorf("message too big (%d > %d)", msg.Size, baseProtocolMaxMsgSize)
	}
	var hs protoHandshake
	if err := msg.Decode(&hs); err != nil {
		return nil, err
	}
	// validate handshake info
	if hs.Version != our.Version {
		SendItems(rw, discMsg, DiscIncompatibleVersion)
		return nil, fmt.Errorf("required version %d, received %d\n", baseProtocolVersion, hs.Version)
	}
	if (hs.ID == discover.NodeID{}) {
		SendItems(rw, discMsg, DiscInvalidIdentity)
		return nil, errors.New("invalid public key in handshake")
	}
	if hs.ID != wantID {
		SendItems(rw, discMsg, DiscUnexpectedIdentity)
		return nil, errors.New("handshake node ID does not match encryption handshake")
	}
	return &hs, nil
}
