//go:build windows

package vpn

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"TomorrowClient/internal/model"
)

// These constants describe the virtual TUN adapter used by the xray+tun2socks
// path. sing-box configures its own adapter internally. The adapter name is a
// package variable (activeTunName) set from settings at connect time so the
// route setup, the stats matcher and the sing-box config all agree.
const (
	tunAddr    = "172.19.0.2"
	tunGateway = "172.19.0.1"
	tunMask    = "255.255.255.252"
)

// activeTunName is the WinTun adapter name in use for the current connection.
// It defaults to the built-in name and is overwritten by the engine on Connect.
var activeTunName = model.DefaultTunName

// run executes a command hidden and returns combined output on error.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	hidden(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s: %v: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}

// configureTun assigns a static IP to the tun2socks WinTun adapter.
func configureTun() error {
	return run("netsh", "interface", "ip", "set", "address",
		"name="+activeTunName, "static", tunAddr, tunMask, tunGateway)
}

// setDNSOnTun points the tun adapter at the given DNS server.
func setDNSOnTun(dns string) error {
	if dns == "" {
		return nil
	}
	return run("netsh", "interface", "ip", "set", "dns",
		"name="+activeTunName, "static", dns)
}

// addRoutes sends the default route through the tun gateway (low metric) and
// pins a /32 host route to the proxy server through the physical gateway so
// the encrypted tunnel traffic itself does not loop back into the tunnel.
func addRoutes(serverIP string) error {
	// Default route split into two /1 halves so it wins over the existing
	// 0.0.0.0/0 without having to delete it.
	if err := run("route", "add", "0.0.0.0", "mask", "128.0.0.0", tunGateway, "metric", "1"); err != nil {
		return err
	}
	if err := run("route", "add", "128.0.0.0", "mask", "128.0.0.0", tunGateway, "metric", "1"); err != nil {
		return err
	}
	// Bypass route for the server itself via the current physical gateway.
	gw, iface, err := defaultGateway()
	if err == nil && gw != "" && serverIP != "" {
		_ = run("route", "add", serverIP, "mask", "255.255.255.255", gw, "metric", "1", "if", iface)
	}
	return nil
}

// delRoutes removes the routes added by addRoutes.
func delRoutes(serverIP string) {
	_ = run("route", "delete", "0.0.0.0", "mask", "128.0.0.0", tunGateway)
	_ = run("route", "delete", "128.0.0.0", "mask", "128.0.0.0", tunGateway)
	if serverIP != "" {
		_ = run("route", "delete", serverIP, "mask", "255.255.255.255")
	}
}

// resolveServerIP resolves a host to its first IPv4 address. If host is already
// an IP it is returned unchanged.
func resolveServerIP(host string) string {
	if ip := net.ParseIP(host); ip != nil {
		return host
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return ""
	}
	for _, ip := range ips {
		if v4 := ip.To4(); v4 != nil {
			return v4.String()
		}
	}
	return ""
}

// defaultGateway returns the current default gateway IP and its interface
// index by querying the routing table via "route print".
func defaultGateway() (gw string, iface string, err error) {
	cmd := exec.Command("route", "print", "0.0.0.0")
	hidden(cmd)
	out, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	// Parse the "Active Routes" lines; the 0.0.0.0 default route holds the
	// gateway in column 3 and the interface IP in column 4.
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) >= 5 && f[0] == "0.0.0.0" && f[1] == "0.0.0.0" {
			return f[2], ifaceIndexFor(f[3]), nil
		}
	}
	return "", "", fmt.Errorf("default gateway not found")
}

// ifaceIndexFor maps an interface IP to its numeric index, which "route add"
// accepts via the "if" argument. Falls back to the IP string on failure.
func ifaceIndexFor(ipStr string) string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ipStr
	}
	target := net.ParseIP(ipStr)
	for _, ifi := range ifaces {
		addrs, _ := ifi.Addrs()
		for _, a := range addrs {
			if ipn, ok := a.(*net.IPNet); ok && ipn.IP.Equal(target) {
				return fmt.Sprintf("%d", ifi.Index)
			}
		}
	}
	return ipStr
}
