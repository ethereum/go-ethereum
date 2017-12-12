package notifications

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	whisper "github.com/ethereum/go-ethereum/whisper/whisperv5"
	"github.com/status-im/status-go/geth/params"
)

const (
	topicSendNotification      = "SEND_NOTIFICATION"
	topicNewChatSession        = "NEW_CHAT_SESSION"
	topicAckNewChatSession     = "ACK_NEW_CHAT_SESSION"
	topicNewDeviceRegistration = "NEW_DEVICE_REGISTRATION"
	topicAckDeviceRegistration = "ACK_DEVICE_REGISTRATION"
	topicCheckClientSession    = "CHECK_CLIENT_SESSION"
	topicConfirmClientSession  = "CONFIRM_CLIENT_SESSION"
	topicDropClientSession     = "DROP_CLIENT_SESSION"
)

var (
	ErrServiceInitError = errors.New("notification service has not been properly initialized")
)

// NotificationServer service capable of handling Push Notifications
type NotificationServer struct {
	whisper *whisper.Whisper
	config  *params.WhisperConfig

	nodeID      string            // proposed server will feature this ID
	discovery   *discoveryService // discovery service handles client/server negotiation, when server is selected
	protocolKey *ecdsa.PrivateKey // private key of service, used to encode handshake communication

	clientSessions   map[string]*ClientSession
	clientSessionsMu sync.RWMutex

	chatSessions   map[string]*ChatSession
	chatSessionsMu sync.RWMutex

	deviceSubscriptions   map[string]*DeviceSubscription
	deviceSubscriptionsMu sync.RWMutex

	firebaseProvider NotificationDeliveryProvider

	quit chan struct{}
}

// ClientSession abstracts notification client, which expects notifications whenever
// some envelope can be decoded with session key (key hash is compared for optimization)
type ClientSession struct {
	ClientKey      string      // public key uniquely identifying a client
	SessionKey     []byte      // actual symkey used for client - server communication
	SessionKeyHash common.Hash // The Keccak256Hash of the symmetric key, which is shared between server/client
	SessionKeyInput []byte      // raw symkey used as input for actual SessionKey
}

// ChatSession abstracts chat session, which some previously registered client can create.
// ChatSession is used by client for sharing common secret, allowing others to register
// themselves and eventually to trigger notifications.
type ChatSession struct {
	ParentKey      string      // public key uniquely identifying a client session used to create a chat session
	ChatKey        string      // ID that uniquely identifies a chat session
	SessionKey     []byte      // actual symkey used for client - server communication
	SessionKeyHash common.Hash // The Keccak256Hash of the symmetric key, which is shared between server/client
}

// DeviceSubscription stores enough information about a device (or group of devices),
// so that Notification Server can trigger notification on that device(s)
type DeviceSubscription struct {
	DeviceID           string           // ID that will be used as destination
	ChatSessionKeyHash common.Hash      // The Keccak256Hash of the symmetric key, which is shared between server/client
	PubKey             *ecdsa.PublicKey // public key of subscriber (to filter out when notification is triggered)
}

// Init used for service initialization, making sure it is safe to call Start()
func (s *NotificationServer) Init(whisperService *whisper.Whisper, whisperConfig *params.WhisperConfig) {
	s.whisper = whisperService
	s.config = whisperConfig

	s.discovery = NewDiscoveryService(s)
	s.clientSessions = make(map[string]*ClientSession)
	s.chatSessions = make(map[string]*ChatSession)
	s.deviceSubscriptions = make(map[string]*DeviceSubscription)
	s.quit = make(chan struct{})

	// setup providers (FCM only, for now)
	s.firebaseProvider = NewFirebaseProvider(whisperConfig.FirebaseConfig)
}

