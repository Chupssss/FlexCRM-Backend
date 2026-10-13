package clients

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

var demoCompanyID = uuid.MustParse("11111111-1111-1111-1111-111111111111")

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("clients: response: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]string{"code": code, "message": message})
}

func handleError(w http.ResponseWriter, err error) {
	var dbError *pgconn.PgError
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, "NOT_FOUND", "Запись не найдена")
	case errors.Is(err, ErrInvalid):
		writeError(w, http.StatusBadRequest, "INVALID_DATA", "Проверьте обязательные поля и связанные записи")
	case errors.Is(err, ErrConflict), errors.As(err, &dbError) && dbError.Code == "23503":
		writeError(w, http.StatusConflict, "CLIENT_HAS_LINKED_RECORDS", "У клиента есть сделки или задачи. Сначала перенесите их к другому клиенту")
	default:
		log.Printf("clients: %v", err)
		writeError(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Не удалось выполнить запрос")
	}
}

func readJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Проверьте JSON и названия полей")
		return false
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "В запросе должен быть один JSON-объект")
		return false
	}
	return true
}

func pathID(w http.ResponseWriter, r *http.Request, key string) (uuid.UUID, bool) {
	id, err := uuid.Parse(r.PathValue(key))
	if err != nil || id == uuid.Nil {
		code := "INVALID_CLIENT_ID"
		if key == "contact_id" {
			code = "INVALID_CONTACT_ID"
		}
		if key == "comment_id" {
			code = "INVALID_COMMENT_ID"
		}
		writeError(w, http.StatusBadRequest, code, "Идентификатор должен быть UUID")
		return uuid.Nil, false
	}
	return id, true
}

func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request) {
	filter := Filter{Search: strings.TrimSpace(r.URL.Query().Get("q")), Type: r.URL.Query().Get("client_type")}
	if value := r.URL.Query().Get("responsible_user_id"); value != "" {
		id, err := uuid.Parse(value)
		if err != nil || id == uuid.Nil {
			writeError(w, http.StatusBadRequest, "INVALID_USER_ID", "Ответственный должен иметь UUID")
			return
		}
		filter.ResponsibleID = &id
	}
	data, err := h.service.GetAll(r.Context(), demoCompanyID, filter)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	data, err := h.service.GetByID(r.Context(), demoCompanyID, id)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "CLIENT_NOT_FOUND", "Клиент не найден")
		return
	}
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var input ClientInput
	if !readJSON(w, r, &input) {
		return
	}
	data, err := h.service.Create(r.Context(), demoCompanyID, input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, data)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var input ClientInput
	if !readJSON(w, r, &input) {
		return
	}
	data, err := h.service.Update(r.Context(), demoCompanyID, id, input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(r.Context(), demoCompanyID, id); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetUsers(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.GetUsers(r.Context(), demoCompanyID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) GetContacts(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	data, err := h.service.GetContacts(r.Context(), demoCompanyID, id)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) SaveContact(w http.ResponseWriter, r *http.Request) {
	clientID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	id := uuid.Nil
	status := http.StatusCreated
	if r.Method == http.MethodPut {
		id, ok = pathID(w, r, "contact_id")
		if !ok {
			return
		}
		status = http.StatusOK
	}
	var input ContactInput
	if !readJSON(w, r, &input) {
		return
	}
	data, err := h.service.SaveContact(r.Context(), demoCompanyID, clientID, id, input)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, status, data)
}

func (h *Handler) DeleteContact(w http.ResponseWriter, r *http.Request) {
	clientID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	id, ok := pathID(w, r, "contact_id")
	if !ok {
		return
	}
	if err := h.service.DeleteContact(r.Context(), demoCompanyID, clientID, id); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetComments(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	data, err := h.service.GetComments(r.Context(), demoCompanyID, id)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) SaveComment(w http.ResponseWriter, r *http.Request) {
	clientID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	id := uuid.Nil
	status := http.StatusCreated
	if r.Method == http.MethodPut {
		id, ok = pathID(w, r, "comment_id")
		if !ok {
			return
		}
		status = http.StatusOK
	}
	var input struct {
		Text string `json:"text"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	data, err := h.service.SaveComment(r.Context(), demoCompanyID, clientID, id, input.Text)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, status, data)
}

func (h *Handler) DeleteComment(w http.ResponseWriter, r *http.Request) {
	clientID, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	id, ok := pathID(w, r, "comment_id")
	if !ok {
		return
	}
	if err := h.service.DeleteComment(r.Context(), demoCompanyID, clientID, id); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetTags(w http.ResponseWriter, r *http.Request) {
	data, err := h.service.GetTags(r.Context(), demoCompanyID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) CreateTag(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	data, err := h.service.CreateTag(r.Context(), demoCompanyID, input.Name)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, data)
}

func (h *Handler) GetClientTags(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	data, err := h.service.GetClientTags(r.Context(), demoCompanyID, id)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (h *Handler) SetTags(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(w, r, "id")
	if !ok {
		return
	}
	var input struct {
		Tags []uuid.UUID `json:"tag_ids"`
	}
	if !readJSON(w, r, &input) {
		return
	}
	if err := h.service.SetTags(r.Context(), demoCompanyID, id, input.Tags); err != nil {
		handleError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
