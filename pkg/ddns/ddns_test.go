package ddns

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Second)
	defer cancel()

	client := NewMockClient(t)

	currentIP := net.ParseIP(`1.0.0.1`).To4()
	client.EXPECT().GetCurrentIPv4(ctx).
		Return(currentIP, `dns_record_id`, nil).
		Once()

	updatedIP := net.ParseIP(`1.1.1.1`).To4()
	client.EXPECT().SetARecordIP(ctx, updatedIP, `dns_record_id`).
		Return(nil).
		Once()

	ticker := make(chan struct{})
	go func() {
		ticker <- struct{}{}
		close(ticker)
	}()

	// dns record content should be updated
	run(ctx, ticker, client, func() (net.IP, error) {
		return updatedIP, nil
	})

	ticker = make(chan struct{})
	go func() {
		ticker <- struct{}{}
		close(ticker)
	}()

	// dns record update should be skipped
	run(ctx, ticker, client, func() (net.IP, error) {
		return net.ParseIP(`1.1.1.1`).To4(), nil
	})
}
