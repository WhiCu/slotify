package http

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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
}

func NewRouter(
	h api.Handler,
	secHandler api.SecurityHandler,
	config RouterConfig,
) (http.Handler, error) {
	r := chi.NewRouter()

	r.Use(
		middleware.Recoverer,
		middleware.RequestID,
		middleware.Logger,
		middleware.RequestSize(config.RequestSizeLimit),
	)

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: config.AllowedOrigins,
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
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

	ogenServer, err := api.NewServer(
		h,
		secHandler,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to init ogen server: %w", err)
	}

	r.Mount("/", ogenServer)

	return r, nil
}
