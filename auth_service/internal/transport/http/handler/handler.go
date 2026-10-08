package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"auth_service/internal/domain/user"
	"auth_service/internal/transport/http/dto"
	"auth_service/internal/transport/http/middleware"
)

// Handler holds the HTTP handlers for the auth service.
type Handler struct {
	svc user.Service
}

// New builds a Handler around the given user service.
func New(svc user.Service) *Handler {
	return &Handler{svc: svc}
}

// Register handles POST /api/v1/auth/register.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	name, err := user.NewName(req.Name)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid name")
		return
	}
	email, err := user.NewEmail(req.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}

	if err := h.svc.Register(r.Context(), name, email, req.Password); err != nil {
		switch {
		case errors.Is(err, user.ErrEmailAlreadyExists):
			writeError(w, http.StatusConflict, "email already registered")
		case errors.Is(err, user.ErrWeakPassword):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "could not register user")
		}
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "user registered"})
}

// Login handles POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	email, err := user.NewEmail(req.Email)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid email")
		return
	}

	id, token, err := h.svc.Login(r.Context(), email, req.Password)
	if err != nil {
		if errors.Is(err, user.ErrInvalidCredentials) {
			writeError(w, http.StatusUnauthorized, "invalid email or password")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not login")
		return
	}

	writeJSON(w, http.StatusOK, dto.AuthResponse{ID: id, Token: token})
}

// Me handles GET /api/v1/auth/me for authenticated users.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	rawID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	id, err := user.NewUserID(rawID)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	u, err := h.svc.Profile(r.Context(), id)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, "user not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "could not load user")
		return
	}

	writeJSON(w, http.StatusOK, dto.UserResponse{
		ID:        u.ID().String(),
		Name:      u.Name().String(),
		Email:     u.Email().String(),
		CreatedAt: u.CreatedAt(),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, dto.ErrorResponse{Error: message})
}
