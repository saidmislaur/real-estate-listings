package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"go.uber.org/zap"
)

type Service interface {
	CreateUser(ctx context.Context, req CreateUserRequest) (*User, error)
	LoginUser(ctx context.Context, req LoginUserRequest) (*User, error)
}

type ValidationError struct {
	Field   string //имя поля которое не прошло валидацию
	Message string // объяснение что именно не так (что за ошибка)
}

func (e ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}

	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (e ValidationError) Is(terget error) bool {
	_, ok := terget.(ValidationError)
	return ok
}

type service struct {
	repo   Repository
	logger *zap.Logger
}

type requestIDKey struct{}

func NewService(repo Repository, logger *zap.Logger) Service {
	l := logger.Named("user.service").With(zap.String("component", "user.service"))
	return &service{
		repo:   repo,
		logger: l,
	}
}

func (s service) CreateUser(ctx context.Context, req CreateUserRequest) (*User, error) {
	reqID, _ := ctx.Value(requestIDKey{}).(string)

	if strings.TrimSpace(req.Email) == "" {
		return nil, ValidationError{Field: "email", Message: ErrRequiredEmail.Error()}
	}
	if strings.TrimSpace(req.Password) == "" {
		return nil, ValidationError{Field: "password", Message: ErrRequiredPasswordHash.Error()}
	}
	if req.Role == "" {
		req.Role = RoleModerator
	}
	if !isValidRole(req.Role) {
		s.logger.Info("create user validation failed: invalid role",
			zap.String("request_id", reqID),
			zap.String("email", req.Email),
			zap.String("role", string(req.Role)),
		)
		return nil, ErrInvalidRole
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("create user: bcrypt generate failed",
			zap.String("request_id", reqID),
			zap.String("login", req.Email),
			zap.Error(err),
		)
		return nil, err
	}

	u := &User{
		Email:        req.Email,
		passwordHash: string(hash),
		Role:         req.Role,
	}

	if err := s.repo.Create(ctx, u); err != nil {
		if errors.Is(err, ErrEmailExists) {
			return nil, ErrEmailExists
		}
		s.logger.Error("create user failed",
			zap.String("request_id", reqID),
			zap.String("login", req.Email),
			zap.String("role", string(req.Role)),
			zap.Error(err),
		)
		return nil, err
	}

	s.logger.Info("user created",
		zap.String("request_id", reqID),
		zap.Int64("user_id", int64(u.ID)),
		zap.String("login", u.Email),
		zap.String("role", string(u.Role)),
	)

	return u, nil
}

func (s service) LoginUser(ctx context.Context, req LoginUserRequest) (*User, error) {
	reqID, _ := ctx.Value(requestIDKey{}).(string)

	if strings.TrimSpace(req.Email) == "" {
		return nil, ValidationError{Field: "email", Message: ErrRequiredEmail.Error()}
	}
	if strings.TrimSpace(req.Password) == "" {
		return nil, ValidationError{Field: "password", Message: ErrRequiredPasswordHash.Error()}
	}

	loginReq := LoginUserRequest{
		Email:    req.Email,
		Password: req.Password,
	}

	storedHash, err := s.repo.GetPasswordHashByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			s.logger.Warn("login attempt - credentials invalid",
				zap.String("request_id", reqID),
				zap.String("email", loginReq.Email),
			)
			return nil, ErrInvalidCredentials // ← никогда не говори "пользователь не найден"
		}
		s.logger.Error("cannot get password hash",
			zap.String("request_id", reqID),
			zap.String("email", loginReq.Email),
			zap.Error(err),
		)
		return nil, fmt.Errorf("login failed: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(loginReq.Password)); err != nil {
		s.logger.Warn("login attempt - invalid password",
			zap.String("request_id", reqID),
			zap.String("email", loginReq.Email),
		)
		return nil, ErrInvalidCredentials
	}

	user, err := s.repo.GetUserByEmail(ctx, loginReq.Email)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			s.logger.Warn("unsuccessful login attempt",
				zap.String("id", reqID),
				zap.String("email", loginReq.Email),
				zap.Error(err),
			)
			return nil, ErrInvalidCredentials
		}

		s.logger.Error("login failed",
			zap.String("id", reqID),
			zap.String("email", loginReq.Email),
			zap.Error(err),
		)
		return nil, fmt.Errorf("login operation failed: %w", err)
	}

	return user, nil
}

func isValidRole(r Role) bool {
	return r == RoleClient || r == RoleModerator
}