// Start begins notification loop, in a separate go routine
func (s *NotificationServer) Start(stack *p2p.Server) error {
	if s.whisper == nil {
		return ErrServiceInitError
	}

	// configure nodeID
	if stack != nil {
		if nodeInfo := stack.NodeInfo(); nodeInfo != nil {
			s.nodeID = nodeInfo.ID
		}
	}

	// configure keys
	identity, err := s.config.ReadIdentityFile()
	if err != nil {
		return err
	}
	s.whisper.AddKeyPair(identity)
	s.protocolKey = identity
	log.Info("protocol pubkey", "key", common.ToHex(crypto.FromECDSAPub(&s.protocolKey.PublicKey)))

	// start discovery protocol
	s.discovery.Start()

	// client session status requests
	clientSessionStatusFilterID, err := s.installKeyFilter(topicCheckClientSession, s.protocolKey)
	if err != nil {
		return fmt.Errorf("failed installing filter: %v", err)
	}
	go s.requestProcessorLoop(clientSessionStatusFilterID, topicDiscoverServer, s.processClientSessionStatusRequest)

	// client session remove requests
	dropClientSessionFilterID, err := s.installKeyFilter(topicDropClientSession, s.protocolKey)
	if err != nil {
		return fmt.Errorf("failed installing filter: %v", err)
	}
	go s.requestProcessorLoop(dropClientSessionFilterID, topicDropClientSession, s.processDropClientSessionRequest)

	log.Info("Whisper Notification Server started")
	return nil
}

// Stop handles stopping the running notification loop, and all related resources
func (s *NotificationServer) Stop() error {
	close(s.quit)

	if s.whisper == nil {
		return ErrServiceInitError
	}

	if s.discovery != nil {
		s.discovery.Stop()
	}

	log.Info("Whisper Notification Server stopped")
	return nil
}

// RegisterClientSession forms a cryptographic link between server and client.
// It does so by sharing a session SymKey and installing filter listening for messages
// encrypted with that key. So, both server and client have a secure way to communicate.
func (s *NotificationServer) RegisterClientSession(session *ClientSession) (sessionKey []byte, err error) {
	s.clientSessionsMu.Lock()
	defer s.clientSessionsMu.Unlock()

	// generate random symmetric session key
	keyName := fmt.Sprintf("%s-%s", "ntfy-client", crypto.Keccak256Hash([]byte(session.ClientKey)).Hex())
	sessionKey, sessionKeyDerived, err := s.makeSessionKey(keyName)
	if err != nil {
		return nil, err
	}

	// populate session key hash (will be used to match decrypted message to a given client id)
	session.SessionKeyInput = sessionKey
	session.SessionKeyHash = crypto.Keccak256Hash(sessionKeyDerived)
	session.SessionKey = sessionKeyDerived

	// append to list of known clients
	// so that it is trivial to go key hash -> client session info
	id := session.SessionKeyHash.Hex()
	s.clientSessions[id] = session

	// setup filter, which will get all incoming messages, that are encrypted with SymKey
	filterID, err := s.installTopicFilter(topicNewChatSession, sessionKeyDerived)
	if err != nil {
		return nil, fmt.Errorf("failed installing filter: %v", err)
	}
	go s.requestProcessorLoop(filterID, topicNewChatSession, s.processNewChatSessionRequest)
	return
}

// RegisterChatSession forms a cryptographic link between server and client.
// This link is meant to be shared with other clients, so that they can use
// the shared SymKey to trigger notifications for devices attached to a given
// chat session.
func (s *NotificationServer) RegisterChatSession(session *ChatSession) (sessionKey []byte, err error) {
	s.chatSessionsMu.Lock()
	defer s.chatSessionsMu.Unlock()

	// generate random symmetric session key
	keyName := fmt.Sprintf("%s-%s", "ntfy-chat", crypto.Keccak256Hash([]byte(session.ParentKey+session.ChatKey)).Hex())
	sessionKey, sessionKeyDerived, err := s.makeSessionKey(keyName)
	if err != nil {
		return nil, err
	}

	// populate session key hash (will be used to match decrypted message to a given client id)
	session.SessionKeyHash = crypto.Keccak256Hash(sessionKeyDerived)
	session.SessionKey = sessionKeyDerived

	// append to list of known clients
	// so that it is trivial to go key hash -> client session info
	id := session.SessionKeyHash.Hex()
	s.chatSessions[id] = session

	// setup filter, to process incoming device registration requests
	filterID1, err := s.installTopicFilter(topicNewDeviceRegistration, sessionKeyDerived)
	if err != nil {
		return nil, fmt.Errorf("failed installing filter: %v", err)
	}
	go s.requestProcessorLoop(filterID1, topicNewDeviceRegistration, s.processNewDeviceRegistrationRequest)

	// setup filter, to process incoming notification trigger requests
	filterID2, err := s.installTopicFilter(topicSendNotification, sessionKeyDerived)
	if err != nil {
		return nil, fmt.Errorf("failed installing filter: %v", err)
	}
	go s.requestProcessorLoop(filterID2, topicSendNotification, s.processSendNotificationRequest)

	return
}

