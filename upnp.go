// package upnp provides a simple and opinionated interface to UPnP-enabled
// routers, allowing users to forward ports and discover their external IP
// address. Specific quirks:
//
// - When attempting to discover UPnP-enabled routers on the network, only the
// first such router is returned. If you have multiple routers, this may cause
// some trouble. But why would you do that?
//
// - Forwarded ports are always symmetric, e.g. the router's port 9980 will be
// mapped to the client's port 9980. This will be unacceptable for some
// purposes, but too bad. Symmetric mappings are the desired behavior 99% of
// the time, and they save a function argument.
//
// - TCP and UDP protocols are forwarded together.
//
// - Ports are forwarded permanently. Some other implementations lease a port
// mapping for a set duration, and then renew it periodically. This is nice,
// because it means mappings won't stick around after they've served their
// purpose. Unfortunately, some routers only support permanent mappings, so this
// package has chosen to support the lowest common denominator. To un-forward a
// port, you must use the Clear function (or do it manually).
//
// Once you've discovered your router, you can retrieve its address by calling
// its Location method. This address can be supplied to Load to connect to the
// router directly, which is much faster than calling Discover.
package upnp

import (
	"errors"
	"net"
	"net/url"
	"strings"

	"github.com/huin/goupnp"
	"github.com/huin/goupnp/dcps/internetgateway1"
)

// An IGD provides an interface to the most commonly used functions of an
// Internet Gateway Device: discovering the external IP, and forwarding ports.
type IGD interface {
	ExternalIP() (string, error)
	Forward(port uint16, description, proto string) error
	Clear(port uint16, proto string) error
	Location() string
}

// upnpDevice implements the IGD interface. It is essentially a bridge between
// IGD and the internetgateway1.WANIPConnection1 and
// internetgateway1.WANPPPConnection1 types.
type upnpDevice struct {
	client interface {
		GetExternalIPAddress() (string, error)
		AddPortMapping(string, uint16, string, uint16, string, bool, string, uint32) error
		DeletePortMapping(string, uint16, string) error
		GetServiceClient() *goupnp.ServiceClient
	}
}

// ExternalIP returns the router's external IP.
func (u *upnpDevice) ExternalIP() (string, error) {
	return u.client.GetExternalIPAddress()
}

// Forward forwards the specified port, and adds its description to the
// router's port mapping table.
//
// TODO: is desc necessary?
// TODO: take an int instead? More convenient.
// EDIT: now explicitly passing in protocol to open for
func (u *upnpDevice) Forward(port uint16, desc, proto string) error {
	proto = strings.ToUpper(proto)
	ip, err := u.getInternalIP()
	if err != nil {
		return err
	}
	return u.client.AddPortMapping("", port, proto, port, ip, true, desc, 0)
}

// Clear un-forwards a port, removing it from the router's port mapping table.
// EDIT: now passing in specific proto to clear
func (u *upnpDevice) Clear(port uint16, proto string) error {
	proto = strings.ToUpper(proto)
	return u.client.DeletePortMapping("", port, proto)
}

// Location returns the URL of the router, for future lookups (see Load).
func (u *upnpDevice) Location() string {
	return u.client.GetServiceClient().Location.String()
}

// getInternalIP returns the user's local IP.
func (u *upnpDevice) getInternalIP() (string, error) {
	host, _, _ := net.SplitHostPort(u.client.GetServiceClient().RootDevice.URLBase.Host)
	devIP := net.ParseIP(host)
	if devIP == nil {
		return "", errors.New("could not determine router's internal IP")
	}

	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}

		for _, addr := range addrs {
			switch x := addr.(type) {
			case *net.IPNet:
				if x.Contains(devIP) {
					return x.IP.String(), nil
				}
			}
		}
	}

	return "", errors.New("could not determine internal IP")
}

// Discover scans the local network for routers and returns the first
// UPnP-enabled router it encounters.
//
// TODO: if more than one client is found, only return those on the same
// subnet as the user?
func Discover() (IGD, error) {
	pppclients, _, _ := internetgateway1.NewWANPPPConnection1Clients()
	if len(pppclients) > 0 {
		return &upnpDevice{pppclients[0]}, nil
	}
	ipclients, _, _ := internetgateway1.NewWANIPConnection1Clients()
	if len(ipclients) > 0 {
		return &upnpDevice{ipclients[0]}, nil
	}
	return nil, errors.New("no UPnP-enabled gateway found")
}

// Load connects to the router service specified by rawurl. This is much
// faster than Discover. Generally, Load should only be called with values
// returned by the IGD's Location method.
func Load(rawurl string) (IGD, error) {
	loc, err := url.Parse(rawurl)
	if err != nil {
		return nil, err
	}
	pppclients, _ := internetgateway1.NewWANPPPConnection1ClientsByURL(loc)
	if len(pppclients) > 0 {
		return &upnpDevice{pppclients[0]}, nil
	}
	ipclients, _ := internetgateway1.NewWANIPConnection1ClientsByURL(loc)
	if len(ipclients) > 0 {
		return &upnpDevice{ipclients[0]}, nil
	}
	return nil, errors.New("no UPnP-enabled gateway found at URL " + rawurl)
}
