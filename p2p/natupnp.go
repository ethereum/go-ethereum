package p2p

// Just enough UPnP to be able to forward ports
//

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	upnpDiscoverAttempts = 3
	upnpDiscoverTimeout  = 5 * time.Second
)

// UPNP returns a NAT port mapper that uses UPnP. It will attempt to
// discover the address of your router using UDP broadcasts.
func UPNP() NAT {
	return &upnpNAT{}
}

type upnpNAT struct {
	serviceURL string
	ourIP      string
}

func (n *upnpNAT) String() string {
	return "UPNP"
}

func (n *upnpNAT) discover() error {
	if n.serviceURL != "" {
		// already discovered
		return nil
	}

	ssdp, err := net.ResolveUDPAddr("udp4", "239.255.255.250:1900")
	if err != nil {
		return err
	}
	// TODO: try on all network interfaces simultaneously.
	// Broadcasting on 0.0.0.0 could select a random interface
	// to send on (platform specific).
	conn, err := net.ListenPacket("udp4", ":0")
	if err != nil {
		return err
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(10 * time.Second))
	st := "ST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n"
	buf := bytes.NewBufferString(
		"M-SEARCH * HTTP/1.1\r\n" +
			"HOST: 239.255.255.250:1900\r\n" +
			st +
			"MAN: \"ssdp:discover\"\r\n" +
			"MX: 2\r\n\r\n")
	message := buf.Bytes()
	answerBytes := make([]byte, 1024)
	for i := 0; i < upnpDiscoverAttempts; i++ {
		_, err = conn.WriteTo(message, ssdp)
		if err != nil {
			return err
		}
		nn, _, err := conn.ReadFrom(answerBytes)
		if err != nil {
			continue
		}
		answer := string(answerBytes[0:nn])
		if strings.Index(answer, "\r\n"+st) < 0 {
			continue
		}
		// HTTP header field names are case-insensitive.
		// http://www.w3.org/Protocols/rfc2616/rfc2616-sec4.html#sec4.2
		locString := "\r\nlocation: "
		answer = strings.ToLower(answer)
		locIndex := strings.Index(answer, locString)
		if locIndex < 0 {
			continue
		}
		loc := answer[locIndex+len(locString):]
		endIndex := strings.Index(loc, "\r\n")
		if endIndex < 0 {
			continue
		}
		locURL := loc[0:endIndex]
		var serviceURL string
		serviceURL, err = getServiceURL(locURL)
		if err != nil {
			return err
		}
		var ourIP string
		ourIP, err = getOurIP()
		if err != nil {
			return err
		}
		n.serviceURL = serviceURL
		n.ourIP = ourIP
		return nil
	}
	return errors.New("UPnP port discovery failed.")
}

func (n *upnpNAT) GetExternalAddress() (addr net.IP, err error) {
	if err := n.discover(); err != nil {
		return nil, err
	}
	info, err := n.getStatusInfo()
	return net.ParseIP(info.externalIpAddress), err
}

func (n *upnpNAT) AddPortMapping(protocol string, extport, intport int, description string, lifetime time.Duration) error {
	if err := n.discover(); err != nil {
		return err
	}

	// A single concatenation would break ARM compilation.
	message := "<u:AddPortMapping xmlns:u=\"urn:schemas-upnp-org:service:WANIPConnection:1\">\r\n" +
		"<NewRemoteHost></NewRemoteHost><NewExternalPort>" + strconv.Itoa(extport)
	message += "</NewExternalPort><NewProtocol>" + protocol + "</NewProtocol>"
	message += "<NewInternalPort>" + strconv.Itoa(extport) + "</NewInternalPort>" +
		"<NewInternalClient>" + n.ourIP + "</NewInternalClient>" +
		"<NewEnabled>1</NewEnabled><NewPortMappingDescription>"
	message += description +
		"</NewPortMappingDescription><NewLeaseDuration>" + fmt.Sprint(lifetime/time.Second) +
		"</NewLeaseDuration></u:AddPortMapping>"

	// TODO: check response to see if the port was forwarded
	_, err := soapRequest(n.serviceURL, "AddPortMapping", message)
	return err
}

func (n *upnpNAT) DeletePortMapping(protocol string, externalPort, internalPort int) error {
	if err := n.discover(); err != nil {
		return err
	}

	message := "<u:DeletePortMapping xmlns:u=\"urn:schemas-upnp-org:service:WANIPConnection:1\">\r\n" +
		"<NewRemoteHost></NewRemoteHost><NewExternalPort>" + strconv.Itoa(externalPort) +
		"</NewExternalPort><NewProtocol>" + protocol + "</NewProtocol>" +
		"</u:DeletePortMapping>"

	// TODO: check response to see if the port was deleted
	_, err := soapRequest(n.serviceURL, "DeletePortMapping", message)
	return err
}

type statusInfo struct {
	externalIpAddress string
}

func (n *upnpNAT) getStatusInfo() (info statusInfo, err error) {
	message := "<u:GetStatusInfo xmlns:u=\"urn:schemas-upnp-org:service:WANIPConnection:1\">\r\n" +
		"</u:GetStatusInfo>"

	var response *http.Response
	response, err = soapRequest(n.serviceURL, "GetStatusInfo", message)
	if err != nil {
		return
	}

	// TODO: Write a soap reply parser. It has to eat the Body and envelope tags...

	response.Body.Close()
	return
}

