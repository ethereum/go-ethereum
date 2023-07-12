package p2p

import (
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enr"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

const (
	portMapDuration        = 10 * time.Minute
	portMapRefreshInterval = 8 * time.Minute
	portMapRetryInterval   = 5 * time.Minute
	extipRetryInterval     = 2 * time.Minute
)

type portMapping struct {
	protocol string
	name     string
	port     int

	// for use by the portMappingLoop goroutine:
	extPort  int // if non-zero, this is mapped port returned by the NAT interface
	nextTime mclock.AbsTime
}

// setupPortMapping starts the port mapping loop if necessary.
// Note: this needs to be called after the LocalNode instance has been set on the server.
func (srv *Server) setupPortMapping() {
	// portMappingRegister will receive up to two values: one for
	// thr TCP port if listening is enabled, and one more for enabling UDP port mapping
	// if discovery is enabled. We make it buffered to avoid blocking their setup while
	// a port mapping request is in progress.
	srv.portMappingRegister = make(chan *portMapping, 2)

	switch srv.NAT.(type) {
	case nil:
		// No NAT interface, do nothing.
		srv.loopWG.Add(1)
		go srv.consumePortMappingRequests()

	case nat.ExtIP:
		// ExtIP doesn't block, set the IP right away.
		ip, _ := srv.NAT.ExternalIP()
		srv.localnode.SetStaticIP(ip)
		srv.loopWG.Add(1)
		go srv.consumePortMappingRequests()

	default:
		// Ask the router about the IP. This takes a while and blocks startup,
		// do it in the background.
		srv.loopWG.Add(1)
		go srv.portMappingLoop()
	}
}

func (srv *Server) consumePortMappingRequests() {
	defer srv.loopWG.Done()
	for {
		select {
		case <-srv.quit:
			return
		case <-srv.portMappingRegister:
		}
	}
}

// portMappingLoop manages port mappings for UDP and TCP.
func (srv *Server) portMappingLoop() {
	defer srv.loopWG.Done()

	newLogger := func(p string, e int, i int) log.Logger {
		return log.New("proto", p, "extport", e, "intport", i, "interface", srv.NAT)
	}

	var (
		refresh        = mclock.NewAlarm(srv.clock)
		mappings       = make(map[string]*portMapping, 2)
		lastExternalIP net.IP
	)
	defer func() {
		refresh.Stop()
		for _, m := range mappings {
			if m.extPort != 0 {
				log := newLogger(m.protocol, m.extPort, m.port)
				log.Debug("Deleting port mapping")
				srv.NAT.DeleteMapping(m.protocol, m.extPort, m.port)
			}
		}
	}()

	for {
		// Schedule next refresh.
		for _, p := range mappings {
			refresh.Schedule(p.nextTime)
		}

		select {
		case <-srv.quit:
			return

		case m := <-srv.portMappingRegister:
			if m.protocol != "TCP" && m.protocol != "UDP" {
				panic("unknown NAT protocol name: " + m.protocol)
			}
			mappings[m.protocol] = m
			m.nextTime = srv.clock.Now()

		case <-refresh.C():
			now := srv.clock.Now()

			// Get/update the external IP address.
			extip, err := srv.NAT.ExternalIP()
			if err != nil {
				log.Debug("Couldn't get external IP", "err", err, "interface", srv.NAT)
				srv.localnode.SetStaticIP(nil)
				refresh.Schedule(now.Add(extipRetryInterval))
			} else if !extip.Equal(lastExternalIP) {
				log.Debug("External IP changed", "ip", extip, "interface", srv.NAT)
				srv.localnode.SetStaticIP(extip)
				lastExternalIP = extip
			}

			// Update all mappings.
			for _, m := range mappings {
				if now < m.nextTime {
					continue
				}

				external := m.port
				if m.extPort != 0 {
					external = m.extPort
				}
				log := newLogger(m.protocol, external, m.port)

				log.Trace("Attempting port mapping")
				p, err := srv.NAT.AddMapping(m.protocol, external, m.port, m.name, portMapDuration)
				now = srv.clock.Now()
				if err != nil {
					log.Debug("Couldn't add port mapping", "err", err)
					m.extPort = 0
					m.nextTime = now.Add(portMapRetryInterval)
					continue
				}
				// It was mapped!
				external = int(p)
				m.nextTime = now.Add(portMapRefreshInterval)
				m.extPort = external
				if external != m.port {
					log = newLogger(m.protocol, external, m.port)
					log.Info("NAT mapped alternative port")
				} else {
					log.Info("NAT mapped port")
				}
				// Update port in local ENR.
				switch m.protocol {
				case "TCP":
					srv.localnode.Set(enr.TCP(external))
				case "UDP":
					srv.localnode.SetFallbackUDP(external)
				}
			}
		}
	}
}
