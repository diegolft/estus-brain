package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rafael/estus-vault/backend/internal/domain"
	"github.com/rafael/estus-vault/backend/internal/service"
)

// VaultHandlers serves the password manager module: plain CRUD for vault
// entries plus revealing one in plaintext. It's intentionally a separate
// handler struct from Handlers so this module's routes can be reviewed and
// evolved without touching the ledger's. Every route sits behind a session;
// Reveal additionally re-confirms the account password, which is why this
// struct needs the auth service at all.
type VaultHandlers struct {
	vault *service.VaultService
	auth  *service.AuthService
}

func NewVaultHandlers(vault *service.VaultService, auth *service.AuthService) *VaultHandlers {
	return &VaultHandlers{vault: vault, auth: auth}
}

func (h *VaultHandlers) List(w http.ResponseWriter, r *http.Request) {
	entries, err := h.vault.List(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]vaultEntryDTO, len(entries))
	for i, e := range entries {
		out[i] = toVaultEntryDTO(e)
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *VaultHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req vaultEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrValidation))
		return
	}
	entry, err := h.vault.Create(r.Context(), req.toInput())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toVaultEntryDTO(entry))
}

// Update handles PUT /api/vault/{id}.
func (h *VaultHandlers) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req vaultEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrValidation))
		return
	}
	entry, err := h.vault.Update(r.Context(), id, req.toInput())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toVaultEntryDTO(entry))
}

// Delete handles DELETE /api/vault/{id}.
func (h *VaultHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.vault.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// Reveal handles POST /api/vault/{id}/reveal, decrypting and returning the
// entry's plaintext password. It requires the account password in the body
// on top of a valid session: this is what replaces the WebAuthn gate dropped
// in migration 0021. Holding a session is deliberately not enough — an
// unattended browser or a stolen token should not be able to read stored
// secrets in the clear, and re-typing the password is the cheapest proof
// that the person at the keyboard is still the owner.
func (h *VaultHandlers) Reveal(w http.ResponseWriter, r *http.Request) {
	user, ok := UserFromContext(r.Context())
	if !ok {
		writeUnauthorized(w, errors.New("authentication required"))
		return
	}
	var req revealRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, fmt.Errorf("%w: invalid JSON body", domain.ErrValidation))
		return
	}
	valid, err := h.auth.VerifyUserPassword(r.Context(), user.ID, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	if !valid {
		writeUnauthorized(w, errors.New("senha incorreta"))
		return
	}

	id := r.PathValue("id")
	password, err := h.vault.Reveal(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, revealResponseDTO{Password: password})
}
