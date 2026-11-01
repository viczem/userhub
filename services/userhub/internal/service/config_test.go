package service

import (
	"context"
	"errors"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/viczem/userhub/services/userhub/internal/domain"
)

func (s *serviceSuite) newConfigSession() *domain.RuntimeSession {
	s.T().Helper()
	session, err := domain.NewRuntimeSession(s.cfg.KeyringHMAC, s.cfg.ConfigSessionIdleTimeout, s.cfg.ConfigSessionTTL)
	s.Require().NoError(err)
	return session
}

func (s *serviceSuite) TestReadConfigUsersRoles() {
	for _, tc := range []struct {
		name  string
		roles []domain.UserRole
	}{
		{
			name: "configured roles",
			roles: []domain.UserRole{
				{Name: "admin", Description: "Manage users"},
				{Name: "user", Description: "User role"},
			},
		},
		{name: "nil roles"},
		{name: "empty roles", roles: []domain.UserRole{}},
	} {
		s.Run(tc.name, func() {
			db := NewMockDatabase[Repository](s.T())
			repo := NewMockRepository(s.T())
			srv := NewService(s.cfg, db)
			session := s.newConfigSession()
			session.ExpiresAt = time.Now().Add(time.Minute)
			original := *session

			var persisted domain.RuntimeSession

			open := db.EXPECT().NewRepository(s.ctx).Return(repo, nil).Once()
			read := repo.EXPECT().ReadRuntimeSession(s.ctx).Return(session, nil).Once()
			update := repo.EXPECT().UpdateRuntimeSession(s.ctx, session).
				Run(func(_ context.Context, session *domain.RuntimeSession) {
					persisted = *session
				}).Return(nil).Once()
			readRoles := repo.EXPECT().ReadUsersRoles(s.ctx).Return(tc.roles, nil).Once()
			commit := repo.EXPECT().Commit(s.ctx).Return(nil).Once()
			closeRepo := repo.EXPECT().Close(s.ctx).Return().Once()
			mock.InOrder(open, read, update, readRoles, commit, closeRepo)

			before := time.Now()
			roles, err := srv.ReadUsersRoles(s.ctx)
			after := time.Now()

			s.Require().NoError(err)
			s.Equal(tc.roles, roles)
			s.True(persisted.ExpiresAt.After(original.ExpiresAt))
			s.False(persisted.ExpiresAt.Before(before.Add(s.cfg.ConfigSessionIdleTimeout)))
			s.False(persisted.ExpiresAt.After(after.Add(s.cfg.ConfigSessionIdleTimeout)))
			s.Equal(original.ValidUntil, persisted.ValidUntil)
			s.Equal(original.Token, persisted.Token)
			s.Equal(original.TokenHMAC, persisted.TokenHMAC)
			s.Equal(original.TokenKeyID, persisted.TokenKeyID)
		})
	}
}

func (s *serviceSuite) TestReadConfigUsersRolesDatabaseError() {
	cause := errors.New("begin transaction failed")
	s.db.EXPECT().NewRepository(s.ctx).Return(nil, cause).Once()

	roles, err := s.srv.ReadUsersRoles(s.ctx)

	s.Nil(roles)
	s.ErrorIs(err, cause)
}

func (s *serviceSuite) TestReadConfigUsersRolesSessionReadError() {
	cause := errors.New("read runtime session failed")
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	read := s.repo.EXPECT().ReadRuntimeSession(s.ctx).Return(nil, cause).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, read, closeRepo)

	roles, err := s.srv.ReadUsersRoles(s.ctx)

	s.Nil(roles)
	s.Require().ErrorIs(err, cause)

	s.repo.AssertNotCalled(s.T(), "UpdateRuntimeSession", mock.Anything, mock.Anything)
	s.repo.AssertNotCalled(s.T(), "ReadUsersRoles", mock.Anything)
}

