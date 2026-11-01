package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/stretchr/testify/mock"
	"github.com/viczem/userhub/services/userhub/internal/domain"
)

func (s *restSuite) TestReadConfigUsersRoles() {
	roles := []domain.UserRole{
		{ID: 1, Name: "admin", Description: "Manage users"},
		{ID: 2, Name: "user", Description: "User role"},
	}
	s.srv.EXPECT().ReadUsersRoles(mock.MatchedBy(func(ctx context.Context) bool {
		return ctx.Value(domain.ConfigSessionTokenKey{}) == "config-token"
	})).Return(roles, nil).Once()

	response := s.requestWithHeaders(http.MethodGet, "/api/config/roles", nil, http.Header{
		"X-UserHub-Key": {"config-token"},
	})

	s.Equal(http.StatusOK, response.Code)
	s.Equal("application/json", response.Header().Get("Content-Type"))
	s.JSONEq(`[
		{"id":1,"name":"admin","description":"Manage users"},
		{"id":2,"name":"user","description":"User role"}
	]`, response.Body.String())
}

func (s *restSuite) TestReadConfigUsersRolesEmpty() {
	for _, tc := range []struct {
		name  string
		roles []domain.UserRole
	}{
		{name: "nil roles"},
		{name: "empty roles", roles: []domain.UserRole{}},
	} {
		s.Run(tc.name, func() {
			s.srv.EXPECT().ReadUsersRoles(mock.Anything).Return(tc.roles, nil).Once()

			response := s.requestWithHeaders(http.MethodGet, "/api/config/roles", nil, http.Header{
				"X-UserHub-Key": {"config-token"},
			})

			s.Equal(http.StatusOK, response.Code)
			s.Equal("application/json", response.Header().Get("Content-Type"))
			s.JSONEq(`[]`, response.Body.String())
		})
	}
}

func (s *restSuite) TestReadConfigUsersRolesError() {
	for _, tc := range []struct {
		name   string
		err    error
		status int
		body   string
	}{
		{
			name:   "invalid config session",
			err:    domain.ErrConfigSession,
			status: http.StatusNotFound,
		},
		{
			name:   "wrapped invalid config session",
			err:    fmt.Errorf("validate token: %w", domain.ErrConfigSession),
			status: http.StatusNotFound,
		},
		{
			name:   "service failure",
			err:    errors.New("read roles failed"),
			status: http.StatusInternalServerError,
			body:   `{"ok":false,"message":"Internal Server Error"}`,
		},
	} {
		s.Run(tc.name, func() {
			s.srv.EXPECT().ReadUsersRoles(mock.Anything).Return(nil, tc.err).Once()

			response := s.requestWithHeaders(http.MethodGet, "/api/config/roles", nil, http.Header{
				"X-UserHub-Key": {"config-token"},
			})

			s.Equal(tc.status, response.Code)
			s.Equal("application/json", response.Header().Get("Content-Type"))
			if tc.body == "" {
				s.Empty(response.Body.String())
				return
			}
			s.JSONEq(tc.body, response.Body.String())
		})
	}
}

func (s *restSuite) TestReadConfigUsersRolesWithoutToken() {
	response := s.request(http.MethodGet, "/api/config/roles", nil)

	s.Equal(http.StatusNotFound, response.Code)
	s.Equal("application/json", response.Header().Get("Content-Type"))
	s.Empty(response.Body.String())
	s.srv.AssertNotCalled(s.T(), "ReadUsersRoles", mock.Anything)
}

func (s *restSuite) TestReadConfigUsersRolesMethodNotAllowed() {
	response := s.requestWithHeaders(http.MethodDelete, "/api/config/roles", nil, http.Header{
		"X-UserHub-Key": {"config-token"},
	})

	s.Equal(http.StatusMethodNotAllowed, response.Code)
	s.Contains(response.Header().Values("Allow"), http.MethodGet)
	s.Contains(response.Header().Values("Allow"), http.MethodPost)
	s.srv.AssertNotCalled(s.T(), "ReadUsersRoles", mock.Anything)
	s.srv.AssertNotCalled(s.T(), "CreateUsersRole", mock.Anything, mock.Anything)
}

func (s *restSuite) TestCreateConfigUsersRole() {
	role := domain.UserRole{Name: "project-admin1", Description: "Manage users"}
	createdRole := role
	createdRole.ID = 3
	s.srv.EXPECT().CreateUsersRole(mock.MatchedBy(func(ctx context.Context) bool {
		return ctx.Value(domain.ConfigSessionTokenKey{}) == "config-token"
	}), role).Return(createdRole, nil).Once()

	response := s.requestWithHeaders(http.MethodPost, "/api/config/roles", strings.NewReader(
		`{"name":"project-admin1","description":"Manage users"}`,
	), http.Header{
		"X-UserHub-Key": {"config-token"},
		"Content-Type":  {"application/json"},
	})

	s.Equal(http.StatusOK, response.Code)
	s.Equal("application/json", response.Header().Get("Content-Type"))
	s.JSONEq(`{"id":3,"name":"project-admin1","description":"Manage users"}`, response.Body.String())
}

func (s *restSuite) TestCreateConfigUsersRoleInvalidName() {
	for _, tc := range []struct {
		name string
		rule string
	}{
		{name: "", rule: "required"},
		{name: "Admin", rule: "slug"},
		{name: "1admin", rule: "slug"},
		{name: "_admin", rule: "slug"},
		{name: "project_admin", rule: "slug"},
		{name: "-admin", rule: "slug"},
		{name: "admin-", rule: "slug"},
		{name: "project--admin", rule: "slug"},
		{name: "project admin", rule: "slug"},
		{name: "\u0430\u0434\u043c\u0438\u043d", rule: "slug"},
	} {
		s.Run(tc.name, func() {
			response := s.requestWithHeaders(http.MethodPost, "/api/config/roles", strings.NewReader(
				fmt.Sprintf(`{"name":%q}`, tc.name),
			), http.Header{
				"X-UserHub-Key": {"config-token"},
				"Content-Type":  {"application/json"},
			})

			s.Equal(http.StatusBadRequest, response.Code)
			s.JSONEq(fmt.Sprintf(`{
				"ok":false,"message":"Request validation failed",
				"fields":[{"field":"name","rule":%q,"param":""}]
			}`, tc.rule), response.Body.String())
			s.srv.AssertNotCalled(s.T(), "CreateUsersRole", mock.Anything, mock.Anything)
		})
	}
}
