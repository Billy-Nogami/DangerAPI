package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"time"

	"threat-service/internal/domain/threat"
	"threat-service/internal/repository"
	"threat-service/internal/transport/generated"
)

const maxRequestBodySize = 1 << 20

// ThreatService описывает сценарии, которые нужны HTTP-обработчикам.
type ThreatService interface {
	Create(ctx context.Context, params threat.CreateParams) (threat.Threat, error)
	GetByMD5(ctx context.Context, md5 threat.MD5) (threat.Threat, error)
	Update(ctx context.Context, md5 threat.MD5, params threat.UpdateParams) (threat.Threat, error)
}

// HealthChecker проверяет доступность зависимостей, нужных для работы сервиса.
type HealthChecker interface {
	Ping(ctx context.Context) error
}

// NewHTTPHandler собирает HTTP-маршруты сервиса угроз.
func NewHTTPHandler(service ThreatService, healthChecker HealthChecker, rateLimiter *RateLimiter) http.Handler {
	handler := &handler{service: service, healthChecker: healthChecker}
	router := generated.HandlerWithOptions(handler, generated.StdHTTPServerOptions{
		Middlewares: []generated.MiddlewareFunc{rateLimiter.Limit},
	})
	return withRequestID(withAccessLog(log.Default(), router))
}

type handler struct {
	service       ThreatService
	healthChecker HealthChecker
}

var _ generated.ServerInterface = (*handler)(nil)

func (h *handler) GetHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.healthChecker.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "Сервис временно недоступен", "Не удалось подключиться к хранилищу")
		return
	}

	writeJSON(w, http.StatusOK, generated.HealthResponse{Status: generated.Ok})
}

func (h *handler) CreateThreat(w http.ResponseWriter, r *http.Request) {
	var request generated.ThreatCreateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON", err.Error())
		return
	}

	entity, err := h.service.Create(r.Context(), threat.CreateParams{
		MD5:         threat.MD5(request.Md5),
		Name:        request.Name,
		Type:        threat.Type(request.Type),
		Severity:    threat.Severity(request.Severity),
		Description: valueOrEmpty(request.Description),
		Categories:  valueOrNil(request.Categories),
		FirstSeenAt: request.FirstSeenAt,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, threatResponseFromDomain(entity))
}

func (h *handler) GetInfoByMd5(w http.ResponseWriter, r *http.Request, md5 generated.ThreatMD5) {
	entity, err := h.service.GetByMD5(r.Context(), threat.MD5(md5))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, threatResponseFromDomain(entity))
}

func (h *handler) UpdateThreat(w http.ResponseWriter, r *http.Request, md5 generated.ThreatMD5) {
	var request generated.ThreatUpdateRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON", err.Error())
		return
	}

	entity, err := h.service.Update(r.Context(), threat.MD5(md5), threat.UpdateParams{
		Name:        request.Name,
		Type:        threat.Type(request.Type),
		Severity:    threat.Severity(request.Severity),
		Description: valueOrEmpty(request.Description),
		Categories:  valueOrNil(request.Categories),
		FirstSeenAt: request.FirstSeenAt,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, threatResponseFromDomain(entity))
}

func decodeJSON(w http.ResponseWriter, r *http.Request, destination any) error {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		return fmt.Errorf("ожидается заголовок Content-Type: application/json")
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodySize)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("тело запроса должно содержать один JSON-объект")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("тело запроса должно содержать только один JSON-объект")
	}

	return nil
}

func writeServiceError(w http.ResponseWriter, err error) {
	var validationError *threat.ValidationError
	switch {
	case errors.As(err, &validationError):
		writeError(w, http.StatusBadRequest, "Некорректные данные запроса", validationError.Error())
	case errors.Is(err, repository.ErrNotFound):
		writeError(w, http.StatusNotFound, "Угроза не найдена", "")
	case errors.Is(err, repository.ErrAlreadyExists):
		writeError(w, http.StatusConflict, "Угроза с таким MD5 уже существует", "")
	default:
		writeError(w, http.StatusInternalServerError, "Внутренняя ошибка сервера", "")
	}
}

func writeError(w http.ResponseWriter, status int, message, details string) {
	response := generated.ErrorResponse{Message: message}
	if details != "" {
		response.Details = &details
	}
	writeJSON(w, status, response)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func threatResponseFromDomain(entity threat.Threat) generated.ThreatResponse {
	categories := entity.Categories
	if categories == nil {
		categories = []string{}
	}

	return generated.ThreatResponse{
		Md5:         string(entity.MD5),
		Name:        entity.Name,
		Type:        generated.ThreatType(entity.Type),
		Severity:    generated.ThreatSeverity(entity.Severity),
		Description: entity.Description,
		Categories:  categories,
		FirstSeenAt: entity.FirstSeenAt,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}

func valueOrNil(value *[]string) []string {
	if value == nil {
		return nil
	}

	return *value
}
