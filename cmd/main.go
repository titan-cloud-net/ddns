package main

import (
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/titan-cloud-net/ddns/pkg/cloudflare"
	"github.com/titan-cloud-net/ddns/pkg/ddns"
	"github.com/titan-cloud-net/ddns/pkg/logger"
	"github.com/titan-cloud-net/ddns/pkg/netlink"
)

func main() {
	log := logger.New()
	fx.New(
		fx.WithLogger(func() fxevent.Logger {
			return &fxevent.SlogLogger{Logger: log}
		}),
		fx.Provide(
			netlink.New,
			ddns.NewConfig,
			cloudflare.NewClient),
		fx.Invoke(
			netlink.Invoke,
			cloudflare.Invoke,
			ddns.Invoke)).
		Run()
}
