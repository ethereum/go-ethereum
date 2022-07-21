package builder

import (
	"errors"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/gorilla/mux"

	"github.com/flashbots/go-boost-utils/bls"
	boostTypes "github.com/flashbots/go-boost-utils/types"

	"github.com/flashbots/go-utils/httplogger"
)

const (
	_PathStatus            = "/eth/v1/builder/status"
	_PathRegisterValidator = "/eth/v1/builder/validators"
	_PathGetHeader         = "/eth/v1/builder/header/{slot:[0-9]+}/{parent_hash:0x[a-fA-F0-9]+}/{pubkey:0x[a-fA-F0-9]+}"
	_PathGetPayload        = "/eth/v1/builder/blinded_blocks"
)

type Service struct {
	srv *http.Server
}

func (s *Service) Start() {
	log.Info("Service started")
	go s.srv.ListenAndServe()
}

func getRouter(localRelay *LocalRelay) http.Handler {
	router := mux.NewRouter()

	// Add routes
	router.HandleFunc("/", localRelay.handleIndex).Methods(http.MethodGet)
	router.HandleFunc(_PathStatus, localRelay.handleStatus).Methods(http.MethodGet)
	router.HandleFunc(_PathRegisterValidator, localRelay.handleRegisterValidator).Methods(http.MethodPost)
	router.HandleFunc(_PathGetHeader, localRelay.handleGetHeader).Methods(http.MethodGet)
	router.HandleFunc(_PathGetPayload, localRelay.handleGetPayload).Methods(http.MethodPost)

	// Add logging and return router
	loggedRouter := httplogger.LoggingMiddleware(router)
	return loggedRouter
}

func NewService(listenAddr string, localRelay *LocalRelay) *Service {
	return &Service{
		srv: &http.Server{
			Addr:    listenAddr,
			Handler: getRouter(localRelay),
			/*
				ReadTimeout:
				ReadHeaderTimeout:
				WriteTimeout:
				IdleTimeout:
			*/
		},
	}
}

type BuilderConfig struct {
	EnableValidatorChecks bool
	BuilderSecretKey      string
	RelaySecretKey        string
	ListenAddr            string
	GenesisForkVersion    string
	BellatrixForkVersion  string
	GenesisValidatorsRoot string
	BeaconEndpoint        string
	RemoteRelayEndpoint   string
}

func Register(stack *node.Node, backend *eth.Ethereum, cfg *BuilderConfig) error {
	envRelaySkBytes, err := hexutil.Decode(cfg.RelaySecretKey)
	if err != nil {
		return errors.New("incorrect builder API secret key provided")
	}

	relaySk, err := bls.SecretKeyFromBytes(envRelaySkBytes[:])
	if err != nil {
		return errors.New("incorrect builder API secret key provided")
	}

	envBuilderSkBytes, err := hexutil.Decode(cfg.BuilderSecretKey)
	if err != nil {
		return errors.New("incorrect builder API secret key provided")
	}

	builderSk, err := bls.SecretKeyFromBytes(envBuilderSkBytes[:])
	if err != nil {
		return errors.New("incorrect builder API secret key provided")
	}

	genesisForkVersionBytes, err := hexutil.Decode(cfg.GenesisForkVersion)
	var genesisForkVersion [4]byte
	copy(genesisForkVersion[:], genesisForkVersionBytes[:4])
	builderSigningDomain := boostTypes.ComputeDomain(boostTypes.DomainTypeAppBuilder, genesisForkVersion, boostTypes.Root{})

	genesisValidatorsRoot := boostTypes.Root(common.HexToHash(cfg.GenesisValidatorsRoot))
	bellatrixForkVersionBytes, err := hexutil.Decode(cfg.BellatrixForkVersion)
	var bellatrixForkVersion [4]byte
	copy(bellatrixForkVersion[:], bellatrixForkVersionBytes[:4])
	proposerSigningDomain := boostTypes.ComputeDomain(boostTypes.DomainTypeBeaconProposer, bellatrixForkVersion, genesisValidatorsRoot)

	var beaconClient IBeaconClient
	beaconClient = NewBeaconClient(cfg.BeaconEndpoint)

	localRelay := NewLocalRelay(relaySk, beaconClient, builderSigningDomain, proposerSigningDomain, ForkData{cfg.GenesisForkVersion, cfg.BellatrixForkVersion, cfg.GenesisValidatorsRoot}, cfg.EnableValidatorChecks)

	var relay IRelay
	if cfg.RemoteRelayEndpoint != "" {
		relay, err = NewRemoteRelay(cfg.RemoteRelayEndpoint, localRelay)
		if err != nil {
			return err
		}
	} else {
		relay = localRelay
	}

	builderBackend := NewBuilder(builderSk, beaconClient, relay, builderSigningDomain)
	builderService := NewService(cfg.ListenAddr, localRelay)
	builderService.Start()

	backend.SetSealedBlockHook(builderBackend.newSealedBlock)
	backend.SetForkchoiceHook(builderBackend.onForkchoice)
	return nil
}
