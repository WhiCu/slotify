package http

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	slogchi "github.com/samber/slog-chi"
	api "github.com/whicu/slotify/api/http"
)

func NewServer(
	addr string,
	readTimeout time.Duration,
	readHeaderTimeout time.Duration,
	writeTimeout time.Duration,
	idleTimeout time.Duration,
	maxHeaderBytes int,
	handler http.Handler,
) *http.Server {
	return &http.Server{
		Addr:              addr,
		ReadTimeout:       readTimeout,
		ReadHeaderTimeout: readHeaderTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		Handler:           handler,
	}
}

type RouterConfig struct {
	TrustedProxies   []string
	AllowedOrigins   []string
	RequestSizeLimit int64
	MetricsPath      string
}

func NewRouter(
	log *slog.Logger,
	h api.Handler,
	secHandler api.SecurityHandler,
	config RouterConfig,
) (http.Handler, error) {
	ogenServer, err := api.NewServer(
		h,
		secHandler,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init ogen server: %w", err)
	}

	r := chi.NewRouter()

	r.Handle(config.MetricsPath, promhttp.Handler())

	r.Group(func(r chi.Router) {
		r.Use(
			middleware.Recoverer,
			middleware.RequestID,
			slogchi.New(log),
			middleware.RequestSize(config.RequestSizeLimit),
		)

		r.Use(cors.Handler(cors.Options{
			AllowedOrigins: config.AllowedOrigins,
			AllowedMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
			},
			AllowedHeaders: []string{
				"Accept",
				"Authorization",
				"Content-Type",
			},
			AllowCredentials: false,
			MaxAge:           300,
		}))

		r.Mount("/", ogenServer)
	})

	return r, nil
}
