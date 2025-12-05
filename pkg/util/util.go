package util

import "net"

func FindPublicIP() (ip *net.IP, err error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok &&
			!ipNet.IP.IsPrivate() &&
			!ipNet.IP.IsLinkLocalUnicast() &&
			!ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return &ipNet.IP, nil
			}

			if ipNet.IP.To16() != nil {
				return &ipNet.IP, nil
			}
		}
	}
	return
}
