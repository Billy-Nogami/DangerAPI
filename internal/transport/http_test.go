package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"threat-service/internal/domain/threat"
	"threat-service/internal/repository"
	"threat-service/internal/transport/generated"
)

func TestCreateThreat(t *testing.T) {
	t.Parallel()

	entity := testThreat()
	service := &threatServiceStub{
		create: func(_ context.Context, params threat.CreateParams) (threat.Threat, error) {
			if params.MD5 != entity.MD5 || params.Type != threat.TypeTrojan {
				t.Fatalf("переданы неверные параметры: %+v", params)
			}
			return entity, nil
		},
	}

	request := httptest.NewRequest(
		http.MethodPost,
		"/info",
		strings.NewReader(`{"md5":"d41d8cd98f00b204e9800998ecf8427e","name":"Trojan.Win32.Generic","type":"trojan","severity":"high","categories":["banker"]}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newTestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("статус = %d, нужен %d", response.Code, http.StatusCreated)
	}
	if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		t.Fatalf("Content-Type = %q", contentType)
	}

	var body generated.ThreatResponse
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("не удалось разобрать ответ: %v", err)
	}
	if body.Md5 != string(entity.MD5) || body.Name != entity.Name {
		t.Fatalf("тело ответа = %+v", body)
	}
}

func TestServiceErrorsAreConvertedToHTTPStatus(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		method     string
		path       string
		body       string
		service    *threatServiceStub
		wantStatus int
	}{
		{
			name:   "некорректный md5",
			method: http.MethodGet,
			path:   "/info/invalid",
			service: &threatServiceStub{
				get: func(context.Context, threat.MD5) (threat.Threat, error) {
					return threat.Threat{}, &threat.ValidationError{Field: "md5", Message: "некорректный формат"}
				},
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "угроза не найдена",
			method: http.MethodGet,
			path:   "/info/d41d8cd98f00b204e9800998ecf8427e",
			service: &threatServiceStub{
				get: func(context.Context, threat.MD5) (threat.Threat, error) {
					return threat.Threat{}, repository.ErrNotFound
				},
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "дублирующийся md5",
			method: http.MethodPost,
			path:   "/info",
			body:   `{"md5":"d41d8cd98f00b204e9800998ecf8427e","name":"Trojan.Win32.Generic","type":"trojan","severity":"high"}`,
			service: &threatServiceStub{
				create: func(context.Context, threat.CreateParams) (threat.Threat, error) {
					return threat.Threat{}, repository.ErrAlreadyExists
				},
			},
			wantStatus: http.StatusConflict,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(testCase.method, testCase.path, strings.NewReader(testCase.body))
			if testCase.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
			response := httptest.NewRecorder()

			newTestHandler(testCase.service).ServeHTTP(response, request)

			if response.Code != testCase.wantStatus {
				t.Fatalf("статус = %d, нужен %d", response.Code, testCase.wantStatus)
			}
			var body generated.ErrorResponse
			if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
				t.Fatalf("не удалось разобрать ошибку: %v", err)
			}
			if body.Message == "" {
				t.Fatal("в ответе отсутствует message")
			}
		})
	}
}

func TestCreateThreatRejectsUnknownJSONFields(t *testing.T) {
	t.Parallel()

	service := &threatServiceStub{
		create: func(context.Context, threat.CreateParams) (threat.Threat, error) {
			return threat.Threat{}, errors.New("сервис не должен быть вызван")
		},
	}
	request := httptest.NewRequest(
		http.MethodPost,
		"/info",
		strings.NewReader(`{"md5":"d41d8cd98f00b204e9800998ecf8427e","name":"Trojan.Win32.Generic","extra":true}`),
	)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	newTestHandler(service).ServeHTTP(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("статус = %d, нужен %d", response.Code, http.StatusBadRequest)
	}
}

func TestHealth(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		healthCheck healthCheckerFunc
		wantStatus  int
	}{
		{
			name: "хранилище доступно",
			healthCheck: func(context.Context) error {
				return nil
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "хранилище недоступно",
			healthCheck: func(context.Context) error {
				return errors.New("соединение с PostgreSQL отсутствует")
			},
			wantStatus: http.StatusServiceUnavailable,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			response := httptest.NewRecorder()

			NewHTTPHandler(newTestLogger(), &threatServiceStub{}, testCase.healthCheck, NewRateLimiter(1, time.Minute)).ServeHTTP(response, request)

			if response.Code != testCase.wantStatus {
				t.Fatalf("статус = %d, нужен %d", response.Code, testCase.wantStatus)
			}
		})
	}
}

func TestDocumentationEndpoints(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(&threatServiceStub{})
	testCases := []struct {
		name        string
		path        string
		contentType string
		contains    string
	}{
		{
			name:        "спецификация OpenAPI",
			path:        "/openapi.yaml",
			contentType: "application/yaml",
			contains:    "openapi: 3.0.3",
		},
		{
			name:        "интерфейс Swagger",
			path:        "/swagger/",
			contentType: "text/html",
			contains:    "Swagger UI",
		},
		{
			name:        "настройка интерфейса Swagger",
			path:        "/swagger/swagger-initializer.js",
			contentType: "application/javascript",
			contains:    "url: \"/openapi.yaml\"",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, testCase.path, nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("статус = %d, нужен %d", response.Code, http.StatusOK)
			}
			if contentType := response.Header().Get("Content-Type"); !strings.HasPrefix(contentType, testCase.contentType) {
				t.Fatalf("Content-Type = %q, нужен префикс %q", contentType, testCase.contentType)
			}
			if !strings.Contains(response.Body.String(), testCase.contains) {
				t.Fatalf("тело ответа не содержит %q", testCase.contains)
			}
		})
	}
}

func TestRequestID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name          string
		requestID     string
		expectedValue string
	}{
		{
			name:      "генерируется при отсутствии заголовка",
			requestID: "",
		},
		{
			name:          "сохраняется корректный заголовок клиента",
			requestID:     "trace-42",
			expectedValue: "trace-42",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			if testCase.requestID != "" {
				request.Header.Set(requestIDHeader, testCase.requestID)
			}
			response := httptest.NewRecorder()

			newTestHandler(&threatServiceStub{}).ServeHTTP(response, request)

			actual := response.Header().Get(requestIDHeader)
			if actual == "" {
				t.Fatal("в ответе отсутствует X-Request-ID")
			}
			if testCase.expectedValue != "" && actual != testCase.expectedValue {
				t.Fatalf("X-Request-ID = %q, нужен %q", actual, testCase.expectedValue)
			}
		})
	}
}

func TestAccessLogContainsRequestData(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&output, nil))
	handler := NewHTTPHandler(logger, &threatServiceStub{}, healthCheckerFunc(func(context.Context) error {
		return nil
	}), NewRateLimiter(100, time.Minute))
	request := httptest.NewRequest(http.MethodGet, "/health", nil)
	request.RemoteAddr = "192.0.2.10:12345"
	request.Header.Set(requestIDHeader, "trace-42")

	handler.ServeHTTP(httptest.NewRecorder(), request)

	logLine := output.String()
	for _, value := range []string{
		"msg=\"обработан HTTP-запрос\"",
		"request_id=trace-42",
		"method=GET",
		"path=/health",
		"status=200",
		"client_ip=192.0.2.10",
	} {
		if !strings.Contains(logLine, value) {
			t.Fatalf("журнал не содержит %q: %s", value, logLine)
		}
	}
}

func TestRateLimit(t *testing.T) {
	t.Parallel()

	service := &threatServiceStub{
		get: func(context.Context, threat.MD5) (threat.Threat, error) {
			return testThreat(), nil
		},
	}
	handler := NewHTTPHandler(newTestLogger(), service, healthCheckerFunc(func(context.Context) error {
		return nil
	}), NewRateLimiter(1, time.Minute))

	for attempt, expectedStatus := range []int{http.StatusOK, http.StatusTooManyRequests} {
		request := httptest.NewRequest(http.MethodGet, "/info/d41d8cd98f00b204e9800998ecf8427e", nil)
		request.RemoteAddr = "192.0.2.10:12345"
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		if response.Code != expectedStatus {
			t.Fatalf("запрос %d: статус = %d, нужен %d", attempt+1, response.Code, expectedStatus)
		}
	}
}

type threatServiceStub struct {
	create func(context.Context, threat.CreateParams) (threat.Threat, error)
	get    func(context.Context, threat.MD5) (threat.Threat, error)
	update func(context.Context, threat.MD5, threat.UpdateParams) (threat.Threat, error)
}

type healthCheckerFunc func(context.Context) error

func (f healthCheckerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

func newTestHandler(service *threatServiceStub) http.Handler {
	return NewHTTPHandler(newTestLogger(), service, healthCheckerFunc(func(context.Context) error {
		return nil
	}), NewRateLimiter(100, time.Minute))
}

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func (s *threatServiceStub) Create(ctx context.Context, params threat.CreateParams) (threat.Threat, error) {
	if s.create == nil {
		return threat.Threat{}, errors.New("сценарий создания не настроен")
	}
	return s.create(ctx, params)
}

func (s *threatServiceStub) GetByMD5(ctx context.Context, md5 threat.MD5) (threat.Threat, error) {
	if s.get == nil {
		return threat.Threat{}, errors.New("сценарий поиска не настроен")
	}
	return s.get(ctx, md5)
}

func (s *threatServiceStub) Update(ctx context.Context, md5 threat.MD5, params threat.UpdateParams) (threat.Threat, error) {
	if s.update == nil {
		return threat.Threat{}, errors.New("сценарий обновления не настроен")
	}
	return s.update(ctx, md5, params)
}

func testThreat() threat.Threat {
	now := time.Date(2026, time.November, 3, 10, 0, 0, 0, time.UTC)
	return threat.Threat{
		MD5:        "d41d8cd98f00b204e9800998ecf8427e",
		Name:       "Trojan.Win32.Generic",
		Type:       threat.TypeTrojan,
		Severity:   threat.SeverityHigh,
		Categories: []string{"banker"},
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}
