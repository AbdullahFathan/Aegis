package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"aegis/pkg/database"

	"github.com/google/uuid"
)

const cookieName = "refresh_token"

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type Service struct {
	Users  *Repository
	Store  RefreshStore
	Tokens Tokens
	TTL    time.Duration
}

func toView(u database.User) UserView {
	return UserView{
		ID:     u.ID.String(),
		Email:  u.Email,
		Name:   u.Name,
		Role:   string(u.Role),
		Status: string(u.Status),
	}
}

func (s *Service) Login(ctx context.Context, email, password string) (TokenResponse, string, error) {
	u, err := s.Users.FindByEmail(strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return TokenResponse{}, "", ErrInvalidCredentials
		}
		return TokenResponse{}, "", err
	}
	if u.Status != database.UserStatusActive || !CheckPassword(u.PasswordHash, password) {
		return TokenResponse{}, "", ErrInvalidCredentials
	}
	return s.issueSession(ctx, u)
}

func (s *Service) Refresh(ctx context.Context, raw string) (TokenResponse, string, error) {
	if raw == "" {
		return TokenResponse{}, "", ErrRefreshNotFound
	}
	userID, err := s.Store.Peek(ctx, raw)
	if err != nil {
		return TokenResponse{}, "", ErrRefreshNotFound
	}
	_ = s.Store.Delete(ctx, raw)
	u, err := s.Users.FindByID(userID)
	if err != nil || u.Status != database.UserStatusActive {
		return TokenResponse{}, "", ErrRefreshNotFound
	}
	return s.issueSession(ctx, u)
}

func (s *Service) Logout(ctx context.Context, raw string) error {
	if raw == "" {
		return nil
	}
	return s.Store.Delete(ctx, raw)
}

func (s *Service) Me(id uuid.UUID) (UserView, error) {
	u, err := s.Users.FindByID(id)
	if err != nil {
		return UserView{}, err
	}
	return toView(u), nil
}

func (s *Service) issueSession(ctx context.Context, u database.User) (TokenResponse, string, error) {
	access, err := s.Tokens.IssueAccess(u)
	if err != nil {
		return TokenResponse{}, "", err
	}
	refresh := uuid.NewString()
	if err := s.Store.Save(ctx, refresh, u.ID, s.TTL); err != nil {
		return TokenResponse{}, "", err
	}
	return TokenResponse{AccessToken: access, User: toView(u)}, refresh, nil
}
