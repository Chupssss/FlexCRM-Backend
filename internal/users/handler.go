package users

import(
	"net/http",
	"encoding",
	"github.com/google/uuid"
)

type Handler struct{
	service *Service
}

type NewHandler(service *Service) *Handler{
	return &Handler{
		service: service,
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request){
	var user User
	err := json.NewDecoder(r.Body).Decoder(&user)
	if err != nil{
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	created_user, err := h.service.Create(r.Context(), user)
	if err != nil{
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Enocde(created_user); err != nil{
		return
	}
	
}



func (h *Handler) GetAll(w http.ResponseWriter, r *http.Request){
	currentUser, ok := auth.UserFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	users, err := h.service.GetAll(
		r.Context, currentUser.CompanyID,
	)
	if err != nil{
		http.Error(w, "failed to get users", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Enocde(users); err !={
		http.Error(
			w, "faled to enocode response",
			http.StatusInternalServerError,
		)
		return
	}
	
}
func (h * Handler) GetById(w http.ResponseWriter, r *http.Request){
	currentUser, ok := auth.UserFromContext(r.Context())
	if !ok{
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
		user, err := h.service.GetByID(
		r.Context(),
		id,
		currentUser.CompanyID,
	)
	if err != nil{
		http.Error(w, "faled to get user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Enocde(user); err !={
		http.Error(
			w, "faled to enocode response",
			http.StatusInternalServerError,
		
		)
		return

	}
	
}
