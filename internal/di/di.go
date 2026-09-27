package di

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/knadh/koanf/v2"
	"github.com/samber/do/v2"
	"github.com/whicu/slotify/internal/application"
	"github.com/whicu/slotify/internal/config"
	"github.com/whicu/slotify/internal/infrastructure/storage"
	"github.com/whicu/slotify/internal/infrastructure/telemetry"
	"github.com/whicu/slotify/pkg/logger"
)

type Config struct {
	CfgView     bool
	RootInvites struct {
		CountInvites int
		Out          io.Writer
	}
}

func New(ctx context.Context, fsys fs.FS, configPath string) *do.RootScope {
	injector := do.NewWithOpts(&do.InjectorOpts{
		Logf:                     diLogf,
		HealthCheckParallelism:   16,
		HealthCheckGlobalTimeout: 20 * time.Second,
	})

	do.ProvideValue(injector, ctx)
	registerPackages(injector, fsys, configPath)

	return injector
}

func diLogf(format string, args ...any) {
	fmt.Printf("[DI] "+format+"\n", args...)
}

func registerPackages(i do.Injector, fsys fs.FS, configPath string) {
	config.Package(fsys, configPath)(i) // no dependencies
	telemetry.Package(i)                // config
	logger.Package(i)                   // config, telemetry
	storage.Package(i)                  // config, telemetry
	application.Package(i)              // config, webauthnadapter, crypto, storage, telemetry
}

func Build(ctx context.Context, injector *do.RootScope, cfg Config) (*http.Server, error) {
	if cfg.CfgView {
		if err := cfgView(injector); err != nil {
			return nil, fmt.Errorf("cfg view: %w", err)
		}
	}

	if err := initTelemetry(injector); err != nil {
		return nil, err
	}

	// log, err := initLogger(injector)
	// if err != nil {
	// 	return nil, err
	// }

	if err := initStorage(ctx, injector); err != nil {
		return nil, err
	}

	srv, err := do.Invoke[*http.Server](injector)
	if err != nil {
		return nil, fmt.Errorf("build http server: %w", err)
	}

	return srv, nil
}

func Run(ctx context.Context, injector *do.RootScope, cfg Config) error {
	srv, err := Build(ctx, injector, cfg)
	if err != nil {
		return err
	}

	log, err := do.Invoke[*slog.Logger](injector) // тот же singleton, что и в Build
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}

	if errServe := serve(ctx, log, srv); errServe != nil {
		return errServe
	}

	return waitForShutdown(ctx, injector)
}

func serve(ctx context.Context, log *slog.Logger, srv *http.Server) error {
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", srv.Addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", srv.Addr, err)
	}

	go func() {
		log.InfoContext(ctx, "http server listening", slog.String("addr", srv.Addr))
		if serveErr := srv.Serve(ln); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.ErrorContext(ctx, "http server failed", slog.Any("error", serveErr))
		}
	}()

	return nil
}

func initTelemetry(injector do.Injector) error {
	if _, err := do.Invoke[*telemetry.Service](injector); err != nil {
		return fmt.Errorf("init telemetry service: %w", err)
	}
	return nil
}

// func initLogger(injector do.Injector) (*slog.Logger, error) {
// 	log, err := do.Invoke[*slog.Logger](injector)
// 	if err != nil {
// 		return nil, fmt.Errorf("init logger: %w", err)
// 	}
// 	return log, nil
// }

func initStorage(ctx context.Context, injector do.Injector) error {
	srg, err := do.Invoke[*storage.Storage](injector)
	if err != nil {
		return fmt.Errorf("init storage: %w", err)
	}
	if errUp := srg.Up(ctx); errUp != nil {
		return fmt.Errorf("storage up: %w", errUp)
	}
	return nil
}

func waitForShutdown(ctx context.Context, injector *do.RootScope) error {
	_, report := injector.ShutdownOnSignalsWithContext(ctx)
	if errStr := report.Error(); errStr != "" {
		return fmt.Errorf("shutdown: %v", errStr)
	}
	return nil
}

func cfgView(i do.Injector) error {
	k, err := do.Invoke[*koanf.Koanf](i)
	if err != nil {
		return fmt.Errorf("init koanf: %w", err)
	}
	fmt.Println(config.DumpFlat(k))
	return nil
}
