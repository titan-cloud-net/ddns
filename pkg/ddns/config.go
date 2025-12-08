package ddns

import (
	"time"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Interval time.Duration `env:"IP_CHECK_INTERVAL" envDefault:"300ms"`
	ZoneName string        `env:"DNS_ZONE"`
}

func NewConfig() (cfg Config, err error) {
	err = env.ParseWithOptions(&cfg, env.Options{RequiredIfNoDef: true})
	return
}
