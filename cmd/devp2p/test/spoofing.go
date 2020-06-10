package test

import (
	"fmt"
	"net"

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

func sppofedWrite(fromAddr *net.UDPAddr, toAddr *net.UDPAddr, payload []byte) error {
	device := "lo0"
	handle, err := pcap.OpenLive(device, 1500, false, pcap.BlockForever)
	if err != nil {
		return err
	}

	lo := layers.Loopback{
		Family: layers.ProtocolFamilyIPv4,
	}

	ip := layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    net.IP{127, 0, 0, 1},
		DstIP:    net.IP{127, 0, 0, 1},
		Protocol: layers.IPProtocolUDP,
	}

	udp := layers.UDP{
		SrcPort: layers.UDPPort(fromAddr.Port),
		DstPort: layers.UDPPort(toAddr.Port),
	}
	udp.SetNetworkLayerForChecksum(&ip)

	options := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	buffer := gopacket.NewSerializeBuffer()

	if err = gopacket.SerializeLayers(buffer, options, &lo, &ip, &udp, gopacket.Payload(payload)); err != nil {
		return err
	}
	outgoingPacket := buffer.Bytes()

	if err = handle.WritePacketData(outgoingPacket); err != nil {
		return err
	}
	return nil
}
