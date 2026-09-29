package http

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	api "github.com/whicu/slotify/api/http"
	bookingapp "github.com/whicu/slotify/internal/application/booking"
	spaceapp "github.com/whicu/slotify/internal/application/space"
	userapp "github.com/whicu/slotify/internal/application/user"
	"github.com/whicu/slotify/internal/config"
	"github.com/whicu/slotify/internal/infrastructure/crypto"
)

func newConfig(i do.Injector) (Config, error) {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return Config{}, err
	}
	def := defaultCfg
	return config.GetConfig(k, "http", &def)
}

func newSecurityHandler(i do.Injector) (api.SecurityHandler, error) {
	logger, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke logger: %w", err)
	}
	verifier, err := do.InvokeAs[*crypto.TokenCodec](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke TokenCodec: %w", err)
	}
	return NewSecurityHandler(logger, verifier), nil
}

func newHandler(i do.Injector) (api.Handler, error) {
	log, err := do.Invoke[*slog.Logger](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke logger: %w", err)
	}

	createUser, err := do.Invoke[*userapp.CreateUser](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke CreateUser: %w", err)
	}

	getUser, err := do.Invoke[*userapp.GetUser](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke GetUser: %w", err)
	}

	listUsers, err := do.Invoke[*userapp.ListUsers](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke ListUsers: %w", err)
	}

	updateRole, err := do.Invoke[*userapp.UpdateRole](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke UpdateRole: %w", err)
	}

	deleteUser, err := do.Invoke[*userapp.DeleteUser](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke DeleteUser: %w", err)
	}

	createSpace, err := do.Invoke[*spaceapp.CreateSpace](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke CreateSpace: %w", err)
	}

	getSpace, err := do.Invoke[*spaceapp.GetSpace](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke GetSpace: %w", err)
	}

	listSpaces, err := do.Invoke[*spaceapp.ListSpaces](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke ListSpaces: %w", err)
	}

	updateSpace, err := do.Invoke[*spaceapp.UpdateSpace](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke UpdateSpace: %w", err)
	}

	deleteSpace, err := do.Invoke[*spaceapp.DeleteSpace](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke DeleteSpace: %w", err)
	}

	deactivateSpace, err := do.Invoke[*spaceapp.DeactivateSpace](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke DeactivateSpace: %w", err)
	}

	createBooking, err := do.Invoke[*bookingapp.CreateBooking](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke CreateBooking: %w", err)
	}

	cancelBooking, err := do.Invoke[*bookingapp.CancelBooking](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke CancelBooking: %w", err)
	}

	getBooking, err := do.Invoke[*bookingapp.GetBooking](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke GetBooking: %w", err)
	}

	listBySpaceDate, err := do.Invoke[*bookingapp.ListBySpaceAndDate](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke ListBySpaceAndDate: %w", err)
	}

	listByUser, err := do.Invoke[*bookingapp.ListByUser](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke ListByUser: %w", err)
	}

	return NewHandler(
		log,
		createUser, getUser, listUsers, updateRole, deleteUser,
		createSpace, getSpace, listSpaces, updateSpace, deleteSpace, deactivateSpace,
		createBooking, cancelBooking, getBooking, listBySpaceDate, listByUser,
	), nil
}

func newRouter(i do.Injector) (http.Handler, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke config: %w", err)
	}

	h, err := do.InvokeAs[api.Handler](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke handler: %w", err)
	}

	sec, err := do.InvokeAs[api.SecurityHandler](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke security handler: %w", err)
	}

	return NewRouter(h, sec, RouterConfig{
		TrustedProxies:   cfg.TrustedProxies,
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		RequestSizeLimit: cfg.RequestSizeLimit,
	})
}

func newHTTPServer(i do.Injector) (*http.Server, error) {
	cfg, err := do.Invoke[Config](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke config: %w", err)
	}

	handler, err := do.Invoke[http.Handler](i)
	if err != nil {
		return nil, fmt.Errorf("http: invoke router: %w", err)
	}

	return NewServer(
		cfg.HostPort(),
		cfg.ReadTimeout,
		cfg.ReadHeaderTimeout,
		cfg.WriteTimeout,
		cfg.IdleTimeout,
		cfg.MaxHeaderBytes,
		handler,
	), nil
}

var Package = do.Package(
	do.Lazy(newConfig),
	do.Lazy(newSecurityHandler),
	do.Lazy(newHandler),
	do.Lazy(newRouter),
	do.Lazy(newHTTPServer),
)
