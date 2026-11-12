package storage

import (
	"net"
	"net/url"
	"time"
)

type Config struct {
	AccessType string `koanf:"access_type" validate:"required,oneof=SQL"`

	Host string `koanf:"host" validate:"required,hostname|ip"`
	Port string `koanf:"port" validate:"required,numeric"`
	User string `koanf:"user" validate:"required"`
	Pass string `koanf:"pass" validate:"required"`
	Name string `koanf:"name" validate:"required"`

	Insecure bool `koanf:"insecure"`

	MaxOpenConns int32 `koanf:"max_open_conns" validate:"required,gt=0"`
	MinOpenConns int32 `koanf:"min_open_conns" validate:"required,gt=0,ltefield=MaxOpenConns"`

	ConnMaxLifetime       time.Duration `koanf:"conn_max_lifetime" validate:"required,gt=0"`
	ConnMaxLifetimeJitter time.Duration `koanf:"conn_max_lifetime_jitter" validate:"required,gte=0"`
	ConnMaxIdleTime       time.Duration `koanf:"conn_max_idle_time" validate:"required,gt=0"`
}

var defaultCfg = Config{
	AccessType: "SQL",

	Insecure: true,

	MaxOpenConns: 16,
	MinOpenConns: 2,

	ConnMaxLifetime:       1 * time.Hour,
	ConnMaxLifetimeJitter: 10 * time.Minute,
	ConnMaxIdleTime:       30 * time.Minute,
}

func (c Config) HostPort() string {
	return net.JoinHostPort(c.Host, c.Port)
}

func (c Config) DSN() string {
	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(c.User, c.Pass),
		Host:   c.HostPort(),
		Path:   c.Name,
	}

	q := u.Query()
	if c.Insecure {
		q.Set("sslmode", "disable")
	} else {
		q.Set("sslmode", "require")
	}
	u.RawQuery = q.Encode()

	return u.String()
}

func (c Config) String() string {
	return "storage.Config{Host: " + c.Host + ":" + c.Port +
		", Name: " + c.Name + ", User: " + c.User + ", Pass: ***REDACTED***}"
}
