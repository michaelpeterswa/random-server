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
			"Event postback is disabled due to Facebook re-engagement being enabled",
			`PostbackTypeNotSupported: Integration doesn't support this Postback type ("session"). Please contact your Kochava Account Management team.`,
			"",
			`{"error":{"message":"(#100) At least one of the parameter 'attribution', 'advertiser_id', 'anon_id', 'page_scoped_user_id', 'user_id_type' or 'ud' is required for the 'custom_app_e`,
			`{"error":{"message":"(#4) Application request limit reached","type":"OAuthException","is_transient":true,"code":4,"`,
			`{"status":400,"error":"Bad Request","errors":[{"codes":["typeMismatch.postBackBean.ctawindow","typeMismatch.ctawindow","typeMismatch.java`,
			"Empty device id.",
			"Error: Doubleclick only supports adid and idfa identifiers, neither found.",
			`{"num_events_processed":1,"num_events_received":1,"events":[{"status":"processed","error_message":"","warning_message":""}]}`,
			"Error: Is Not Allowed Network, Apple Ads",
			`PostbackTypeNotSupported: Integration doesn't support this Postback type ("click"). Please contact your Kochava Account Management team.`,
			"Missing delivery URL",
			`"EventTypeNotSupported: Integration doesn't support the Event type: "". Please contact your Kochava Account Management team."`,
			"Error: Invalid postback type: event",
			`{"errors":["no IDFA, GAID, or GUM data found"],"warnings":[]}`,
			`{"statusCode":422,"success":false}`,
			"Error: getaddrinfo ENOTFOUND odm-postback.pinsightmedia.com odm-postback.pinsightmedia.com:443",
			`{"errors":["The request doesn't contain any event"],"warnings":[]}`,
			"user_id Decryption Failed",
		}

		randomDetail := errorDetails[rand.Intn(len(errorDetails))]

		w.WriteHeader(randomStatusCode)
		_, _ = w.Write([]byte(randomDetail))

		// rfc9457.NewRFC9457(
		// 	rfc9457.WithStatus(randomStatusCode),
		// 	rfc9457.WithDetail(randomDetail),
		// 	rfc9457.WithTitle("server error"),
		// 	rfc9457.WithInstance(r.URL.Path),
		// ).ServeHTTP(w, r)
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