func (s *serviceSuite) TestReadConfigUsersRolesInvalidSession() {
	for _, tc := range []struct {
		name   string
		mutate func(*domain.RuntimeSession)
	}{
		{name: "empty token", mutate: func(session *domain.RuntimeSession) { session.Token = "" }},
		{name: "empty HMAC", mutate: func(session *domain.RuntimeSession) { session.TokenHMAC = "" }},
		{name: "token mismatch", mutate: func(session *domain.RuntimeSession) { session.Token = "invalid-token" }},
		{name: "unknown key", mutate: func(session *domain.RuntimeSession) { session.TokenKeyID = 2 }},
		{name: "idle timeout expired", mutate: func(session *domain.RuntimeSession) {
			session.ExpiresAt = time.Now().Add(-time.Minute)
		}},
		{name: "TTL expired", mutate: func(session *domain.RuntimeSession) {
			session.ValidUntil = time.Now().Add(-time.Minute)
		}},
	} {
		s.Run(tc.name, func() {
			db := NewMockDatabase[Repository](s.T())
			repo := NewMockRepository(s.T())
			srv := NewService(s.cfg, db)
			session := s.newConfigSession()
			tc.mutate(session)
			original := *session

			open := db.EXPECT().NewRepository(s.ctx).Return(repo, nil).Once()
			read := repo.EXPECT().ReadRuntimeSession(s.ctx).Return(session, nil).Once()
			deleteSession := repo.EXPECT().DeleteRuntimeSession(s.ctx).Return(nil).Once()
			commit := repo.EXPECT().Commit(s.ctx).Return(nil).Once()
			closeRepo := repo.EXPECT().Close(s.ctx).Return().Once()
			mock.InOrder(open, read, deleteSession, commit, closeRepo)

			roles, err := srv.ReadUsersRoles(s.ctx)

			s.Nil(roles)
			s.Require().ErrorIs(err, domain.ErrConfigSession)
			s.Equal(original, *session)
			repo.AssertNotCalled(s.T(), "UpdateRuntimeSession", mock.Anything, mock.Anything)
			repo.AssertNotCalled(s.T(), "ReadUsersRoles", mock.Anything)
		})
	}
}

func (s *serviceSuite) TestReadConfigUsersRolesProlongError() {
	session := s.newConfigSession()
	session.ValidUntil = time.Now().Add(time.Minute)
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	read := s.repo.EXPECT().ReadRuntimeSession(s.ctx).Return(session, nil).Once()
	deleteSession := s.repo.EXPECT().DeleteRuntimeSession(s.ctx).Return(nil).Once()
	commit := s.repo.EXPECT().Commit(s.ctx).Return(nil).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, read, deleteSession, commit, closeRepo)

	roles, err := s.srv.ReadUsersRoles(s.ctx)

	s.Nil(roles)
	s.Require().ErrorIs(err, domain.ErrConfigSession)
	s.Require().ErrorContains(err, "prolongation expired")
	s.repo.AssertNotCalled(s.T(), "UpdateRuntimeSession", mock.Anything, mock.Anything)
	s.repo.AssertNotCalled(s.T(), "ReadUsersRoles", mock.Anything)
}

func (s *serviceSuite) TestReadConfigUsersRolesSessionUpdateError() {
	cause := errors.New("update runtime session failed")
	session := s.newConfigSession()
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	read := s.repo.EXPECT().ReadRuntimeSession(s.ctx).Return(session, nil).Once()
	update := s.repo.EXPECT().UpdateRuntimeSession(s.ctx, session).Return(cause).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, read, update, closeRepo)

	roles, err := s.srv.ReadUsersRoles(s.ctx)

	s.Nil(roles)
	s.Require().ErrorIs(err, cause)
	s.repo.AssertNotCalled(s.T(), "ReadUsersRoles", mock.Anything)
}

func (s *serviceSuite) TestReadConfigUsersRolesRepositoryError() {
	cause := errors.New("read roles failed")
	session := s.newConfigSession()
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	read := s.repo.EXPECT().ReadRuntimeSession(s.ctx).Return(session, nil).Once()
	update := s.repo.EXPECT().UpdateRuntimeSession(s.ctx, session).Return(nil).Once()
	readRoles := s.repo.EXPECT().ReadUsersRoles(s.ctx).
		Return([]domain.UserRole{{Name: "partial result"}}, cause).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, read, update, readRoles, closeRepo)

	roles, err := s.srv.ReadUsersRoles(s.ctx)

	s.Nil(roles)
	s.ErrorIs(err, cause)
}
