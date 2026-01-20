package user

import "time"

type Role string

const (
	RoleModerator Role = "moderator"
	RoleClient    Role = "client"
)

type User struct {
	ID           int
	Email        string
	passwordHash string
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}
