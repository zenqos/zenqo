package user

// User is the entity returned in API responses.
// No struct tags needed — Zenqo automatically converts field names to camelCase.
type User struct {
	ID    int64
	Name  string
	Email string
}

// CreateUserDTO is the request body for POST /users.
type CreateUserDTO struct {
	Name  string
	Email string
}

// UpdateUserDTO is the request body for PUT /users/:id.
// Empty fields are ignored (partial update).
type UpdateUserDTO struct {
	Name  string
	Email string
}
