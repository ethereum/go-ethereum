package test

import (
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/log"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func GetNetworkInterface(netInterface string) (*net.Interface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var iface *net.Interface
	ip := net.ParseIP(netInterface)
	if ip == nil {
		//look for eth0 or the specified override
		for _, i := range ifaces {
			if i.Name == netInterface {
				iface = &i
			}
		}
	} else {

		for _, i := range ifaces {
			addrs, err := i.Addrs()
			if err == nil && addrs != nil {
				for _, addr := range addrs {
					ip, _, err = net.ParseCIDR(addr.String())
					fmt.Println(ip.String())
					if err == nil && ip.String() == netInterface {

						iface = &i
					}
				}
			}
		}
	}

	return iface, nil
}

type udpFrameOptions struct {
	sourceIP, destIP     net.IP
	sourcePort, destPort uint16
	sourceMac, destMac   net.HardwareAddr //we won't implemenent ARP as docker will supply the mac addresses we need
	isIPv6               bool
	payloadBytes         []byte
}

func spoofedWrite(toaddr *net.UDPAddr, fromaddr *net.UDPAddr, what string, packet []byte, macAddr string, netInterface string) error {

	mac, err := net.ParseMAC(macAddr)
	if err != nil {
		return err
	}

	iface, err := GetNetworkInterface(netInterface)
	if err != nil {
		return err
	}

	if nil == iface {
		return fmt.Errorf("interface not found: " + netInterface)
	}

	opts := udpFrameOptions{
		sourceIP:     fromaddr.IP.To4(),
		destIP:       toaddr.IP.To4(),
		sourcePort:   uint16(fromaddr.Port),
		destPort:     uint16(toaddr.Port),
		sourceMac:    iface.HardwareAddr,
		destMac:      mac,
		isIPv6:       false,
		payloadBytes: packet,
	}

	handle, err := pcap.OpenLive(iface.Name, 65536, true, pcap.BlockForever)
	if err != nil {
		return err
	}

	defer handle.Close()

	rawPacket, err := createSerializedUDPFrame(opts)
	if err != nil {
		return err
	}

	if err := handle.WritePacketData(rawPacket); err != nil {
		return err
	}

	log.Trace(">> "+what, "from", fromaddr, "addr", toaddr, "err", err)
	return err
}

// createSerializedUDPFrame creates an Ethernet frame encapsulating our UDP
// packet for injection to the local network
func createSerializedUDPFrame(opts udpFrameOptions) ([]byte, error) {

	buf := gopacket.NewSerializeBuffer()
	serializeOpts := gopacket.SerializeOptions{
		FixLengths:       true,
		ComputeChecksums: true,
	}
	ethernetType := layers.EthernetTypeIPv4
	if opts.isIPv6 {
		ethernetType = layers.EthernetTypeIPv6
	}
	eth := &layers.Ethernet{
		SrcMAC:       opts.sourceMac,
		DstMAC:       opts.destMac,
		EthernetType: ethernetType,
	}
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(opts.sourcePort),
		DstPort: layers.UDPPort(opts.destPort),
		// we configured "Length" and "Checksum" to be set for us
	}
	if !opts.isIPv6 {
		ip := &layers.IPv4{
			SrcIP:    opts.sourceIP,
			DstIP:    opts.destIP,
			Protocol: layers.IPProtocolUDP,
			Version:  4,
			TTL:      32,
		}
		udp.SetNetworkLayerForChecksum(ip)
		err := gopacket.SerializeLayers(buf, serializeOpts, eth, ip, udp, gopacket.Payload(opts.payloadBytes))
		if err != nil {
			return nil, err
		}
	} else {
		ip := &layers.IPv6{
			SrcIP:      opts.sourceIP,
			DstIP:      opts.destIP,
			NextHeader: layers.IPProtocolUDP,
			Version:    6,
			HopLimit:   32,
		}
		ip.LayerType()
		udp.SetNetworkLayerForChecksum(ip)
		err := gopacket.SerializeLayers(buf, serializeOpts, eth, ip, udp, gopacket.Payload(opts.payloadBytes))
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func getMacAddr() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var as []string
	for _, ifa := range interfaces {
		a := ifa.HardwareAddr.String()
		if a != "" {
			as = append(as, a)
		}
	}
	return as, nil
}
