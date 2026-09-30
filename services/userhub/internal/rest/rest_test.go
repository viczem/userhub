package rest

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/viczem/userhub/services/userhub/internal/config"
)

type restSuite struct {
	suite.Suite
	cfg *config.Config
	srv *Mockservice
	api *REST
}

func TestRESTSuite(t *testing.T) {
	suite.Run(t, new(restSuite))
}

func (s *restSuite) SetupTest() {
	s.cfg = &config.Config{
		AppEnv: config.AppEnvDevelopment,
		HTTP: config.HTTPConfig{
			MaxBodyBytes: 8192,
		},
	}
	s.srv = NewMockservice(s.T())
	s.api = NewREST(s.cfg, s.srv)
}

func (s *restSuite) request(method, target string, body io.Reader) *httptest.ResponseRecorder {
	s.T().Helper()

	request := httptest.NewRequest(method, target, body).WithContext(s.T().Context())
	response := httptest.NewRecorder()
	s.api.ServeHTTP(response, request)

	return response
}

func (s *restSuite) TestNotFound() {
	response := s.request(http.MethodGet, "/unknown", nil)

	s.Equal(http.StatusNotFound, response.Code)
	s.srv.AssertNotCalled(s.T(), "HealthReady")
}

func (s *restSuite) TestMethodNotAllowed() {
	response := s.request(http.MethodPost, "/health/ready", nil)

	s.Equal(http.StatusMethodNotAllowed, response.Code)
	s.Contains(response.Header().Values("Allow"), http.MethodGet)
	s.srv.AssertNotCalled(s.T(), "HealthReady")
}
