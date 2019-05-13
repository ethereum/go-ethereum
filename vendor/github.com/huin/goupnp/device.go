// This file contains XML structures for communicating with UPnP devices.

package goupnp

import (
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"

	"github.com/huin/goupnp/scpd"
	"github.com/huin/goupnp/soap"
)

const (
	DeviceXMLNamespace = "urn:schemas-upnp-org:device-1-0"
)

// RootDevice is the device description as described by section 2.3 "Device
// description" in
// http://upnp.org/specs/arch/UPnP-arch-DeviceArchitecture-v1.1.pdf
type RootDevice struct {
	XMLName     xml.Name    `xml:"root"`
	SpecVersion SpecVersion `xml:"specVersion"`
	URLBase     url.URL     `xml:"-"`
	URLBaseStr  string      `xml:"URLBase"`
	Device      Device      `xml:"device"`
}

// SetURLBase sets the URLBase for the RootDevice and its underlying components.
func (root *RootDevice) SetURLBase(urlBase *url.URL) {
	root.URLBase = *urlBase
	root.URLBaseStr = urlBase.String()
	root.Device.SetURLBase(urlBase)
}

// SpecVersion is part of a RootDevice, describes the version of the
// specification that the data adheres to.
type SpecVersion struct {
	Major int32 `xml:"major"`
	Minor int32 `xml:"minor"`
}

// Device is a UPnP device. It can have child devices.
type Device struct {
	DeviceType       string    `xml:"deviceType"`
	FriendlyName     string    `xml:"friendlyName"`
	Manufacturer     string    `xml:"manufacturer"`
	ManufacturerURL  URLField  `xml:"manufacturerURL"`
	ModelDescription string    `xml:"modelDescription"`
	ModelName        string    `xml:"modelName"`
	ModelNumber      string    `xml:"modelNumber"`
	ModelURL         URLField  `xml:"modelURL"`
	SerialNumber     string    `xml:"serialNumber"`
	UDN              string    `xml:"UDN"`
	UPC              string    `xml:"UPC,omitempty"`
	Icons            []Icon    `xml:"iconList>icon,omitempty"`
	Services         []Service `xml:"serviceList>service,omitempty"`
	Devices          []Device  `xml:"deviceList>device,omitempty"`

	// Extra observed elements:
	PresentationURL URLField `xml:"presentationURL"`
}

// VisitDevices calls visitor for the device, and all its descendent devices.
func (device *Device) VisitDevices(visitor func(*Device)) {
	visitor(device)
	for i := range device.Devices {
		device.Devices[i].VisitDevices(visitor)
	}
}

// VisitServices calls visitor for all Services under the device and all its
// descendent devices.
func (device *Device) VisitServices(visitor func(*Service)) {
	device.VisitDevices(func(d *Device) {
		for i := range d.Services {
			visitor(&d.Services[i])
		}
	})
}

// FindService finds all (if any) Services under the device and its descendents
// that have the given ServiceType.
func (device *Device) FindService(serviceType string) []*Service {
	var services []*Service
	device.VisitServices(func(s *Service) {
		if s.ServiceType == serviceType {
			services = append(services, s)
		}
	})
	return services
}

// SetURLBase sets the URLBase for the Device and its underlying components.
func (device *Device) SetURLBase(urlBase *url.URL) {
	device.ManufacturerURL.SetURLBase(urlBase)
	device.ModelURL.SetURLBase(urlBase)
	device.PresentationURL.SetURLBase(urlBase)
	for i := range device.Icons {
		device.Icons[i].SetURLBase(urlBase)
	}
	for i := range device.Services {
		device.Services[i].SetURLBase(urlBase)
	}
	for i := range device.Devices {
		device.Devices[i].SetURLBase(urlBase)
	}
}

func (device *Device) String() string {
	return fmt.Sprintf("Device ID %s : %s (%s)", device.UDN, device.DeviceType, device.FriendlyName)
}

// Icon is a representative image that a device might include in its
// description.
type Icon struct {
	Mimetype string   `xml:"mimetype"`
	Width    int32    `xml:"width"`
	Height   int32    `xml:"height"`
	Depth    int32    `xml:"depth"`
	URL      URLField `xml:"url"`
}

// SetURLBase sets the URLBase for the Icon.
func (icon *Icon) SetURLBase(url *url.URL) {
	icon.URL.SetURLBase(url)
}

// Service is a service provided by a UPnP Device.
type Service struct {
	ServiceType string   `xml:"serviceType"`
	ServiceId   string   `xml:"serviceId"`
	SCPDURL     URLField `xml:"SCPDURL"`
	ControlURL  URLField `xml:"controlURL"`
	EventSubURL URLField `xml:"eventSubURL"`
}

// SetURLBase sets the URLBase for the Service.
func (srv *Service) SetURLBase(urlBase *url.URL) {
	srv.SCPDURL.SetURLBase(urlBase)
	srv.ControlURL.SetURLBase(urlBase)
	srv.EventSubURL.SetURLBase(urlBase)
}

func (srv *Service) String() string {
	return fmt.Sprintf("Service ID %s : %s", srv.ServiceId, srv.ServiceType)
}

// RequestSCDP requests the SCPD (soap actions and state variables description)
// for the service.
func (srv *Service) RequestSCDP() (*scpd.SCPD, error) {
	if !srv.SCPDURL.Ok {
		return nil, errors.New("bad/missing SCPD URL, or no URLBase has been set")
	}
	s := new(scpd.SCPD)
	if err := requestXml(srv.SCPDURL.URL.String(), scpd.SCPDXMLNamespace, s); err != nil {
		return nil, err
	}
	return s, nil
}

func (srv *Service) NewSOAPClient() *soap.SOAPClient {
	return soap.NewSOAPClient(srv.ControlURL.URL)
}

// URLField is a URL that is part of a device description.
type URLField struct {
	URL url.URL `xml:"-"`
	Ok  bool    `xml:"-"`
	Str string  `xml:",chardata"`
}

func (uf *URLField) SetURLBase(urlBase *url.URL) {
	refUrl, err := url.Parse(uf.Str)
	if err != nil {
		uf.URL = url.URL{}
		uf.Ok = false
		return
	}

	uf.URL = *urlBase.ResolveReference(refUrl)
	uf.Ok = true
}
