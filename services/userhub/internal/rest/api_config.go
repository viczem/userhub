package rest

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/viczem/userhub/services/userhub/internal/domain"
	appservice "github.com/viczem/userhub/services/userhub/internal/service"
)

type configService interface {
	ReadUsersRoles(ctx context.Context) ([]domain.UserRole, error)
	CreateUsersRole(ctx context.Context, role domain.UserRole) (domain.UserRole, error)
	DeleteUsersRole(ctx context.Context, roleID int32) error
	PatchUsersRole(ctx context.Context, roleID int32, patch appservice.UserRolePatch) (domain.UserRole, error)
}

type usersRoleResponse struct {
	ID          int32  `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// readUsersRoles reads the configured user roles and writes them as JSON.
func (api *REST) readUsersRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := api.srv.ReadUsersRoles(r.Context())
	if err != nil {
		if errors.Is(err, domain.ErrConfigSession) {
			slog.ErrorContext(r.Context(), "validate config session", "error", err)
			// Return 404 intentionally to hide the existence of this endpoint.
			writeJSON(w, r, http.StatusNotFound, nil)
			return
		}

		slog.ErrorContext(r.Context(), "read users roles", "error", err)
		writeJSON(w, r, http.StatusInternalServerError, errorMessage(http.StatusText(http.StatusInternalServerError)))
		return
	}

	resp := make([]usersRoleResponse, 0, len(roles))
	for _, role := range roles {
		resp = append(resp, usersRoleResponse{
			ID:          role.ID,
			Name:        role.Name,
			Description: role.Description,
		})
	}

	writeJSON(w, r, http.StatusOK, resp)
}

func readUsersRoleID(w http.ResponseWriter, r *http.Request) (int32, error) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 32)
	if err != nil || id <= 0 {
		writeJSON(w, r, http.StatusNotFound, nil)
		return 0, errors.New("invalid user role ID")
	}

	return int32(id), nil
}

type usersRoleRequest struct {
	Name        string `json:"name" mod:"strip_html,trim" validate:"required,max=100,slug"`
	Description string `json:"description" mod:"strip_html,trim" validate:"max=500"`
}

// createUsersRole creates a new user role and writes it as JSON.
func (api *REST) createUsersRole(w http.ResponseWriter, r *http.Request) {
	var request usersRoleRequest
	if err := readAndValidateJSON(w, r, &request); err != nil {
		return
	}

	role := domain.UserRole{
		Name:        request.Name,
		Description: request.Description,
	}

	role, err := api.srv.CreateUsersRole(r.Context(), role)
	if err != nil {
		if errors.Is(err, domain.ErrConfigSession) {
			slog.ErrorContext(r.Context(), "validate config session", "error", err)
			writeJSON(w, r, http.StatusNotFound, nil)
			return
		}

		if errors.Is(err, domain.ErrUsersRoleConflict) {
			slog.ErrorContext(r.Context(), "create users role", "error", err)
			writeJSON(w, r, http.StatusConflict, errorMessage(err.Error()))
			return
		}

		slog.ErrorContext(r.Context(), "create users role", "error", err)
		writeJSON(w, r, http.StatusInternalServerError, errorMessage(http.StatusText(http.StatusInternalServerError)))
		return
	}

	writeJSON(w, r, http.StatusOK, usersRoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
	})
}

type usersRolePatchRequest struct {
	Name        *string `json:"name" mod:"strip_html,trim" validate:"omitnil,min=1,max=100,slug"`
	Description *string `json:"description" mod:"strip_html,trim" validate:"omitnil,max=500"`
}

func (api *REST) updateUsersRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := readUsersRoleID(w, r)
	if err != nil {
		return
	}

	var request usersRolePatchRequest
	if err := readAndValidateJSON(w, r, &request); err != nil {
		return
	}

	patch := appservice.UserRolePatch{
		Name:        request.Name,
		Description: request.Description,
	}

	role, err := api.srv.PatchUsersRole(r.Context(), roleID, patch)
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrConfigSession):
			slog.ErrorContext(r.Context(), "validate config session", "error", err)
			writeJSON(w, r, http.StatusNotFound, nil)
		case errors.Is(err, domain.ErrUsersRoleNotFound):
			slog.ErrorContext(r.Context(), "patch users role", "error", err)
			writeJSON(w, r, http.StatusNotFound, errorMessage(err.Error()))
		case errors.Is(err, domain.ErrUsersRoleConflict):
			slog.WarnContext(r.Context(), "patch users role", "error", err)
			writeJSON(w, r, http.StatusConflict, errorMessage(domain.ErrUsersRoleConflict.Error()))

		default:
			slog.ErrorContext(r.Context(), "patch users role", "error", err)
			writeJSON(w, r, http.StatusInternalServerError, errorMessage(http.StatusText(http.StatusInternalServerError)))
		}
		return
	}

	writeJSON(w, r, http.StatusOK, usersRoleResponse{
		ID:          role.ID,
		Name:        role.Name,
		Description: role.Description,
	})
}

func (api *REST) deleteUsersRole(w http.ResponseWriter, r *http.Request) {
	roleID, err := readUsersRoleID(w, r)
	if err != nil {
		return
	}

	if err := api.srv.DeleteUsersRole(r.Context(), roleID); err != nil {
		switch {
		case errors.Is(err, domain.ErrConfigSession):
			slog.ErrorContext(r.Context(), "validate config session", "error", err)
			writeJSON(w, r, http.StatusNotFound, nil)
		case errors.Is(err, domain.ErrUsersRoleNotFound):
			slog.ErrorContext(r.Context(), "delete users role", "error", err)
			writeJSON(w, r, http.StatusNotFound, errorMessage(err.Error()))

		default:
			slog.ErrorContext(r.Context(), "delete users role", "error", err)
			writeJSON(w, r, http.StatusInternalServerError, errorMessage(err.Error()))
		}
		return
	}

	writeJSON(w, r, http.StatusOK, successMessage(fmt.Sprintf("User role '%d' was deleted", roleID)))
}
