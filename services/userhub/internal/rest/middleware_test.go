package rest

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/viczem/userhub/services/userhub/internal/config"
)

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
	s.Equal("Internal Server Error\n", response.Body.String())
}

func (s *restSuite) TestDevelopmentPropagatesPanic() {
	s.srv.EXPECT().HealthReady().Run(func() {
		panic("private service failure")
	}).Return(false).Once()

	s.PanicsWithValue("private service failure", func() {
		s.request(http.MethodGet, "/health/ready", nil)
	})
}
