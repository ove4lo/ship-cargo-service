package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/ove4lo/ship-cargo-service/internal/model"
	"github.com/ove4lo/ship-cargo-service/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler represents an authorization handler
type AuthHandler struct {
	userRepo *repository.UserRepository
	jwtSecret string
	jwtExp time.Duration
}

// NewAuthHandler creates an instance of AuthHandler
func NewAuthHandler(userRepo *repository.UserRepository, jwtSecret string, jwtExp time.Duration) *AuthHandler {
	return &AuthHandler{
		userRepo: userRepo,
		jwtSecret: jwtSecret,
		jwtExp: jwtExp,
	}
}

// registerRequest represents the registration HTTP request body
type registerRequest struct {
	Name string `json:"name"`
	Email string `json:"email"`
	Password string `json:"password"`
	Role model.Role `json:"role"`
}

// loginRequest represents the login request payload
type loginRequest struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

// tokenResponse represents the authentication token response payload
type tokenResponse struct {
	Token string `json:"token"`
}

// Register handles HTTP requests for user registration
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.Email == "" || req.Password == "" {
		http.Error(w, `{"error":"name, email or password are required"}`, http.StatusBadRequest)
		return
	}

	if req.Role == "" {
		req.Role = model.RoleSupplier
	}

	if req.Role != model.RoleSupplier && req.Role != model.RoleManager {
		http.Error(w, `{"error":"role must be supplier or manager"}`, http.StatusBadRequest)
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	user := &model.User{
		Name: req.Name,
		Email: req.Email,
		PasswordHash: string(hash),
		Role: req.Role,
	}

	if err := h.userRepo.Create(r.Context(), user); err != nil {
		http.Error(w, `{"error":"email already exists"}`, http.StatusConflict)
		return
	}

	token, err := h.generateToken(user)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tokenResponse{Token: token})
}

// Login handles HTTP requests for user authentication
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	user, err := h.userRepo.GetByEmail(r.Context(), req.Email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
			return
		}
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		http.Error(w, `{"error":"invalid credentials"}`, http.StatusUnauthorized)
		return
	}

	token, err := h.generateToken(user)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokenResponse{Token: token})
}

// generateToken creates a signed JWT token containing user identity and role claims
func (h *AuthHandler) generateToken(user *model.User) (string, error) {
	claims := jwt.MapClaims{
		"user_id": user.ID,
		"role": user.Role,
		"exp": time.Now().Add(h.jwtExp).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(h.jwtSecret))
}