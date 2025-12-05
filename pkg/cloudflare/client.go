package cloudflare

import (
	"context"
	"fmt"
	"net"

	"github.com/cloudflare/cloudflare-go/v6"
	"github.com/cloudflare/cloudflare-go/v6/dns"
	"github.com/cloudflare/cloudflare-go/v6/zones"
	"go.uber.org/fx"

	"titan-cloud-net/ddns/pkg/ddns"
)

type client struct {
	zone   string
	zoneID *string
	*cloudflare.Client
}

type params struct {
	fx.In
	fx.Lifecycle
	fx.Shutdowner

	ddns.Zone
}

func NewClient(p params) ddns.Client {
	client := &client{zone: string(p.Zone), Client: cloudflare.NewClient()}
	p.Lifecycle.Append(fx.StartHook(client.start))
	return client
}

func (c *client) GetCurrentIPv4(ctx context.Context) (ip *net.IP, recordID string, err error) {
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
			ip = &ipv4
		}
		recordID = res.Result[0].ID
	}
	return
}

func (c *client) SetARecordIP(ctx context.Context, ip *net.IP, recordID string) (err error) {
	_, err = c.DNS.Records.Edit(ctx, recordID,
		dns.RecordEditParams{
			ZoneID: cloudflare.F(*c.zoneID),
			Body:   dns.ARecordParam{Content: cloudflare.F(ip.To4().String())}})
	if err != nil {
		err = fmt.Errorf("failed to edit DNS A record: %s", err)
	}
	return
}

func (c *client) start(ctx context.Context) (err error) {
	c.zoneID, err = c.findZoneID(ctx, c.zone)
	return
}

func (c *client) findZoneID(ctx context.Context, zone string) (zoneID *string, err error) {
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
