package service

import (
	"context"
	"errors"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/viczem/userhub/services/userhub/internal/domain"
)

func (s *serviceSuite) TestCreateRuntimeSession() {
	var persisted *domain.RuntimeSession
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	create := s.repo.EXPECT().CreateRuntimeSession(s.ctx, mock.Anything).
		Run(func(_ context.Context, session *domain.RuntimeSession) {
			persisted = session
		}).Return(nil).Once()
	commit := s.repo.EXPECT().Commit(s.ctx).Return(nil).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, create, commit, closeRepo)

	before := time.Now()
	session, err := s.srv.CreateRuntimeSession(s.ctx)
	after := time.Now()

	s.Require().NoError(err)
	s.Require().NotNil(session)
	s.Same(persisted, session)
	s.NotEmpty(session.Token)
	s.NotEmpty(session.TokenHMAC)
	s.NotEqual(session.Token, session.TokenHMAC)
	s.Equal(s.cfg.KeyringHMAC.ActiveID, session.TokenKeyID)
	s.False(session.ExpiresAt.Before(before.Add(s.cfg.ConfigSessionIdleTimeout)))
	s.False(session.ExpiresAt.After(after.Add(s.cfg.ConfigSessionIdleTimeout)))
	s.Equal(s.cfg.ConfigSessionTTL-s.cfg.ConfigSessionIdleTimeout, session.ValidUntil.Sub(session.ExpiresAt))
}

func (s *serviceSuite) TestCreateRuntimeSessionGenerationError() {
	s.cfg.KeyringHMAC = domain.Keyring{}

	session, err := s.srv.CreateRuntimeSession(s.ctx)

	s.Nil(session)
	s.assertRuntimeError(err, domain.ErrKeyring, "runtime create session", "new runtime session")
	s.db.AssertNotCalled(s.T(), "NewRepository", mock.Anything)
}

func (s *serviceSuite) TestCreateRuntimeSessionDatabaseError() {
	cause := errors.New("begin transaction failed")
	s.db.EXPECT().NewRepository(s.ctx).Return(nil, cause).Once()

	session, err := s.srv.CreateRuntimeSession(s.ctx)

	s.Nil(session)
	s.assertRuntimeError(err, cause, "runtime create session", "new repository")
}

func (s *serviceSuite) TestCreateRuntimeSessionRepositoryError() {
	cause := errors.New("persist runtime session failed")
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	create := s.repo.EXPECT().CreateRuntimeSession(s.ctx, mock.Anything).Return(cause).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, create, closeRepo)

	session, err := s.srv.CreateRuntimeSession(s.ctx)

	s.Nil(session)
	s.assertRuntimeError(err, cause, "runtime create session", "repo - create runtime session")
	s.repo.AssertNotCalled(s.T(), "Commit", mock.Anything)
}

func (s *serviceSuite) TestCreateRuntimeSessionCommitError() {
	cause := errors.New("commit failed")
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	create := s.repo.EXPECT().CreateRuntimeSession(s.ctx, mock.Anything).Return(nil).Once()
	commit := s.repo.EXPECT().Commit(s.ctx).Return(cause).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, create, commit, closeRepo)

	session, err := s.srv.CreateRuntimeSession(s.ctx)

	s.Nil(session)
	s.assertRuntimeError(err, cause, "runtime create session", "commit")
}

func (s *serviceSuite) TestDeleteRuntimeSession() {
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	deleteSession := s.repo.EXPECT().DeleteRuntimeSession(s.ctx).Return(nil).Once()
	commit := s.repo.EXPECT().Commit(s.ctx).Return(nil).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, deleteSession, commit, closeRepo)

	err := s.srv.DeleteRuntimeSession(s.ctx)

	s.NoError(err)
}

func (s *serviceSuite) TestDeleteRuntimeSessionDatabaseError() {
	cause := errors.New("begin transaction failed")
	s.db.EXPECT().NewRepository(s.ctx).Return(nil, cause).Once()

	err := s.srv.DeleteRuntimeSession(s.ctx)

	s.assertRuntimeError(err, cause, "runtime delete session", "new repository")
}

func (s *serviceSuite) TestDeleteRuntimeSessionRepositoryError() {
	cause := errors.New("delete runtime session failed")
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	deleteSession := s.repo.EXPECT().DeleteRuntimeSession(s.ctx).Return(cause).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, deleteSession, closeRepo)

	err := s.srv.DeleteRuntimeSession(s.ctx)

	s.assertRuntimeError(err, cause, "runtime delete session", "repo - delete runtime session")
	s.repo.AssertNotCalled(s.T(), "Commit", mock.Anything)
}

func (s *serviceSuite) TestDeleteRuntimeSessionCommitError() {
	cause := errors.New("commit failed")
	open := s.db.EXPECT().NewRepository(s.ctx).Return(s.repo, nil).Once()
	deleteSession := s.repo.EXPECT().DeleteRuntimeSession(s.ctx).Return(nil).Once()
	commit := s.repo.EXPECT().Commit(s.ctx).Return(cause).Once()
	closeRepo := s.repo.EXPECT().Close(s.ctx).Return().Once()
	mock.InOrder(open, deleteSession, commit, closeRepo)

	err := s.srv.DeleteRuntimeSession(s.ctx)

	s.assertRuntimeError(err, cause, "runtime delete session", "commit")
}
