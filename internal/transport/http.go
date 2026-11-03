package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	"threat-service/internal/domain/threat"
	"threat-service/internal/repository"
)

const maxRequestBodySize = 1 << 20

// ThreatService описывает сценарии, которые нужны HTTP-обработчикам.
type ThreatService interface {
	Create(ctx context.Context, params threat.CreateParams) (threat.Threat, error)
	GetByMD5(ctx context.Context, md5 threat.MD5) (threat.Threat, error)
	Update(ctx context.Context, md5 threat.MD5, params threat.UpdateParams) (threat.Threat, error)
}

// NewHTTPHandler собирает HTTP-маршруты сервиса угроз.
func NewHTTPHandler(service ThreatService) http.Handler {
	handler := &handler{service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /info", handler.createThreat)
	mux.HandleFunc("GET /info/{md5}", handler.getThreat)
	mux.HandleFunc("PUT /info/{md5}", handler.updateThreat)
	return mux
}

type handler struct {
	service ThreatService
}

type createThreatRequest struct {
	MD5         string          `json:"md5"`
	Name        string          `json:"name"`
	Type        threat.Type     `json:"type"`
	Severity    threat.Severity `json:"severity"`
	Description string          `json:"description"`
	Categories  []string        `json:"categories"`
	FirstSeenAt *time.Time      `json:"firstSeenAt"`
}

type updateThreatRequest struct {
	Name        string          `json:"name"`
	Type        threat.Type     `json:"type"`
	Severity    threat.Severity `json:"severity"`
	Description string          `json:"description"`
	Categories  []string        `json:"categories"`
	FirstSeenAt *time.Time      `json:"firstSeenAt"`
}

type threatResponse struct {
	MD5         string          `json:"md5"`
	Name        string          `json:"name"`
	Type        threat.Type     `json:"type"`
	Severity    threat.Severity `json:"severity"`
	Description string          `json:"description"`
	Categories  []string        `json:"categories"`
	FirstSeenAt *time.Time      `json:"firstSeenAt"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type errorResponse struct {
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (h *handler) createThreat(w http.ResponseWriter, r *http.Request) {
	var request createThreatRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON", err.Error())
		return
	}

	entity, err := h.service.Create(r.Context(), threat.CreateParams{
		MD5:         threat.MD5(request.MD5),
		Name:        request.Name,
		Type:        request.Type,
		Severity:    request.Severity,
		Description: request.Description,
		Categories:  request.Categories,
		FirstSeenAt: request.FirstSeenAt,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, threatResponseFromDomain(entity))
}

func (h *handler) getThreat(w http.ResponseWriter, r *http.Request) {
	entity, err := h.service.GetByMD5(r.Context(), threat.MD5(r.PathValue("md5")))
	if err != nil {
		writeServiceError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, threatResponseFromDomain(entity))
}

func (h *handler) updateThreat(w http.ResponseWriter, r *http.Request) {
	var request updateThreatRequest
	if err := decodeJSON(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "Некорректный JSON", err.Error())
		return
	}

	entity, err := h.service.Update(r.Context(), threat.MD5(r.PathValue("md5")), threat.UpdateParams{
		Name:        request.Name,
		Type:        request.Type,
		Severity:    request.Severity,
		Description: request.Description,
		Categories:  request.Categories,
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
	writeJSON(w, status, errorResponse{Message: message, Details: details})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return
	}
}

func threatResponseFromDomain(entity threat.Threat) threatResponse {
	categories := entity.Categories
	if categories == nil {
		categories = []string{}
	}

	return threatResponse{
		MD5:         string(entity.MD5),
		Name:        entity.Name,
		Type:        entity.Type,
		Severity:    entity.Severity,
		Description: entity.Description,
		Categories:  categories,
		FirstSeenAt: entity.FirstSeenAt,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}
}
