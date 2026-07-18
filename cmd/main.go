package main

import (
	"go.uber.org/fx"

	"github.com/titan-cloud-net/ddns/pkg/cloudflare"
	"github.com/titan-cloud-net/ddns/pkg/ddns"
	"github.com/titan-cloud-net/ddns/pkg/logger"
	"github.com/titan-cloud-net/ddns/pkg/netlink"
)

func main() {
	fx.New(
		logger.New(),
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
