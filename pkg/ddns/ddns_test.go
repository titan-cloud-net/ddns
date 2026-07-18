package ddns

import (
	"context"
	"net"
	"testing"
	"time"
)

// fakeWatcher is a minimal netlink.Watcher for tests; the real Watch mock
// lives in pkg/netlink's own test-only mocks_test.go and isn't importable here.
type fakeWatcher struct {
	updates chan net.IP
}

func (w *fakeWatcher) Watch() <-chan net.IP {
	return w.updates
}

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

	// Test Case 1: DNS record content should be updated when IP changes
	watcher := &fakeWatcher{updates: make(chan net.IP, 1)}
	watcher.updates <- updatedIP
	run(ctx, client, watcher)

	// Test Case 2: DNS record update should be skipped when IP hasn't changed
	watcher = &fakeWatcher{updates: make(chan net.IP, 1)}
	watcher.updates <- net.ParseIP(`1.1.1.1`).To4()
	run(ctx, client, watcher)
}
