package ddns

import (
	"context"
	"net"
	"testing"
	"time"
)

// TestRun verifies the DDNS monitoring loop behavior
func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()

	// Create a mock DNS client with expected method calls
	client := NewMockClient(t)

	// Set up the initial DNS record IP address
	currentIP := net.ParseIP(`1.0.0.1`).To4()
	client.EXPECT().GetCurrentIPv4(ctx).
		Return(currentIP, `dns_record_id`, nil).
		Once()

	// Set up the new public IP address that differs from DNS
	updatedIP := net.ParseIP(`1.1.1.1`).To4()
	client.EXPECT().SetARecordIP(ctx, updatedIP, `dns_record_id`).
		Return(nil).
		Once()

	// Create a ticker that sends one signal then closes
	ticker := make(chan struct{})
	go func() {
		ticker <- struct{}{}
		close(ticker)
	}()

	// Test Case 1: DNS record content should be updated when IP changes
	run(ctx, ticker, client, func() (net.IP, error) {
		return updatedIP, nil
	})

	// Create a new ticker for the second test
	ticker = make(chan struct{})
	go func() {
		ticker <- struct{}{}
		close(ticker)
	}()

	// Test Case 2: DNS record update should be skipped when IP hasn't changed
	run(ctx, ticker, client, func() (net.IP, error) {
		return net.ParseIP(`1.1.1.1`).To4(), nil
	})
}
