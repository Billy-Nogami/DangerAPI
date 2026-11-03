package transport

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"strings"
	"time"
)

const requestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(requestIDHeader)
		if !isValidRequestID(requestID) {
			requestID = newRequestID()
		}

		w.Header().Set(requestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func withAccessLog(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}
		next.ServeHTTP(recorder, r)

		logger.Printf(
			"request_id=%s method=%s path=%s status=%d duration=%s",
			requestIDFromContext(r.Context()),
			r.Method,
			r.URL.Path,
			recorder.Status(),
			time.Since(startedAt).Round(time.Millisecond),
		)
	})
}

func newRequestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return time.Now().UTC().Format("20060102T150405.000000000Z")
	}

	return hex.EncodeToString(bytes[:])
}

func isValidRequestID(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}

	for _, symbol := range value {
		if symbol >= 'a' && symbol <= 'z' ||
			symbol >= 'A' && symbol <= 'Z' ||
			symbol >= '0' && symbol <= '9' ||
			strings.ContainsRune("-_.", symbol) {
			continue
		}
		return false
	}

	return true
}

func requestIDFromContext(ctx context.Context) string {
	requestID, _ := ctx.Value(requestIDContextKey{}).(string)
	return requestID
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(body []byte) (int, error) {
	if r.status == 0 {
		r.WriteHeader(http.StatusOK)
	}
	return r.ResponseWriter.Write(body)
}

func (r *statusRecorder) Status() int {
	if r.status == 0 {
		return http.StatusOK
	}

	return r.status
}
