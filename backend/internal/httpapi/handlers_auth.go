package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/rafael/estus-vault/backend/internal/domain"
	"github.com/rafael/estus-vault/backend/internal/service"
)

// AuthHandlers serves the login flow. Only Login is reachable without a
// session; the router mounts the rest behind requireSession, which is why
// none of them re-check the token themselves.
type AuthHandlers struct {
	auth *service.AuthService
}

func NewAuthHandlers(auth *service.AuthService) *AuthHandlers {
	return &AuthHandlers{auth: auth}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authUserDTO struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

func toAuthUserDTO(u domain.User) authUserDTO {
	return authUserDTO{ID: u.ID, Email: u.Email, Name: u.Name}
}

type loginResponse struct {
	Token     string      `json:"token"`
	ExpiresAt string      `json:"expires_at"`
	User      authUserDTO `json:"user"`
}

// Login handles POST /api/auth/login. Every credential failure answers with
// the same 401 and the same message: writeError is bypassed here because it
// maps by sentinel and would happily describe which half was wrong.
func (h *AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrValidation))
		return
	}
	token, expiresAt, user, err := h.auth.Login(r.Context(), req.Email, req.Password, r.UserAgent())
	if err != nil {
		writeUnauthorized(w, err)
		return
	}
	writeJSON(w, http.StatusOK, loginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z07:00"),
		User:      toAuthUserDTO(user),
	})
}

// Logout handles POST /api/auth/logout. It answers 204 even when the token
// was already gone — the caller's goal is to end up logged out, and failing
// a repeat logout would only give the frontend an error it can't act on.
func (h *AuthHandlers) Logout(w http.ResponseWriter, r *http.Request) {
	if err := h.auth.Logout(r.Context(), bearerToken(r)); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// Me handles GET /api/auth/me, reading the user the middleware already
// resolved rather than hitting the database a second time.
func (h *AuthHandlers) Me(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, errors.New("not authenticated"))
		return
	}
	writeJSON(w, http.StatusOK, toAuthUserDTO(user))
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

// ChangePassword handles POST /api/auth/password.
func (h *AuthHandlers) ChangePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, errors.New("not authenticated"))
		return
	}
	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrValidation))
		return
	}
	err := h.auth.ChangePassword(r.Context(), user.ID, req.CurrentPassword, req.NewPassword)
	if errors.Is(err, service.ErrInvalidCredentials) {
		writeUnauthorized(w, err)
		return
	}
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

type verifyPasswordRequest struct {
	Password string `json:"password"`
}

type verifyPasswordResponse struct {
	OK bool `json:"ok"`
}

// VerifyPassword handles POST /api/auth/verify-password. A wrong password is
// a 200 with ok:false rather than a 401: this backs a re-confirmation prompt
// on a screen the user is already authorized to see, so a failure here means
// "try typing it again", not "your session ended".
func (h *AuthHandlers) VerifyPassword(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, errors.New("not authenticated"))
		return
	}
	var req verifyPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrValidation))
		return
	}
	valid, err := h.auth.VerifyUserPassword(r.Context(), user.ID, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, verifyPasswordResponse{OK: valid})
}

// writeUnauthorized exists because writeError has no 401 case: it maps
// domain sentinels, and authentication failures deliberately don't carry one
// — they must all look identical from the outside.
func writeUnauthorized(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusUnauthorized, errorBody{Error: err.Error()})
}

// bearerToken pulls the token out of "Authorization: Bearer <token>",
// returning "" for anything that doesn't match that shape.
func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
