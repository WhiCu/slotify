package application

import (
	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/slotify/internal/config"
)

func newConfig(i do.Injector) (Config, error) {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return Config{}, err
	}
	def := defaultCfg
	return config.GetConfig(k, "app", &def)
}

var Package = do.Package(
	do.Lazy(newConfig),
)
