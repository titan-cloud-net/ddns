// Package cloudflare provides a Cloudflare DNS client implementation for the DDNS service.
// It interacts with the Cloudflare API to manage DNS A records for dynamic DNS updates.
package cloudflare

import (
	"context"
	"fmt"
	"net"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/dns"
	"github.com/cloudflare/cloudflare-go/v6/zones"

	"github.com/titan-cloud-net/ddns/pkg/ddns"
)

// client implements the ddns.Client interface using the Cloudflare API
type client struct {
	zoneID *string // Cloudflare zone ID, populated during startup
	*cloudflare.Client
}

// NewClient creates a new Cloudflare DNS client and registers lifecycle hooks
// The client is initialized during the fx app startup phase
func NewClient(cfg ddns.Config) (dc ddns.Client, err error) {
	client := &client{Client: cloudflare.NewClient()}

	// initialize the zone ID before the service begins
	client.zoneID, err = client.findZoneID(context.Background(), cfg.ZoneName)
	return client, err
}

// GetCurrentIPv4 retrieves the current IPv4 address from the Cloudflare DNS A record
// It returns the IP address, record ID, and any error encountered during the lookup
func (c *client) GetCurrentIPv4(ctx context.Context) (ip net.IP, recordID string, err error) {
	// Query Cloudflare API for A records in the specified zone
	res, err := c.DNS.Records.List(ctx,
		dns.RecordListParams{
			ZoneID: cloudflare.F(*c.zoneID),
			Type:   cloudflare.F(dns.RecordListParamsTypeA)})
	if err != nil {
		err = fmt.Errorf("failed to find A record IP: %w", err)
		return
	}

	if len(res.Result) != 0 {
		ipv4 := net.ParseIP(res.Result[0].Content)
		if ipv4 != nil {
			ip = ipv4
		}
		recordID = res.Result[0].ID
	}
	return
}

// SetARecordIP updates the DNS A record with the specified IP address
// It uses the record ID to target the specific record to update
func (c *client) SetARecordIP(ctx context.Context, ip net.IP, recordID string) (err error) {
	_, err = c.DNS.Records.Edit(ctx, recordID,
		dns.RecordEditParams{
			ZoneID: cloudflare.F(*c.zoneID),
			Body:   dns.ARecordParam{Content: cloudflare.F(ip.To4().String())}})
	if err != nil {
		err = fmt.Errorf("failed to edit DNS A record: %s", err)
	}
	return
}

// findZoneID queries the Cloudflare API to find the zone ID for the specified zone name
// The zone ID is required for subsequent DNS record operations
func (c *client) findZoneID(ctx context.Context, zone string) (zoneID *string, err error) {
	// Query Cloudflare API for zones matching the specified name
	res, err := c.Zones.List(ctx, zones.ZoneListParams{Name: cloudflare.F(zone)})
	if err != nil {
		return nil, fmt.Errorf("failed to find public ip: %w", err)
	}

	if len(res.Result) != 0 {
		zoneID = &res.Result[0].ID
		return
	}
	err = fmt.Errorf("failed to find id for zone %s", zone)
	return
}
