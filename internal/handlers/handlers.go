package handlers

import (
	"encoding/json"
	"log/slog"
	"math/rand"
	"net/http"
	"time"

	"alpineworks.io/rfc9457"
	"github.com/michaelpeterswa/random-server/internal/config"
)

type HandlersClient struct {
	c *config.Config
}

func NewHandlersClient(c *config.Config) *HandlersClient {
	return &HandlersClient{c: c}
}

type SuccessResponse struct {
	Message string `json:"message"`
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// LoggingMiddleware logs HTTP requests
func (h *HandlersClient) LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap the ResponseWriter to capture status code
		wrapped := &responseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK, // default status
		}

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)

		slog.Info("HTTP request",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("query", r.URL.RawQuery),
			slog.Int("status", wrapped.statusCode),
			slog.Duration("duration", duration),
			slog.String("user_agent", r.UserAgent()),
			slog.String("remote_addr", r.RemoteAddr),
		)
	})
}

func (h *HandlersClient) CatchAllHandler(w http.ResponseWriter, r *http.Request) {
	// Generate random error based on RandomErrorRate
	if rand.Float64() < h.c.RandomErrorRate {
		// Generate random error status code (400-599)
		errorCodes := []int{
			http.StatusBadRequest,
			http.StatusInternalServerError,
			http.StatusBadGateway,
			http.StatusServiceUnavailable,
			http.StatusGatewayTimeout,
		}
		randomStatusCode := errorCodes[rand.Intn(len(errorCodes))]

		errorDetails := []string{
			"failed to process request",
			"uh oh, something went wrong",
			"unexpected error occurred",
			"something went wrong",
			"internal server error",
		}

		randomDetail := errorDetails[rand.Intn(len(errorDetails))]

		rfc9457.NewRFC9457(
			rfc9457.WithStatus(randomStatusCode),
			rfc9457.WithDetail(randomDetail),
			rfc9457.WithTitle("server error"),
			rfc9457.WithInstance(r.URL.Path),
		).ServeHTTP(w, r)
		return
	}

	var response = &SuccessResponse{
		Message: "request processed successfully",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		rfc9457.NewRFC9457(
			rfc9457.WithStatus(http.StatusInternalServerError),
			rfc9457.WithDetail(err.Error()),
			rfc9457.WithTitle("json encoding error"),
			rfc9457.WithInstance(r.URL.Path),
		).ServeHTTP(w, r)
	}

}
