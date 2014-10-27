package p2p

import (
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"
)

const (
	DialerTimeout             = 180 //seconds
	KeepAlivePeriod           = 60  //minutes
	portMappingUpdateInterval = 900 // seconds = 15 mins
	upnpDiscoverAttempts      = 3
)

// Dialer is not an interface in net, so we define one
// *net.Dialer conforms to this
type Dialer interface {
	Dial(network, address string) (net.Conn, error)
}

type Network interface {
	Start() error
	Listener(net.Addr) (net.Listener, error)
	Dialer(net.Addr) (Dialer, error)
	NewAddr(string, int) (addr net.Addr, err error)
	ParseAddr(string) (addr net.Addr, err error)
}

type NAT interface {
	GetExternalAddress() (addr net.IP, err error)
	AddPortMapping(protocol string, externalPort, internalPort int, description string, timeout int) (mappedExternalPort int, err error)
	DeletePortMapping(protocol string, externalPort, internalPort int) (err error)
}

type TCPNetwork struct {
	nat     NAT
	natType NATType
	quit    chan chan bool
	ports   chan string
}

type NATType int

const (
	NONE = iota
	UPNP
	PMP
)

const (
	portMappingTimeout = 1200 // 20 mins
)

func NewTCPNetwork(natType NATType) (net *TCPNetwork) {
	return &TCPNetwork{
		natType: natType,
		ports:   make(chan string),
	}
}

func (self *TCPNetwork) Dialer(addr net.Addr) (Dialer, error) {
	return &net.Dialer{
		Timeout: DialerTimeout * time.Second,
		// KeepAlive: KeepAlivePeriod * time.Minute,
		LocalAddr: addr,
	}, nil
}

func (self *TCPNetwork) Listener(addr net.Addr) (net.Listener, error) {
	if self.natType == UPNP {
		_, port, _ := net.SplitHostPort(addr.String())
		if self.quit == nil {
			self.quit = make(chan chan bool)
			go self.updatePortMappings()
		}
		self.ports <- port
	}
	return net.Listen(addr.Network(), addr.String())
}

func (self *TCPNetwork) Start() (err error) {
	switch self.natType {
	case NONE:
	case UPNP:
		nat, uerr := upnpDiscover(upnpDiscoverAttempts)
		if uerr != nil {
			err = fmt.Errorf("UPNP failed: ", uerr)
		} else {
			self.nat = nat
		}
	case PMP:
		err = fmt.Errorf("PMP not implemented")
	default:
		err = fmt.Errorf("Invalid NAT type: %v", self.natType)
	}
	return
}

func (self *TCPNetwork) Stop() {
	q := make(chan bool)
	self.quit <- q
	<-q
}

func (self *TCPNetwork) addPortMapping(lport int) (err error) {
	_, err = self.nat.AddPortMapping("TCP", lport, lport, "p2p listen port", portMappingTimeout)
	if err != nil {
		logger.Errorf("unable to add port mapping on %v: %v", lport, err)
	} else {
		logger.Debugf("succesfully added port mapping on %v", lport)
	}
	return
}

func (self *TCPNetwork) updatePortMappings() {
	timer := time.NewTimer(portMappingUpdateInterval * time.Second)
	lports := []int{}
out:
	for {
		select {
		case port := <-self.ports:
			int64lport, _ := strconv.ParseInt(port, 10, 16)
			lport := int(int64lport)
			if err := self.addPortMapping(lport); err != nil {
				lports = append(lports, lport)
			}
		case <-timer.C:
			for lport := range lports {
				if err := self.addPortMapping(lport); err != nil {
				}
			}
		case errc := <-self.quit:
			errc <- true
			break out
		}
	}

	timer.Stop()
	for lport := range lports {
		if err := self.nat.DeletePortMapping("TCP", lport, lport); err != nil {
			logger.Debugf("unable to remove port mapping on %v: %v", lport, err)
		} else {
			logger.Debugf("succesfully removed port mapping on %v", lport)
		}
	}
}

func (self *TCPNetwork) NewAddr(host string, port int) (net.Addr, error) {
	ip, err := self.lookupIP(host)
	if err == nil {
		return &net.TCPAddr{
			IP:   ip,
			Port: port,
		}, nil
	}
	return nil, err
}

func (self *TCPNetwork) ParseAddr(address string) (net.Addr, error) {
	host, port, err := net.SplitHostPort(address)
	if err == nil {
		iport, _ := strconv.Atoi(port)
		addr, e := self.NewAddr(host, iport)
		return addr, e
	}
	return nil, err
}

func (*TCPNetwork) lookupIP(host string) (ip net.IP, err error) {
	if ip = net.ParseIP(host); ip != nil {
		return
	}

	var ips []net.IP
	ips, err = net.LookupIP(host)
	if err != nil {
		logger.Warnln(err)
		return
	}
	if len(ips) == 0 {
		err = fmt.Errorf("No IP addresses available for %v", host)
		logger.Warnln(err)
		return
	}
	if len(ips) > 1 {
		// Pick a random IP address, simulating round-robin DNS.
		rand.Seed(time.Now().UTC().UnixNano())
		ip = ips[rand.Intn(len(ips))]
	} else {
		ip = ips[0]
	}
	return
}
