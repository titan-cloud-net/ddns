// Package util provides utility functions for network operations.
package util

import "net"

// FindPublicIP searches through all network interfaces to find a public IP address.
// It filters out private, link-local, and loopback addresses, preferring IPv4 over IPv6.
// Returns the first public IP address found, or nil if none exists.
func FindPublicIP() (ip net.IP, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	// Iterate through all addresses to find a public IP
	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok &&
			!ipNet.IP.IsPrivate() &&
			!ipNet.IP.IsLinkLocalUnicast() &&
			!ipNet.IP.IsLoopback() {
			// Check for IPv4 address first
			if ipNet.IP.To4() != nil {
				return ipNet.IP, nil
			}

			// Fall back to IPv6 if no IPv4 is available
			if ipNet.IP.To16() != nil {
				return ipNet.IP, nil
			}
		}
	}
	return
}
