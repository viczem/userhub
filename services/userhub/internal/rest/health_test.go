package rest

import "net/http"

func (s *restSuite) TestLive() {
	response := s.request(http.MethodGet, "/health/live", nil)

	s.Equal(http.StatusOK, response.Code)
	s.Empty(response.Body.String())
	s.Equal("no-cache, no-store, no-transform, must-revalidate, private, max-age=0", response.Header().Get("Cache-Control"))
	s.srv.AssertNotCalled(s.T(), "HealthReady")
}

func (s *restSuite) TestReady() {
	s.srv.EXPECT().HealthReady().Return(true).Once()

	response := s.request(http.MethodGet, "/health/ready", nil)

	s.Equal(http.StatusOK, response.Code)
	s.Empty(response.Body.String())
}

func (s *restSuite) TestNotReady() {
	s.srv.EXPECT().HealthReady().Return(false).Once()

	response := s.request(http.MethodGet, "/health/ready", nil)

	s.Equal(http.StatusServiceUnavailable, response.Code)
	s.Empty(response.Body.String())
}