// RegisterDeviceSubscription persists device id, so that it can be used to trigger notifications.
func (s *NotificationServer) RegisterDeviceSubscription(subscription *DeviceSubscription) error {
	s.deviceSubscriptionsMu.Lock()
	defer s.deviceSubscriptionsMu.Unlock()

	// if one passes the same id again, we will just overwrite
	id := fmt.Sprintf("%s-%s", "ntfy-device",
		crypto.Keccak256Hash([]byte(subscription.ChatSessionKeyHash.Hex()+subscription.DeviceID)).Hex())
	s.deviceSubscriptions[id] = subscription

	log.Info("device registered", "device", subscription.DeviceID)
	return nil
}

// DropClientSession uninstalls session
func (s *NotificationServer) DropClientSession(id string) {
	dropChatSessions := func(parentKey string) {
		s.chatSessionsMu.Lock()
		defer s.chatSessionsMu.Unlock()

		for key, chatSession := range s.chatSessions {
			if chatSession.ParentKey == parentKey {
				delete(s.chatSessions, key)
				log.Info("drop chat session", "key", key)
			}
		}
	}

	dropDeviceSubscriptions := func(parentKey string) {
		s.deviceSubscriptionsMu.Lock()
		defer s.deviceSubscriptionsMu.Unlock()

		for key, subscription := range s.deviceSubscriptions {
			if hex.EncodeToString(crypto.FromECDSAPub(subscription.PubKey)) == parentKey {
				delete(s.deviceSubscriptions, key)
				log.Info("drop device subscription", "key", key)
			}
		}
	}

	s.clientSessionsMu.Lock()
	if session, ok := s.clientSessions[id]; ok {
		delete(s.clientSessions, id)
		log.Info("server drops client session", "id", id)
		s.clientSessionsMu.Unlock()

		dropDeviceSubscriptions(session.ClientKey)
		dropChatSessions(session.ClientKey)
	}
}

// processNewChatSessionRequest processes incoming client requests of type:
// client has a session key, and ready to create a new chat session (which is
// a bag of subscribed devices, basically)
func (s *NotificationServer) processNewChatSessionRequest(msg *whisper.ReceivedMessage) error {
	s.clientSessionsMu.RLock()
	defer s.clientSessionsMu.RUnlock()

	var parsedMessage struct {
		ChatID string `json:"chat"`
	}
	if err := json.Unmarshal(msg.Payload, &parsedMessage); err != nil {
		return err
	}

	if msg.Src == nil {
		return errors.New("message 'from' field is required")
	}

	clientSession, ok := s.clientSessions[msg.SymKeyHash.Hex()]
	if !ok {
		return errors.New("client session not found")
	}

	// register chat session
	parentKey := hex.EncodeToString(crypto.FromECDSAPub(msg.Src))
	sessionKey, err := s.RegisterChatSession(&ChatSession{
		ParentKey: parentKey,
		ChatKey:   parsedMessage.ChatID,
	})
	if err != nil {
		return err
	}

	// confirm that chat has been successfully created
	msgParams := whisper.MessageParams{
		Dst:      msg.Src,
		KeySym:   clientSession.SessionKey,
		Topic:    MakeTopic([]byte(topicAckNewChatSession)),
		Payload:  []byte(`{"server": "0x` + s.nodeID + `", "key": "0x` + hex.EncodeToString(sessionKey) + `"}`),
		TTL:      uint32(s.config.TTL),
		PoW:      s.config.MinimumPoW,
		WorkTime: 5,
	}
	response, err := whisper.NewSentMessage(&msgParams)
	if err != nil {
		return fmt.Errorf("failed to create server response message: %v", err)
	}
	env, err := response.Wrap(&msgParams)
	if err != nil {
		return fmt.Errorf("failed to wrap server response message: %v", err)
	}

	if err := s.whisper.Send(env); err != nil {
		return fmt.Errorf("failed to send server response message: %v", err)
	}

	log.Info("server confirms chat creation", "dst",
		common.ToHex(crypto.FromECDSAPub(msgParams.Dst)), "topic", msgParams.Topic.String())
	return nil
}

