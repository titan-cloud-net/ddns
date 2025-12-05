package main

import (
	"flag"

	"go.uber.org/fx"

	"titan-cloud-net/ddns/pkg/cloudflare"
	"titan-cloud-net/ddns/pkg/ddns"
)

func main() {
	zone := flag.String("dns-zone", "", "Cloudflare DNS Zone")
	flag.Parse()

	fx.New(
		fx.Supply(ddns.Zone(*zone)),
		fx.Provide(cloudflare.NewClient),
		fx.Invoke(ddns.Invoke)).
		Run()
}
