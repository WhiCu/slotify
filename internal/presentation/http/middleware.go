package http

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ogen-go/ogen/middleware"
)

const (
	MaxChaosErrorRate     = 100
	DefaultChaosErrorRate = 5
	DefaultChaosDelay     = 1 * time.Second / 4
)

var (
	errChaosInjected = errors.New("chaos: injected failure")

	rateError atomic.Uint32 // процент ошибок 0..100
	delayNs   atomic.Int64  // time.Duration в наносекундах
)

// errRate - the percentage of requests that should fail (0-MaxChaosErrorRate)
// Delay - the amount of time to Delay the request before responding
func Chaos() middleware.Middleware {
	rateError.Store(DefaultChaosErrorRate)
	delayNs.Store(int64(DefaultChaosDelay))

	return func(req middleware.Request, next middleware.Next) (middleware.Response, error) {
		delay := delayNs.Load()
		rate := rateError.Load()

		if delay > 0 {
			time.Sleep(time.Duration(rand.N(delay)))
		}

		if rate > 0 && rand.N[uint32](100) < rate {
			return middleware.Response{}, errChaosInjected
		}

		res, err := next(req)

		if delay > 0 {
			time.Sleep(time.Duration(rand.N(delay)))
		}

		return res, err
	}
}

func ChaosOptions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	if v := q.Get("rate"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 || n > MaxChaosErrorRate {
			http.Error(w, "rate must be an integer in 0..100", http.StatusBadRequest)
			return
		}
		rateError.Store(uint32(n))
	}

	if v := q.Get("delay"); v != "" {
		d, err := time.ParseDuration(v) // "500ms", "2s", "1m"
		if err != nil || d < 0 {
			http.Error(w, "delay must be a non-negative duration, e.g. 500ms", http.StatusBadRequest)
			return
		}
		delayNs.Store(int64(d))
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Chaos options updated: rate=%d%%, delay=%s", rateError.Load(), time.Duration(delayNs.Load()).String())
}

func ChaosGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Chaos options updated: rate=%d%%, delay=%s", rateError.Load(), time.Duration(delayNs.Load()).String())
}
