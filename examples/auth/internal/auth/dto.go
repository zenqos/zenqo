package auth

// LoginDTO is the request body for POST /auth/login.
type LoginDTO struct {
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=6"`
}

// RegisterDTO is the request body for POST /auth/register.
type RegisterDTO struct {
	Name     string `validate:"required,min=2,max=50"`
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=6,max=72"`
}
