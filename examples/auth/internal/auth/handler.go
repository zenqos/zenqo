package auth

import (
	"net/http"

	"github.com/zenqos/zenqo/core"
)

// UserStore is the interface that auth needs from the user layer.
// This avoids a circular import between auth and user packages.
type UserStore interface {
	FindByEmail(email string) *UserInfo
	CreateUser(name, email, password string) *UserInfo
}

// UserInfo is a minimal user representation for auth purposes.
type UserInfo struct {
	ID       int64
	Email    string
	Password string
}

// Controller handles authentication routes: login and register.
type Controller struct {
	core.BaseController
	store  UserStore
	secret string
}

// NewController creates the auth controller with login/register routes.
func NewController(store UserStore, secret string) *Controller {
	c := &Controller{store: store, secret: secret}
	c.SetBasePath("/auth")

	c.POST("/register", c.register)
	c.POST("/login", c.login)

	return c
}

// POST /api/v1/auth/register
func (c *Controller) register(r *http.Request) (any, error) {
	dto, err := core.Bind[RegisterDTO](r)
	if err != nil {
		return nil, err
	}

	if c.store.FindByEmail(dto.Email) != nil {
		return nil, core.ErrBadRequest("email already registered")
	}

	u := c.store.CreateUser(dto.Name, dto.Email, dto.Password)

	token, err := GenerateToken(c.secret, u.ID, u.Email)
	if err != nil {
		return nil, core.ErrInternal("failed to generate token")
	}

	return map[string]any{
		"user": map[string]any{
			"id":    u.ID,
			"email": u.Email,
		},
		"token": token,
	}, nil
}

// POST /api/v1/auth/login
func (c *Controller) login(r *http.Request) (any, error) {
	dto, err := core.Bind[LoginDTO](r)
	if err != nil {
		return nil, err
	}

	u := c.store.FindByEmail(dto.Email)
	if u == nil || u.Password != dto.Password {
		return nil, core.ErrUnauthorized("invalid email or password")
	}

	token, err := GenerateToken(c.secret, u.ID, u.Email)
	if err != nil {
		return nil, core.ErrInternal("failed to generate token")
	}

	return map[string]any{
		"user": map[string]any{
			"id":    u.ID,
			"email": u.Email,
		},
		"token": token,
	}, nil
}
