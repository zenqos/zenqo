package user

// User is the API response entity.
// Zenqo automatically converts PascalCase → camelCase in JSON responses:
//
//	ID → "id", Name → "name", Email → "email"
//
// No struct tags needed.
type User struct {
	ID    int64
	Name  string
	Email string
}

// CreateUserDTO is the request body for POST /users.
// Expected JSON: { "name": "John", "email": "john@example.com" }
type CreateUserDTO struct {
	Name  string `validate:"required,min=2,max=50"`
	Email string `validate:"required,email"`
}

// UpdateUserDTO is the request body for PUT /users/{id}.
// Pointer fields distinguish "not sent" (nil) from "set to empty" (*"").
type UpdateUserDTO struct {
	Name  *string `validate:"max=50"`
	Email *string `validate:"email"`
}
