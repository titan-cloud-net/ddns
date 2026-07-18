// Package ddns provides dynamic DNS functionality to keep DNS records synchronized
// with the current public IP address of the system.
package ddns

import (
	"context"
	"log/slog"
	"net"
	"sync/atomic"

	"github.com/titan-cloud-net/ddns/pkg/netlink"
	"go.uber.org/fx"
)

// Client defines the interface for DNS client operations
type Client interface {
	// GetCurrentIPv4 retrieves the current IPv4 address from the DNS record
	// and returns the IP, record ID, and any error encountered
	GetCurrentIPv4(ctx context.Context) (ip net.IP, recordID string, err error)

	// SetARecordIP updates the DNS A record with the specified IP address
	SetARecordIP(ctx context.Context, ip net.IP, recordID string) (err error)

	// GetCurrentIPv6 retrieves the current IPv6 address from the DNS record
	// and returns the IP, record ID, and any error encountered
	GetCurrentIPv6(ctx context.Context) (ip net.IP, recordID string, err error)

	// SetAAAARecordIP updates the DNS AAAA record with the specified IP address
	SetAAAARecordIP(ctx context.Context, ip net.IP, recordID string) (err error)
}

type params struct {
	fx.In
	fx.Lifecycle

	Client
	Config
	netlink.Watcher
}

// publicIPv4 and publicIPv6 store the last known public IP address for each
// family using atomic operations for thread-safe access across goroutines
var (
	publicIPv4 atomic.Pointer[net.IP]
	publicIPv6 atomic.Pointer[net.IP]
)

func Invoke(p params) {
	ctx, cancel := context.WithCancel(context.Background())
	p.Lifecycle.Append(fx.StartStopHook(
		func() {
			go run(ctx, p.Client, p.Watcher)
		},
		func() {
			cancel()
		}))
}

func run(ctx context.Context, client Client, watcher netlink.Watcher) {
	for ip := range watcher.Watch() {
		slog.Debug("interface event", "ip", ip.String())
		if !ip.IsPrivate() &&
			!ip.IsLinkLocalUnicast() &&
			!ip.IsLoopback() &&
			ip.IsGlobalUnicast() {
			// Update the A record for an IPv4 address
			if ip.To4() != nil {
				updateIPv4(ctx, client, ip)
				continue
			}

			// Update the AAAA record for an IPv6 address
			if ip.To16() != nil {
				updateIPv6(ctx, client, ip)
				continue
			}
		}
	}
}

// updateIPv4 checks the current public IPv4 address and updates the DNS A record if it has changed
func updateIPv4(ctx context.Context, client Client, ip net.IP) {
	// Load the last known public IP from atomic storage
	lastIP := publicIPv4.Load()

	// If IP hasn't changed since last check, skip DNS query
	if lastIP != nil && lastIP.Equal(ip) {
		return
	}

	dnsIP, recordID, err := client.GetCurrentIPv4(ctx)
	if err != nil || dnsIP == nil {
		slog.Error("failed to find interface with public ip", "ip", ip, "error", err)
		return
	}

	// Compare the current public IP with the DNS record IP
	if ip.String() != dnsIP.String() {
		// IP addresses don't match, update the DNS record
		err := client.SetARecordIP(ctx, ip, recordID)
		if err != nil {
			slog.Error("failed to set a record content", "error", err)
			return
		}
		slog.Info("DNS A record updated", "updated_content", ip.String(), "previous_content", dnsIP.String())
	}
	publicIPv4.Store(&ip)
}

// updateIPv6 checks the current public IPv6 address and updates the DNS AAAA record if it has changed
func updateIPv6(ctx context.Context, client Client, ip net.IP) {
	// Load the last known public IP from atomic storage
	lastIP := publicIPv6.Load()

	// If IP hasn't changed since last check, skip DNS query
	if lastIP != nil && lastIP.Equal(ip) {
		return
	}

	dnsIP, recordID, err := client.GetCurrentIPv6(ctx)
	if err != nil || dnsIP == nil {
		slog.Error("failed to find interface with public ip", "ip", ip, "error", err)
		return
	}

	// Compare the current public IP with the DNS record IP
	if ip.String() != dnsIP.String() {
		// IP addresses don't match, update the DNS record
		err := client.SetAAAARecordIP(ctx, ip, recordID)
		if err != nil {
			slog.Error("failed to set aaaa record content", "error", err)
			return
		}
		slog.Info("DNS AAAA record updated", "updated_content", ip.String(), "previous_content", dnsIP.String())
	}
	publicIPv6.Store(&ip)
}
