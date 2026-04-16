package auth

import (
	"context"
	"fmt"
	"net/mail"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type TokenManager interface {
	Issue(principal Principal) (string, error)
	Parse(token string) (Principal, error)
}

type Service struct {
	repo   Repository
	tokens TokenManager
}

func NewService(repo Repository, tokens TokenManager) *Service {
	return &Service{repo: repo, tokens: tokens}
}

func (s *Service) DummyLogin(_ context.Context, role Role) (*AuthResponse, error) {
	if !isValidRole(role) {
		return nil, ErrInvalidRole
	}

	token, err := s.tokens.Issue(Principal{Role: role})
	if err != nil {
		return nil, fmt.Errorf("issue dummy token: %w", err)
	}

	return &AuthResponse{Token: token, Role: role}, nil
}

func (s *Service) Register(ctx context.Context, req RegisterRequest) (*AuthResponse, error) {
	if err := validateCredentials(req.Email, req.Password, req.Role); err != nil {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &User{
		Email:        strings.TrimSpace(strings.ToLower(req.Email)),
		PasswordHash: string(hash),
		Role:         req.Role,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.issueForUser(user)
}

func (s *Service) Login(ctx context.Context, req LoginRequest) (*AuthResponse, error) {
	email := strings.TrimSpace(strings.ToLower(req.Email))
	if email == "" {
		return nil, ErrEmailRequired
	}
	if strings.TrimSpace(req.Password) == "" {
		return nil, ErrPasswordRequired
	}

	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		if err == ErrNotFound {
			return nil, ErrInvalidCredentials
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return s.issueForUser(user)
}

func (s *Service) ParseToken(token string) (Principal, error) {
	return s.tokens.Parse(token)
}

func (s *Service) issueForUser(user *User) (*AuthResponse, error) {
	token, err := s.tokens.Issue(Principal{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
	})
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}

	return &AuthResponse{Token: token, Role: user.Role}, nil
}

func validateCredentials(email, password string, role Role) error {
	if strings.TrimSpace(email) == "" {
		return ErrEmailRequired
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return ErrInvalidEmail
	}
	if strings.TrimSpace(password) == "" {
		return ErrPasswordRequired
	}
	if !isValidRole(role) {
		return ErrInvalidRole
	}
	return nil
}

func isValidRole(role Role) bool {
	return role == RoleClient || role == RoleModerator
}
