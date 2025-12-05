package ddns

import (
	"context"
	"log/slog"
	"net"
	"time"

	"go.uber.org/fx"

	"titan-cloud-net/ddns/pkg/util"
)

type Zone string

type Client interface {
	GetCurrentIPv4(ctx context.Context) (ip *net.IP, recordID string, err error)
	SetARecordIP(ctx context.Context, ip *net.IP, recordID string) (err error)
}

type params struct {
	fx.In
	fx.Lifecycle

	Client
}

func Invoke(p params) {
	ctx, cancel := context.WithCancel(context.Background())
	p.Lifecycle.Append(fx.StopHook(func() {
		cancel()
	}))

	go run(ctx, p.Client)
}

func run(ctx context.Context, client Client) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			ip, err := util.FindPublicIP()
			if err != nil || ip == nil {
				slog.ErrorContext(ctx, "failed to find interface with public ip", "ip", ip, "error", err)
				continue
			}

			dnsIP, recordID, err := client.GetCurrentIPv4(ctx)
			if err != nil || dnsIP == nil {
				slog.ErrorContext(ctx, "failed to find interface with public ip", "ip", ip, "error", err)
				continue
			}

			if ip.String() != dnsIP.String() {
				err := client.SetARecordIP(ctx, ip, recordID)
				if err != nil {
					slog.ErrorContext(ctx, "failed to set a record content", "error", err)
					continue
				}
				slog.InfoContext(ctx, "dns a record updated", "ip", ip.String(), "content", dnsIP.String())
			}
		}
	}
}
