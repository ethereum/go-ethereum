package goupnp

import (
	"fmt"
	"github.com/huin/goupnp/soap"
)

// ServiceClient is a SOAP client, root device and the service for the SOAP
// client rolled into one value. The root device and service are intended to be
// informational.
type ServiceClient struct {
	SOAPClient *soap.SOAPClient
	RootDevice *RootDevice
	Service    *Service
}

func NewServiceClients(searchTarget string) (clients []ServiceClient, errors []error, err error) {
	var maybeRootDevices []MaybeRootDevice
	if maybeRootDevices, err = DiscoverDevices(searchTarget); err != nil {
		return
	}

	clients = make([]ServiceClient, 0, len(maybeRootDevices))

	for _, maybeRootDevice := range maybeRootDevices {
		if maybeRootDevice.Err != nil {
			errors = append(errors, maybeRootDevice.Err)
			continue
		}

		device := &maybeRootDevice.Root.Device
		srvs := device.FindService(searchTarget)
		if len(srvs) == 0 {
			errors = append(errors, fmt.Errorf("goupnp: service %q not found within device %q (UDN=%q)",
				searchTarget, device.FriendlyName, device.UDN))
			continue
		}

		for _, srv := range srvs {
			clients = append(clients, ServiceClient{
				SOAPClient: srv.NewSOAPClient(),
				RootDevice: maybeRootDevice.Root,
				Service:    srv,
			})
		}
	}

	return
}

// GetServiceClient returns the ServiceClient itself. This is provided so that the
// service client attributes can be accessed via an interface method on a
// wrapping type.
func (client *ServiceClient) GetServiceClient() *ServiceClient {
	return client
}
