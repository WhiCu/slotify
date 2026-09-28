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

func newUserRepository(i do.Injector) (*UserRepository, error) {
	storage, err := do.Invoke[*Storage](i)
	if err != nil {
		return nil, err
	}

	return NewUserRepository(storage), nil
}

func newSpaceRepository(i do.Injector) (*SpaceRepository, error) {
	storage, err := do.Invoke[*Storage](i)
	if err != nil {
		return nil, err
	}

	return NewSpaceRepository(storage), nil
}

func newReservationRepository(i do.Injector) (*ReservationRepository, error) {
	storage, err := do.Invoke[*Storage](i)
	if err != nil {
		return nil, err
	}

	return NewReservationRepository(storage), nil
}

var Package = do.Package(
	do.Lazy(newConfig),
	do.Lazy(newStorage),
	do.Lazy(newUserRepository),
	do.Lazy(newSpaceRepository),
	do.Lazy(newReservationRepository),
)
