package httpapi

import (
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/franklinlem/flagpilot/apps/api/internal/auth"
)

func (h *authHandlers) changePassword(w http.ResponseWriter, r *http.Request) {
	if !h.validOrigin(r) {
		problem(w, r, 403, "Origem não permitida")
		return
	}
	token := cookieToken(r)
	identity, err := h.service.Find(r.Context(), token)
	if err != nil {
		h.authError(w, r, err)
		return
	}
	if !auth.ValidCSRF(token, r.Header.Get("X-CSRF-Token")) {
		problem(w, r, 403, "Token CSRF inválido")
		return
	}
	var body struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if !decodeJSON(w, r, &body) {
		return
	}
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	if !h.passwordLimiter.Allow(ip, identity.User.ID) {
		h.authError(w, r, auth.ErrBusy)
		return
	}
	err = h.service.ChangePassword(r.Context(), token, body.CurrentPassword, body.NewPassword)
	switch {
	case errors.Is(err, auth.ErrCredentials):
		problem(w, r, 403, "Senha atual incorreta")
		return
	case errors.Is(err, auth.ErrValidation):
		problem(w, r, 422, "Informe a senha atual e uma nova senha diferente, com pelo menos 12 caracteres e até 256 bytes UTF-8")
		return
	case err != nil:
		h.authError(w, r, err)
		return
	}
	h.setCookie(w, "", time.Unix(1, 0))
	w.WriteHeader(http.StatusNoContent)
}