// processNewDeviceRegistrationRequest processes incoming client requests of type:
// client has a session key, creates chat, and obtains chat SymKey (to be shared with
// others). Then using that chat SymKey client registers it's device ID with server.
func (s *NotificationServer) processNewDeviceRegistrationRequest(msg *whisper.ReceivedMessage) error {
	s.chatSessionsMu.RLock()
	defer s.chatSessionsMu.RUnlock()

	var parsedMessage struct {
		DeviceID string `json:"device"`
	}
	if err := json.Unmarshal(msg.Payload, &parsedMessage); err != nil {
		return err
	}

	if msg.Src == nil {
		return errors.New("message 'from' field is required")
	}

	chatSession, ok := s.chatSessions[msg.SymKeyHash.Hex()]
	if !ok {
		return errors.New("chat session not found")
	}

	if len(parsedMessage.DeviceID) <= 0 {
		return errors.New("'device' cannot be empty")
	}

	// register chat session
	err := s.RegisterDeviceSubscription(&DeviceSubscription{
		DeviceID:           parsedMessage.DeviceID,
		ChatSessionKeyHash: chatSession.SessionKeyHash,
		PubKey:             msg.Src,
	})
	if err != nil {
		return err
	}

	// confirm that client has been successfully subscribed
	msgParams := whisper.MessageParams{
		Dst:      msg.Src,
		KeySym:   chatSession.SessionKey,
		Topic:    MakeTopic([]byte(topicAckDeviceRegistration)),
		Payload:  []byte(`{"server": "0x` + s.nodeID + `"}`),
		TTL:      uint32(s.config.TTL),
		PoW:      s.config.MinimumPoW,
		WorkTime: 5,
	}
	response, err := whisper.NewSentMessage(&msgParams)
	if err != nil {
		return fmt.Errorf("failed to create server response message: %v", err)
	}
	env, err := response.Wrap(&msgParams)
	if err != nil {
		return fmt.Errorf("failed to wrap server response message: %v", err)
	}

	if err := s.whisper.Send(env); err != nil {
		return fmt.Errorf("failed to send server response message: %v", err)
	}

	log.Info("server confirms device registration", "dst",
		common.ToHex(crypto.FromECDSAPub(msgParams.Dst)), "topic", msgParams.Topic.String())
	return nil
}

// processSendNotificationRequest processes incoming client requests of type:
// when client has session key, and ready to use it to send notifications
func (s *NotificationServer) processSendNotificationRequest(msg *whisper.ReceivedMessage) error {
	s.deviceSubscriptionsMu.RLock()
	defer s.deviceSubscriptionsMu.RUnlock()

	for _, subscriber := range s.deviceSubscriptions {
		if subscriber.ChatSessionKeyHash == msg.SymKeyHash {
			if whisper.IsPubKeyEqual(msg.Src, subscriber.PubKey) {
				continue // no need to notify ourselves
			}

			if s.firebaseProvider != nil {
				err := s.firebaseProvider.Send(subscriber.DeviceID, string(msg.Payload))
				if err != nil {
					log.Info("cannot send notification", "error", err)
				}
			}
		}
	}

	return nil
}

// processClientSessionStatusRequest processes incoming client requests when:
// client wants to learn whether it is already registered on some of the servers
func (s *NotificationServer) processClientSessionStatusRequest(msg *whisper.ReceivedMessage) error {
	s.clientSessionsMu.RLock()
	defer s.clientSessionsMu.RUnlock()

	if msg.Src == nil {
		return errors.New("message 'from' field is required")
	}

	var sessionKey []byte
	pubKey := hex.EncodeToString(crypto.FromECDSAPub(msg.Src))
	for _, clientSession := range s.clientSessions {
		if clientSession.ClientKey == pubKey {
			sessionKey = clientSession.SessionKeyInput
			break
		}
	}

	// session is not found
	if sessionKey == nil {
		return nil
	}

	// let client know that we have session for a given public key
	msgParams := whisper.MessageParams{
		Src:      s.protocolKey,
		Dst:      msg.Src,
		Topic:    MakeTopic([]byte(topicConfirmClientSession)),
		Payload:  []byte(`{"server": "0x` + s.nodeID + `", "key": "0x` + hex.EncodeToString(sessionKey) + `"}`),
		TTL:      uint32(s.config.TTL),
		PoW:      s.config.MinimumPoW,
		WorkTime: 5,
	}
	response, err := whisper.NewSentMessage(&msgParams)
	if err != nil {
		return fmt.Errorf("failed to create server response message: %v", err)
	}
	env, err := response.Wrap(&msgParams)
	if err != nil {
		return fmt.Errorf("failed to wrap server response message: %v", err)
	}

	if err := s.whisper.Send(env); err != nil {
		return fmt.Errorf("failed to send server response message: %v", err)
	}

	log.Info("server confirms client session", "dst",
		common.ToHex(crypto.FromECDSAPub(msgParams.Dst)), "topic", msgParams.Topic.String())
	return nil
}

