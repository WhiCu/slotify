package storage

import (
	"context"
	"log/slog"

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
	return config.GetConfig(k, "storage", &def)
}

func newStorage(i do.Injector) (*Storage, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, err
	}

	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, err
	}

	ctx, err := do.Invoke[context.Context](i)
	if err != nil {
		return nil, err
	}

	return NewStorage(ctx, log, cfg)
}

var Package = do.Package(
	do.Lazy(newConfig),
	do.Lazy(newStorage),
)
