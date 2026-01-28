package user

type CreateUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     Role   `json:"role"`
}

type CreateUserResponse struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
	Role  Role   `json:"role"`
}
