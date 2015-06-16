package comms

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/rpc/api"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/rs/cors"
)

var (
	// main HTTP rpc listener
	httpListener         *stoppableTCPListener
	listenerStoppedError = fmt.Errorf("Listener has stopped")
)

type HttpConfig struct {
	ListenAddress string
	ListenPort    uint
	CorsDomain    string
}

func StartHttp(cfg HttpConfig, codec codec.Codec, apis ...api.EthereumApi) error {
	if httpListener != nil {
		if fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort) != httpListener.Addr().String() {
			return fmt.Errorf("RPC service already running on %s ", httpListener.Addr().String())
		}
		return nil // RPC service already running on given host/port
	}

	l, err := newStoppableTCPListener(fmt.Sprintf("%s:%d", cfg.ListenAddress, cfg.ListenPort))
	if err != nil {
		glog.V(logger.Error).Infof("Can't listen on %s:%d: %v", cfg.ListenAddress, cfg.ListenPort, err)
		return err
	}
	httpListener = l

	api := api.Merge(apis...)
	var handler http.Handler
	if len(cfg.CorsDomain) > 0 {
		var opts cors.Options
		opts.AllowedMethods = []string{"POST"}
		opts.AllowedOrigins = strings.Split(cfg.CorsDomain, " ")

		c := cors.New(opts)
		handler = newStoppableHandler(c.Handler(gethHttpHandler(codec, api)), l.stop)
	} else {
		handler = newStoppableHandler(gethHttpHandler(codec, api), l.stop)
	}

	go http.Serve(l, handler)

	return nil
}

func StopHttp() {
	if httpListener != nil {
		httpListener.Stop()
		httpListener = nil
	}
}
