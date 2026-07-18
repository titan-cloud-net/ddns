package ddns

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	ZoneName string `env:"DNS_ZONE"`
}

func NewConfig() (cfg Config, err error) {
	err = env.ParseWithOptions(&cfg, env.Options{RequiredIfNoDef: true})
	return
}
