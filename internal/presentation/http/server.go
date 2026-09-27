package http

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
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
	h http.Handler,
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

	r.Mount("/", h)

	return r, nil
}
