package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/viczem/userhub/services/userhub/internal/config"
	"github.com/viczem/userhub/services/userhub/internal/domain"
)

type serviceSuite struct {
	suite.Suite
	ctx  context.Context
	cfg  *config.Config
	db   *MockDatabase[Repository]
	repo *MockRepository
	srv  *Service[Repository]
}

func TestServiceSuite(t *testing.T) {
	suite.Run(t, new(serviceSuite))
}

func (s *serviceSuite) SetupTest() {
	s.ctx = s.T().Context()
	s.cfg = &config.Config{
		KeyringHMAC: domain.Keyring{
			ActiveID: 1,
			Keys: map[int16][domain.KeyringKeySize]byte{
				1: {1},
			},
		},
		ConfigSessionIdleTimeout: 5 * time.Minute,
		ConfigSessionTTL:         30 * time.Minute,
	}
	s.db = NewMockDatabase[Repository](s.T())
	s.repo = NewMockRepository(s.T())
	s.srv = NewService(s.cfg, s.db)
}

func (s *serviceSuite) assertRuntimeError(err, cause error, kind domain.ErrorKind, message string) {
	s.T().Helper()
	s.Require().Error(err)
	s.ErrorIs(err, cause)
	s.ErrorIs(err, kind)

	var domainErr *domain.Error
	s.Require().ErrorAs(err, &domainErr)
	s.Equal(message, domainErr.Message)
}