// processDropClientSessionRequest processes incoming client requests when:
// client wants to drop its sessions with notification servers (if they exist)
func (s *NotificationServer) processDropClientSessionRequest(msg *whisper.ReceivedMessage) error {
	if msg.Src == nil {
		return errors.New("message 'from' field is required")
	}

	s.clientSessionsMu.RLock()
	pubKey := hex.EncodeToString(crypto.FromECDSAPub(msg.Src))
	for _, clientSession := range s.clientSessions {
		if clientSession.ClientKey == pubKey {
			s.clientSessionsMu.RUnlock()
			s.DropClientSession(clientSession.SessionKeyHash.Hex())
			break
		}
	}
	return nil
}

// installTopicFilter installs Whisper filter using symmetric key
func (s *NotificationServer) installTopicFilter(topicName string, topicKey []byte) (filterID string, err error) {
	topic := MakeTopicAsBytes([]byte(topicName))
	filter := whisper.Filter{
		KeySym:   topicKey,
		Topics:   [][]byte{topic},
		AllowP2P: true,
	}
	filterID, err = s.whisper.Subscribe(&filter)
	if err != nil {
		return "", fmt.Errorf("failed installing filter: %v", err)
	}

	log.Debug(fmt.Sprintf("installed topic filter %v for topic %x (%s)", filterID, topic, topicName))
	return
}

// installKeyFilter installs Whisper filter using asymmetric key
func (s *NotificationServer) installKeyFilter(topicName string, key *ecdsa.PrivateKey) (filterID string, err error) {
	topic := MakeTopicAsBytes([]byte(topicName))
	filter := whisper.Filter{
		KeyAsym:  key,
		Topics:   [][]byte{topic},
		AllowP2P: true,
	}
	filterID, err = s.whisper.Subscribe(&filter)
	if err != nil {
		return "", fmt.Errorf("failed installing filter: %v", err)
	}

	log.Info(fmt.Sprintf("installed key filter %v for topic %x (%s)", filterID, topic, topicName))
	return
}

// requestProcessorLoop processes incoming client requests, by listening to a given filter,
// and executing process function on each incoming message
func (s *NotificationServer) requestProcessorLoop(filterID string, topicWatched string, fn messageProcessingFn) {
	log.Debug(fmt.Sprintf("request processor started: %s", topicWatched))

	filter := s.whisper.GetFilter(filterID)
	if filter == nil {
		log.Warn(fmt.Sprintf("filter is not installed: %s (for topic '%s')", filterID, topicWatched))
		return
	}

	ticker := time.NewTicker(time.Millisecond * 50)

	for {
		select {
		case <-ticker.C:
			messages := filter.Retrieve()
			for _, msg := range messages {
				if err := fn(msg); err != nil {
					log.Warn("failed processing incoming request", "error", err)
				}
			}
		case <-s.quit:
			log.Debug("request processor stopped", "topic", topicWatched)
			return
		}
	}
}

// makeSessionKey generates and saves random SymKey, allowing to establish secure
// channel between server and client
func (s *NotificationServer) makeSessionKey(keyName string) (sessionKey, sessionKeyDerived []byte, err error) {
	// wipe out previous occurrence of symmetric key
	s.whisper.DeleteSymKey(keyName)

	sessionKey, err = makeSessionKey()
	if err != nil {
		return nil, nil, err
	}

	keyName, err = s.whisper.AddSymKey(keyName, sessionKey)
	if err != nil {
		return nil, nil, err
	}

	sessionKeyDerived, err = s.whisper.GetSymKey(keyName)
	if err != nil {
		return nil, nil, err
	}

	return
}
