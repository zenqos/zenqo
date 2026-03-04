package user

import (
	"context"
	"fmt"
	"net/http"

	"github.com/zenqos/zenqo/core"
	"my-app/internal/auth"
)

// Controller handles protected user routes.
// All routes require JWT authentication via the controller-level Guard.
type Controller struct {
	core.BaseController
	svc *Service
}

// NewController creates the user controller with JWT Guard applied to all routes.
func NewController(svc *Service, jwtSecret string) *Controller {
	c := &Controller{svc: svc}
	c.SetBasePath("/users")
	c.UseControllerGuard(&auth.JWTGuard{Secret: jwtSecret})
	c.UseControllerInterceptor(&LogInterceptor{})

	c.GET("/", c.findAll)
	c.GET("/me", c.me)
	c.GET("/{id}", c.findOne)
	c.PUT("/{id}", c.update)
	c.DELETE("/{id}", c.remove)

	return c
}

// GET /api/v1/users/
func (c *Controller) findAll(r *http.Request) (any, error) {
	return c.svc.FindAll(), nil
}

// GET /api/v1/users/me — returns the authenticated user
func (c *Controller) me(r *http.Request) (any, error) {
	claims := auth.GetClaims(r)
	if claims == nil {
		return nil, core.ErrUnauthorized("invalid token")
	}
	u, err := c.svc.FindOne(claims.UserID)
	if err != nil {
		return nil, core.ErrNotFound("user not found")
	}
	return u.Public(), nil
}

// GET /api/v1/users/{id}
func (c *Controller) findOne(r *http.Request) (any, error) {
	id, err := core.Param[int64](r, "id")
	if err != nil {
		return nil, err
	}
	u, err := c.svc.FindOne(id)
	if err != nil {
		return nil, core.ErrNotFound("user not found")
	}
	return u.Public(), nil
}

// PUT /api/v1/users/{id}
func (c *Controller) update(r *http.Request) (any, error) {
	id, err := core.Param[int64](r, "id")
	if err != nil {
		return nil, err
	}
	dto, err := core.Bind[UpdateUserDTO](r)
	if err != nil {
		return nil, err
	}
	u, err := c.svc.Update(id, dto)
	if err != nil {
		return nil, core.ErrNotFound("user not found")
	}
	return u.Public(), nil
}

// DELETE /api/v1/users/{id}
func (c *Controller) remove(r *http.Request) (any, error) {
	id, err := core.Param[int64](r, "id")
	if err != nil {
		return nil, err
	}
	if err := c.svc.Delete(id); err != nil {
		return nil, core.ErrNotFound("user not found")
	}
	return nil, nil
}

// LogInterceptor logs request method/path and the response status code.
type LogInterceptor struct{}

func (l *LogInterceptor) Before(ctx context.Context, r *http.Request) context.Context {
	core.Zlog("Request", r.Method+" "+r.URL.Path)
	return ctx
}

func (l *LogInterceptor) After(ctx context.Context, w http.ResponseWriter, statusCode int) {
	core.Zlog("Response", fmt.Sprintf("status=%d", statusCode))
}
