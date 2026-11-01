package rest

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/viczem/userhub/services/userhub/internal/config"
	"github.com/viczem/userhub/services/userhub/internal/domain"
)

func (s *restSuite) TestValidateConfigSessionMiddleware() {
	for _, tc := range []struct {
		name  string
		token string
		want  string
	}{
		{name: "missing token"},
		{name: "whitespace token", token: " \t "},
		{name: "token", token: "config-token", want: "config-token"},
		{name: "trimmed token", token: " \t config-token \t ", want: "config-token"},
	} {
		s.Run(tc.name, func() {
			called := false
			handler := validateConfigSessionMiddleware()(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				called = true
				s.Equal(tc.want, r.Context().Value(domain.ConfigSessionTokenKey{}))
				w.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodGet, "/api/config/test", nil).WithContext(s.T().Context())
			if tc.token != "" {
				request.Header.Set("X-UserHub-Key", tc.token)
			}
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			if tc.want == "" {
				s.Equal(http.StatusNotFound, response.Code)
				s.Equal("application/json", response.Header().Get("Content-Type"))
				s.Empty(response.Body.String())
				s.False(called)
				return
			}

			s.Equal(http.StatusNoContent, response.Code)
			s.Empty(response.Body.String())
			s.True(called)
		})
	}
}

func (s *restSuite) TestBodyAtLimit() {
	s.srv.EXPECT().HealthReady().Return(true).Once()

	response := s.request(http.MethodGet, "/health/ready", strings.NewReader(strings.Repeat("a", s.cfg.HTTP.MaxBodyBytes)))

	s.Equal(http.StatusOK, response.Code)
}

func (s *restSuite) TestBodyOverLimit() {
	response := s.request(http.MethodGet, "/health/ready", strings.NewReader(strings.Repeat("a", s.cfg.HTTP.MaxBodyBytes+1)))

	s.Equal(http.StatusRequestEntityTooLarge, response.Code)
	s.Equal("request body too large\n", response.Body.String())
	s.srv.AssertNotCalled(s.T(), "HealthReady")
}

func (s *restSuite) TestBodyOverLimitWithoutContentLength() {
	request := httptest.NewRequest(http.MethodGet, "/health/ready", strings.NewReader(strings.Repeat("a", s.cfg.HTTP.MaxBodyBytes+1)))
	request.ContentLength = -1
	response := httptest.NewRecorder()

	s.api.ServeHTTP(response, request)

	s.Equal(http.StatusRequestEntityTooLarge, response.Code)
	s.Equal("request body too large\n", response.Body.String())
	s.srv.AssertNotCalled(s.T(), "HealthReady")
}

func (s *restSuite) TestProductionRecoversPanic() {
	s.cfg.AppEnv = config.AppEnvProduction
	s.api = NewREST(s.cfg, s.srv)
	s.srv.EXPECT().HealthReady().Run(func() {
		panic("private service failure")
	}).Return(false).Once()

	response := s.request(http.MethodGet, "/health/ready", nil)

	s.Equal(http.StatusInternalServerError, response.Code)
	s.Equal("application/json", response.Header().Get("Content-Type"))
	s.Empty(response.Body.String())
}

func (s *restSuite) TestDevelopmentPropagatesPanic() {
	s.srv.EXPECT().HealthReady().Run(func() {
		panic("private service failure")
	}).Return(false).Once()

	s.PanicsWithValue("private service failure", func() {
		s.request(http.MethodGet, "/health/ready", nil)
	})
}
