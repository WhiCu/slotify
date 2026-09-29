package application

import "time"

type User struct {
	TTL time.Duration `koanf:"ttl" validate:"required,gt=0"`
}

type Config struct {
	User User `koanf:"user" validate:"required"`
}

var defaultCfg = Config{
	User: User{
		TTL: 8 * time.Hour,
	},
}
