package example_test

import (
	"fmt"
	"os"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/internetgateway1"
)

// Use discovered WANPPPConnection1 services to find external IP addresses.
func Example_WANPPPConnection1_GetExternalIPAddress() {
	clients, errors, err := internetgateway1.NewWANPPPConnection1Clients()
	extIPClients := make([]GetExternalIPAddresser, len(clients))
	for i, client := range clients {
		extIPClients[i] = client
	}
	DisplayExternalIPResults(extIPClients, errors, err)
	// Output:
}

// Use discovered WANIPConnection services to find external IP addresses.
func Example_WANIPConnection_GetExternalIPAddress() {
	clients, errors, err := internetgateway1.NewWANIPConnection1Clients()
	extIPClients := make([]GetExternalIPAddresser, len(clients))
	for i, client := range clients {
		extIPClients[i] = client
	}
	DisplayExternalIPResults(extIPClients, errors, err)
	// Output:
}

type GetExternalIPAddresser interface {
	GetExternalIPAddress() (NewExternalIPAddress string, err error)
	GetServiceClient() *goupnp.ServiceClient
}

func DisplayExternalIPResults(clients []GetExternalIPAddresser, errors []error, err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error discovering service with UPnP: ", err)
		return
	}

	if len(errors) > 0 {
		fmt.Fprintf(os.Stderr, "Error discovering %d services:\n", len(errors))
		for _, err := range errors {
			fmt.Println("  ", err)
		}
	}

	fmt.Fprintf(os.Stderr, "Successfully discovered %d services:\n", len(clients))
	for _, client := range clients {
		device := &client.GetServiceClient().RootDevice.Device

		fmt.Fprintln(os.Stderr, "  Device:", device.FriendlyName)
		if addr, err := client.GetExternalIPAddress(); err != nil {
			fmt.Fprintf(os.Stderr, "    Failed to get external IP address: %v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "    External IP address: %v\n", addr)
		}
	}
}
