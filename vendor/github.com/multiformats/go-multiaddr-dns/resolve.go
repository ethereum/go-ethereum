package madns

import (
	"context"
	"fmt"
	"net"
	"strings"

	ma "github.com/multiformats/go-multiaddr"
)

var ResolvableProtocols = []ma.Protocol{DnsaddrProtocol, Dns4Protocol, Dns6Protocol}
var DefaultResolver = &Resolver{Backend: net.DefaultResolver}

type backend interface {
	LookupIPAddr(context.Context, string) ([]net.IPAddr, error)
	LookupTXT(context.Context, string) ([]string, error)
}

type Resolver struct {
	Backend backend
}

type MockBackend struct {
	IP  map[string][]net.IPAddr
	TXT map[string][]string
}

func (r *MockBackend) LookupIPAddr(ctx context.Context, name string) ([]net.IPAddr, error) {
	results, ok := r.IP[name]
	if ok {
		return results, nil
	} else {
		return []net.IPAddr{}, nil
	}
}

func (r *MockBackend) LookupTXT(ctx context.Context, name string) ([]string, error) {
	results, ok := r.TXT[name]
	if ok {
		return results, nil
	} else {
		return []string{}, nil
	}
}

func Matches(maddr ma.Multiaddr) bool {
	protos := maddr.Protocols()
	if len(protos) == 0 {
		return false
	}

	for _, p := range ResolvableProtocols {
		if protos[0].Code == p.Code {
			return true
		}
	}

	return false
}

func Resolve(ctx context.Context, maddr ma.Multiaddr) ([]ma.Multiaddr, error) {
	return DefaultResolver.Resolve(ctx, maddr)
}

func (r *Resolver) Resolve(ctx context.Context, maddr ma.Multiaddr) ([]ma.Multiaddr, error) {
	if !Matches(maddr) {
		return []ma.Multiaddr{maddr}, nil
	}

	protos := maddr.Protocols()
	if protos[0].Code == Dns4Protocol.Code {
		return r.resolveDns4(ctx, maddr)
	}
	if protos[0].Code == Dns6Protocol.Code {
		return r.resolveDns6(ctx, maddr)
	}
	if protos[0].Code == DnsaddrProtocol.Code {
		return r.resolveDnsaddr(ctx, maddr)
	}

	panic("unreachable")
}

func (r *Resolver) resolveDns4(ctx context.Context, maddr ma.Multiaddr) ([]ma.Multiaddr, error) {
	value, err := maddr.ValueForProtocol(Dns4Protocol.Code)
	if err != nil {
		return nil, fmt.Errorf("error resolving %s: %s", maddr.String(), err)
	}

	encap := ma.Split(maddr)[1:]

	result := []ma.Multiaddr{}
	records, err := r.Backend.LookupIPAddr(ctx, value)
	if err != nil {
		return result, err
	}

	for _, r := range records {
		ip4 := r.IP.To4()
		if ip4 == nil {
			continue
		}
		ip4maddr, err := ma.NewMultiaddr("/ip4/" + ip4.String())
		if err != nil {
			return result, err
		}
		parts := append([]ma.Multiaddr{ip4maddr}, encap...)
		result = append(result, ma.Join(parts...))
	}
	return result, nil
}

func (r *Resolver) resolveDns6(ctx context.Context, maddr ma.Multiaddr) ([]ma.Multiaddr, error) {
	value, err := maddr.ValueForProtocol(Dns6Protocol.Code)
	if err != nil {
		return nil, fmt.Errorf("error resolving %s: %s", maddr.String(), err)
	}

	encap := ma.Split(maddr)[1:]

	result := []ma.Multiaddr{}
	records, err := r.Backend.LookupIPAddr(ctx, value)
	if err != nil {
		return result, err
	}

	for _, r := range records {
		if r.IP.To4() != nil {
			continue
		}
		ip6maddr, err := ma.NewMultiaddr("/ip6/" + r.IP.To16().String())
		if err != nil {
			return result, err
		}
		parts := append([]ma.Multiaddr{ip6maddr}, encap...)
		result = append(result, ma.Join(parts...))
	}
	return result, nil
}

func (r *Resolver) resolveDnsaddr(ctx context.Context, maddr ma.Multiaddr) ([]ma.Multiaddr, error) {
	value, err := maddr.ValueForProtocol(DnsaddrProtocol.Code)
	if err != nil {
		return nil, fmt.Errorf("error resolving %s: %s", maddr.String(), err)
	}

	trailer := ma.Split(maddr)[1:]

	result := []ma.Multiaddr{}
	records, err := r.Backend.LookupTXT(ctx, "_dnsaddr."+value)
	if err != nil {
		return result, err
	}

	for _, r := range records {
		rv := strings.Split(r, "dnsaddr=")
		if len(rv) != 2 {
			continue
		}

		rmaddr, err := ma.NewMultiaddr(rv[1])
		if err != nil {
			return result, err
		}

		if matchDnsaddr(rmaddr, trailer) {
			result = append(result, rmaddr)
		}
	}
	return result, nil
}

// XXX probably insecure
func matchDnsaddr(maddr ma.Multiaddr, trailer []ma.Multiaddr) bool {
	parts := ma.Split(maddr)
	if ma.Join(parts[len(parts)-len(trailer):]...).Equal(ma.Join(trailer...)) {
		return true
	}
	return false
}
