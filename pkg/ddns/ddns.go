// Package ddns provides dynamic DNS functionality to keep DNS records synchronized
// with the current public IP address of the system.
package ddns

import (
	"context"
	"log/slog"
	"net"
	"time"

	"github.com/titan-cloud-net/ddns/pkg/util"
	"go.uber.org/fx"
)

// Client defines the interface for DNS client operations
type Client interface {
	// GetCurrentIPv4 retrieves the current IPv4 address from the DNS record
	// and returns the IP, record ID, and any error encountered
	GetCurrentIPv4(ctx context.Context) (ip net.IP, recordID string, err error)

	// SetARecordIP updates the DNS A record with the specified IP address
	SetARecordIP(ctx context.Context, ip net.IP, recordID string) (err error)
}

type params struct {
	fx.In
	fx.Lifecycle

	Client
	Config
}

// findIP is a function type for finding the public IP address
type findIP func() (net.IP, error)

// publicIP stores the last known public IP address using atomic operations
// for thread-safe access across goroutines
var publicIP net.IP

// Invoke initializes and starts the DDNS monitoring service
// It sets up a background goroutine that continuously monitors and updates DNS records
func Invoke(p params) {
	ctx, cancel := context.WithCancel(context.Background())
	// Register a stop hook to cancel the context when the application shuts down
	p.Lifecycle.Append(fx.StopHook(func() {
		cancel()
	}))

	// Start the DDNS monitoring loop in a separate goroutine
	slog.Info("ip check", "interval", p.Interval.String())
	ticker := make(chan struct{})
	// Create a ticker goroutine that sends periodic signals at the configured interval
	go func() {
		for {
			ticker <- struct{}{}
			time.Sleep(p.Interval)
		}
	}()

	go run(ctx, ticker, p.Client, util.FindPublicIP)
}

// run executes the main DDNS monitoring loop
// It periodically checks if the system's public IP has changed and updates the DNS record if needed
func run(ctx context.Context, tick <-chan struct{}, client Client, f findIP) {
	for {
		select {
		case <-ctx.Done():
			return
		case _, open := <-tick:
			if !open {
				return
			}
			// Perform IP check and update if necessary
			updateIP(ctx, client, f)
		}
	}
}

// updateIP checks the current public IP and updates the DNS record if it has changed
func updateIP(ctx context.Context, client Client, f findIP) {
	// Load the last known public IP from atomic storage
	lastIP := publicIP

	// Find the current public IP address
	ip, err := f()
	if err != nil || ip == nil {
		slog.ErrorContext(ctx, "failed to find interface with public ip", "ip", ip, "error", err)
		return
	}

	// If IP hasn't changed since last check, skip DNS query
	if lastIP != nil && lastIP.Equal(ip) {
		return
	}

	dnsIP, recordID, err := client.GetCurrentIPv4(ctx)
	if err != nil || dnsIP == nil {
		slog.ErrorContext(ctx, "failed to find interface with public ip", "ip", ip, "error", err)
		return
	}

	// Compare the current public IP with the DNS record IP
	if ip.String() != dnsIP.String() {
		// IP addresses don't match, update the DNS record
		err := client.SetARecordIP(ctx, ip, recordID)
		if err != nil {
			slog.ErrorContext(ctx, "failed to set a record content", "error", err)
			return
		}
		publicIP = ip
		slog.InfoContext(ctx, "DNS A record updated", "updated_content", ip.String(), "previous_content", dnsIP.String())
	}
}
