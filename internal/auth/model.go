package auth

import "time"

type Role string

const (
	RoleClient    Role = "client"
	RoleModerator Role = "moderator"
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Principal struct {
	UserID int64  `json:"user_id,omitempty"`
	Email  string `json:"email,omitempty"`
	Role   Role   `json:"role"`
}

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type DummyLoginRequest struct {
	Role Role `json:"role"`
}

type AuthResponse struct {
	Token string `json:"token"`
	Role  Role   `json:"role"`
}
