package test

import (
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

func spoofedWrite(fromAddr *net.UDPAddr, toAddr *net.UDPAddr, payload []byte) error {
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