// service represents the Service type in an UPnP xml description.
// Only the parts we care about are present and thus the xml may have more
// fields than present in the structure.
type service struct {
	ServiceType string `xml:"serviceType"`
	ControlURL  string `xml:"controlURL"`
}

// deviceList represents the deviceList type in an UPnP xml description.
// Only the parts we care about are present and thus the xml may have more
// fields than present in the structure.
type deviceList struct {
	XMLName xml.Name `xml:"deviceList"`
	Device  []device `xml:"device"`
}

// serviceList represents the serviceList type in an UPnP xml description.
// Only the parts we care about are present and thus the xml may have more
// fields than present in the structure.
type serviceList struct {
	XMLName xml.Name  `xml:"serviceList"`
	Service []service `xml:"service"`
}

// device represents the device type in an UPnP xml description.
// Only the parts we care about are present and thus the xml may have more
// fields than present in the structure.
type device struct {
	XMLName     xml.Name    `xml:"device"`
	DeviceType  string      `xml:"deviceType"`
	DeviceList  deviceList  `xml:"deviceList"`
	ServiceList serviceList `xml:"serviceList"`
}

// specVersion represents the specVersion in a UPnP xml description.
// Only the parts we care about are present and thus the xml may have more
// fields than present in the structure.
type specVersion struct {
	XMLName xml.Name `xml:"specVersion"`
	Major   int      `xml:"major"`
	Minor   int      `xml:"minor"`
}

// root represents the Root document for a UPnP xml description.
// Only the parts we care about are present and thus the xml may have more
// fields than present in the structure.
type root struct {
	XMLName     xml.Name `xml:"root"`
	SpecVersion specVersion
	Device      device
}

func getChildDevice(d *device, deviceType string) *device {
	dl := d.DeviceList.Device
	for i := 0; i < len(dl); i++ {
		if dl[i].DeviceType == deviceType {
			return &dl[i]
		}
	}
	return nil
}

func getChildService(d *device, serviceType string) *service {
	sl := d.ServiceList.Service
	for i := 0; i < len(sl); i++ {
		if sl[i].ServiceType == serviceType {
			return &sl[i]
		}
	}
	return nil
}

func getOurIP() (ip string, err error) {
	hostname, err := os.Hostname()
	if err != nil {
		return
	}
	p, err := net.LookupIP(hostname)
	if err != nil && len(p) > 0 {
		return
	}
	return p[0].String(), nil
}

func getServiceURL(rootURL string) (url string, err error) {
	r, err := http.Get(rootURL)
	if err != nil {
		return
	}
	defer r.Body.Close()
	if r.StatusCode >= 400 {
		err = errors.New(string(r.StatusCode))
		return
	}
	var root root
	err = xml.NewDecoder(r.Body).Decode(&root)

	if err != nil {
		return
	}
	a := &root.Device
	if a.DeviceType != "urn:schemas-upnp-org:device:InternetGatewayDevice:1" {
		err = errors.New("No InternetGatewayDevice")
		return
	}
	b := getChildDevice(a, "urn:schemas-upnp-org:device:WANDevice:1")
	if b == nil {
		err = errors.New("No WANDevice")
		return
	}
	c := getChildDevice(b, "urn:schemas-upnp-org:device:WANConnectionDevice:1")
	if c == nil {
		err = errors.New("No WANConnectionDevice")
		return
	}
	d := getChildService(c, "urn:schemas-upnp-org:service:WANIPConnection:1")
	if d == nil {
		err = errors.New("No WANIPConnection")
		return
	}
	url = combineURL(rootURL, d.ControlURL)
	return
}

func combineURL(rootURL, subURL string) string {
	protocolEnd := "://"
	protoEndIndex := strings.Index(rootURL, protocolEnd)
	a := rootURL[protoEndIndex+len(protocolEnd):]
	rootIndex := strings.Index(a, "/")
	return rootURL[0:protoEndIndex+len(protocolEnd)+rootIndex] + subURL
}

func soapRequest(url, function, message string) (r *http.Response, err error) {
	fullMessage := "<?xml version=\"1.0\" ?>" +
		"<s:Envelope xmlns:s=\"http://schemas.xmlsoap.org/soap/envelope/\" s:encodingStyle=\"http://schemas.xmlsoap.org/soap/encoding/\">\r\n" +
		"<s:Body>" + message + "</s:Body></s:Envelope>"

	req, err := http.NewRequest("POST", url, strings.NewReader(fullMessage))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "text/xml ; charset=\"utf-8\"")
	req.Header.Set("User-Agent", "Darwin/10.0.0, UPnP/1.0, MiniUPnPc/1.3")
	//req.Header.Set("Transfer-Encoding", "chunked")
	req.Header.Set("SOAPAction", "\"urn:schemas-upnp-org:service:WANIPConnection:1#"+function+"\"")
	req.Header.Set("Connection", "Close")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")

	r, err = http.DefaultClient.Do(req)
	if err != nil {
		return
	}

	if r.Body != nil {
		defer r.Body.Close()
	}

	if r.StatusCode >= 400 {
		// log.Stderr(function, r.StatusCode)
		err = errors.New("Error " + strconv.Itoa(r.StatusCode) + " for " + function)
		r = nil
		return
	}
	return
}
