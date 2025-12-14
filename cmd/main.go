package main

import (
	"go.uber.org/fx"

	"github.com/titan-cloud-net/ddns/pkg/cloudflare"
	"github.com/titan-cloud-net/ddns/pkg/ddns"
)

func main() {
	fx.New(
		fx.Provide(
			ddns.NewConfig,
			cloudflare.NewClient),
		fx.Invoke(
			cloudflare.Invoke,
			ddns.Invoke)).
		Run()
}
