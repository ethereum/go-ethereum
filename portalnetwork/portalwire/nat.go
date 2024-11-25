package portalwire

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
	extPort  int // the mapped port returned by the NAT interface
	nextTime mclock.AbsTime
}

// setupPortMapping starts the port mapping loop if necessary.
// Note: this needs to be called after the LocalNode instance has been set on the server.
func (p *PortalProtocol) setupPortMapping() {
	// portMappingRegister will receive up to two values: one for the TCP port if
	// listening is enabled, and one more for enabling UDP port mapping if discovery is
	// enabled. We make it buffered to avoid blocking setup while a mapping request is in
	// progress.
	p.portMappingRegister = make(chan *portMapping, 2)

	switch p.NAT.(type) {
	case nil:
		// No NAT interface configured.
		go p.consumePortMappingRequests()

	case nat.ExtIP:
		// ExtIP doesn't block, set the IP right away.
		ip, _ := p.NAT.ExternalIP()
		p.localNode.SetStaticIP(ip)
		go p.consumePortMappingRequests()

	case nat.STUN:
		// STUN doesn't block, set the IP right away.
		ip, _ := p.NAT.ExternalIP()
		p.localNode.SetStaticIP(ip)
		go p.consumePortMappingRequests()

	default:
		go p.portMappingLoop()
	}
}

func (p *PortalProtocol) consumePortMappingRequests() {
	for {
		select {
		case <-p.closeCtx.Done():
			return
		case <-p.portMappingRegister:
		}
	}
}

// portMappingLoop manages port mappings for UDP and TCP.
func (p *PortalProtocol) portMappingLoop() {
	newLogger := func(proto string, e int, i int) log.Logger {
		return log.New("proto", proto, "extport", e, "intport", i, "interface", p.NAT)
	}

	var (
		mappings  = make(map[string]*portMapping, 2)
		refresh   = mclock.NewAlarm(p.clock)
		extip     = mclock.NewAlarm(p.clock)
		lastExtIP net.IP
	)
	extip.Schedule(p.clock.Now())
	defer func() {
		refresh.Stop()
		extip.Stop()
		for _, m := range mappings {
			if m.extPort != 0 {
				log := newLogger(m.protocol, m.extPort, m.port)
				log.Debug("Deleting port mapping")
				p.NAT.DeleteMapping(m.protocol, m.extPort, m.port)
			}
		}
	}()

	for {
		// Schedule refresh of existing mappings.
		for _, m := range mappings {
			refresh.Schedule(m.nextTime)
		}

		select {
		case <-p.closeCtx.Done():
			return

		case <-extip.C():
			extip.Schedule(p.clock.Now().Add(extipRetryInterval))
			ip, err := p.NAT.ExternalIP()
			if err != nil {
				log.Debug("Couldn't get external IP", "err", err, "interface", p.NAT)
			} else if !ip.Equal(lastExtIP) {
				log.Debug("External IP changed", "ip", extip, "interface", p.NAT)
			} else {
				continue
			}
			// Here, we either failed to get the external IP, or it has changed.
			lastExtIP = ip
			p.localNode.SetStaticIP(ip)
			p.Log.Debug("set static ip in nat", "ip", p.localNode.Node().IP().String())
			// Ensure port mappings are refreshed in case we have moved to a new network.
			for _, m := range mappings {
				m.nextTime = p.clock.Now()
			}

		case m := <-p.portMappingRegister:
			if m.protocol != "TCP" && m.protocol != "UDP" {
				panic("unknown NAT protocol name: " + m.protocol)
			}
			mappings[m.protocol] = m
			m.nextTime = p.clock.Now()

		case <-refresh.C():
			for _, m := range mappings {
				if p.clock.Now() < m.nextTime {
					continue
				}

				external := m.port
				if m.extPort != 0 {
					external = m.extPort
				}
				log := newLogger(m.protocol, external, m.port)

				log.Trace("Attempting port mapping")
				port, err := p.NAT.AddMapping(m.protocol, external, m.port, m.name, portMapDuration)
				if err != nil {
					log.Debug("Couldn't add port mapping", "err", err)
					m.extPort = 0
					m.nextTime = p.clock.Now().Add(portMapRetryInterval)
					continue
				}
				// It was mapped!
				m.extPort = int(port)
				m.nextTime = p.clock.Now().Add(portMapRefreshInterval)
				if external != m.extPort {
					log = newLogger(m.protocol, m.extPort, m.port)
					log.Info("NAT mapped alternative port")
				} else {
					log.Info("NAT mapped port")
				}

				// Update port in local ENR.
				switch m.protocol {
				case "TCP":
					p.localNode.Set(enr.TCP(m.extPort))
				case "UDP":
					p.localNode.SetFallbackUDP(m.extPort)
				}
			}
		}
	}
}
