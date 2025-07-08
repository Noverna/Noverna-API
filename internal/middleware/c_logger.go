package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"noverna.de/m/v2/internal/logger"
)

type LoggerConfig struct {
	Logger           *logger.Logger
	SkipPaths        []string
	LogRequestBody   bool
	LogResponseBody  bool
	LogHeaders       bool
	MaxBodySize      int64
	RedactHeaders    []string
	RedactBodyFields []string
}

func DefaultLoggerConfig(l *logger.Logger) *LoggerConfig {
	return &LoggerConfig{
		Logger:           l,
		SkipPaths:        []string{"/health", "/metrics"},
		LogRequestBody:   false,
		LogResponseBody:  false,
		LogHeaders:       true,
		MaxBodySize:      1024 * 1024, // 1MB
		RedactHeaders:    []string{"Authorization", "Cookie", "X-Api-Key"},
		RedactBodyFields: []string{"password", "token", "secret"},
	}
}

type LogEntry struct {
	RequestID    string
	Method       string
	URL          string
	RemoteAddr   string
	UserAgent    string
	Referer      string
	RequestBody  string
	ResponseBody string
	StatusCode   int
	Duration     time.Duration
	Size         int64
	Headers      map[string]string
	Error        error
}

type responseWriter struct {
	http.ResponseWriter
	body       *bytes.Buffer
	statusCode int
	size       int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(data []byte) (int, error) {
	if rw.body != nil {
		rw.body.Write(data)
	}
	n, err := rw.ResponseWriter.Write(data)
	rw.size += int64(n)
	return n, err
}

func LoggerMiddleware(config *LoggerConfig) func(next http.Handler) http.Handler {
	if config == nil {
		config = DefaultLoggerConfig(logger.NewLogger())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			if shouldSkipPath(r.URL.Path, config.SkipPaths) {
				next.ServeHTTP(w, r)
				return
			}

			requestID := middleware.GetReqID(r.Context())
			if requestID == "" {
				requestID = generateRequestID()
			}

			entry := &LogEntry{
				RequestID:  requestID,
				Method:     r.Method,
				URL:        r.URL.String(),
				RemoteAddr: r.RemoteAddr,
				UserAgent:  r.UserAgent(),
				Referer:    r.Referer(),
				Headers:    make(map[string]string),
			}

			if config.LogRequestBody {
				body, err := readAndRestoreBody(r, config.MaxBodySize)
				if err != nil {
					config.Logger.Error("Failed to read request body", map[string]interface{}{
						"request_id": requestID,
						"error":      err.Error(),
					})
				} else {
					entry.RequestBody = redactSensitiveData(body, config.RedactBodyFields)
				}
			}

			if config.LogHeaders {
				for name, values := range r.Header {
					if len(values) > 0 {
						if shouldRedactHeader(name, config.RedactHeaders) {
							entry.Headers[name] = "[REDACTED]"
						} else {
							entry.Headers[name] = values[0]
						}
					}
				}
			}

			var rw *responseWriter
			if config.LogResponseBody {
				rw = &responseWriter{
					ResponseWriter: w,
					body:           &bytes.Buffer{},
					statusCode:     http.StatusOK,
				}
			} else {
				rw = &responseWriter{
					ResponseWriter: w,
					statusCode:     http.StatusOK,
				}
			}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)
			entry.Duration = duration
			entry.StatusCode = rw.statusCode
			entry.Size = rw.size

			if config.LogResponseBody && rw.body != nil {
				responseBody := rw.body.String()
				if len(responseBody) > int(config.MaxBodySize) {
					responseBody = responseBody[:config.MaxBodySize] + "... [TRUNCATED]"
				}
				entry.ResponseBody = redactSensitiveData(responseBody, config.RedactBodyFields)
			}

			logRequest(config.Logger, entry)
		})
	}
}

func logRequest(l *logger.Logger, entry *LogEntry) {
	fields := map[string]interface{}{
		"request_id":   entry.RequestID,
		"method":       entry.Method,
		"url":          entry.URL,
		"remote_addr":  entry.RemoteAddr,
		"user_agent":   entry.UserAgent,
		"status_code":  entry.StatusCode,
		"duration_ms":  entry.Duration.Milliseconds(),
		"size_bytes":   entry.Size,
	}

	if entry.Referer != "" {
		fields["referer"] = entry.Referer
	}
	if len(entry.Headers) > 0 {
		fields["headers"] = entry.Headers
	}
	if entry.RequestBody != "" {
		fields["request_body"] = entry.RequestBody
	}
	if entry.ResponseBody != "" {
		fields["response_body"] = entry.ResponseBody
	}

	message := fmt.Sprintf("%s %s %d", entry.Method, entry.URL, entry.StatusCode)

	switch {
	case entry.StatusCode >= 500:
		l.Error(message, fields)
	case entry.StatusCode >= 400:
		l.Warn(message, fields)
	case entry.StatusCode >= 300:
		l.Info(message, fields)
	default:
		l.Info(message, fields)
	}
}


func shouldSkipPath(path string, skipPaths []string) bool {
	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

func shouldRedactHeader(header string, redactHeaders []string) bool {
	for _, redactHeader := range redactHeaders {
		if header == redactHeader {
			return true
		}
	}
	return false
}

func readAndRestoreBody(r *http.Request, maxSize int64) (string, error) {
	if r.Body == nil {
		return "", nil
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxSize))
	if err != nil {
		return "", err
	}

	r.Body = io.NopCloser(bytes.NewBuffer(body))

	return string(body), nil
}

func redactSensitiveData(data string, sensitiveFields []string) string {
	result := data
    for _, field := range sensitiveFields {
        result = strings.Replace(result, field, "[REDACTED]", -1)
    }
    return result
}

func generateRequestID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func SimpleLoggerMiddleware(l *logger.Logger) func(next http.Handler) http.Handler {
	config := &LoggerConfig{
		Logger:          l,
		SkipPaths:       []string{"/health"},
		LogRequestBody:  false,
		LogResponseBody: false,
		LogHeaders:      false,
		MaxBodySize:     1024,
	}
	return LoggerMiddleware(config)
}

func DetailedLoggerMiddleware(l *logger.Logger) func(next http.Handler) http.Handler {
	config := &LoggerConfig{
		Logger:           l,
		SkipPaths:        []string{},
		LogRequestBody:   true,
		LogResponseBody:  true,
		LogHeaders:       true,
		MaxBodySize:      1024 * 10, // 10KB
		RedactHeaders:    []string{"Authorization", "Cookie"},
		RedactBodyFields: []string{"password", "token"},
	}
	return LoggerMiddleware(config)
}

func SecurityAwareLoggerMiddleware(l *logger.Logger) func(next http.Handler) http.Handler {
	config := &LoggerConfig{
		Logger:           l,
		SkipPaths:        []string{"/health", "/metrics"},
		LogRequestBody:   false,
		LogResponseBody:  false,
		LogHeaders:       true,
		MaxBodySize:      512,
		RedactHeaders:    []string{"Authorization", "Cookie", "X-Api-Key", "X-Auth-Token"},
		RedactBodyFields: []string{"password", "token", "secret", "key", "auth"},
	}
	return LoggerMiddleware(config)
}