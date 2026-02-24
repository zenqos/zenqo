package user

import (
	"net/http"

	"github.com/zenqos/zenqo/core"
)

// Controller handles HTTP requests for the /users resource.
type Controller struct {
	core.BaseController
	svc *Service
}

// NewController creates the user controller and registers all routes.
func NewController() *Controller {
	c := &Controller{svc: NewService()}
	c.SetBasePath("/users")

	c.GET("/", c.findAll)
	c.GET("/{id}", c.findOne)
	c.POST("/", c.create)
	c.PUT("/{id}", c.update)
	c.DELETE("/{id}", c.remove)

	return c
}

// GET /api/v1/users
// → 200 { "success": true, "data": [ { "id": 1, "name": "Alice", ... }, ... ] }
func (c *Controller) findAll(r *http.Request) (any, error) {
	return c.svc.FindAll(), nil
}

// GET /api/v1/users/{id}
// → 200 { "success": true, "data": { "id": 1, "name": "Alice", "email": "..." } }
// → 404 { "code": 404, "message": "user not found" }
func (c *Controller) findOne(r *http.Request) (any, error) {
	id, err := core.Param[int64](r, "id")
	if err != nil {
		return nil, err
	}
	u, err := c.svc.FindOne(id)
	if err != nil {
		return nil, core.ErrNotFound("user not found")
	}
	return u, nil
}

// POST /api/v1/users
// → 201 { "success": true, "data": { "id": 3, ... } }    ← 201 is automatic for POST
// → 400 { "code": 400, "message": "validation failed", "errors": [...] }
func (c *Controller) create(r *http.Request) (any, error) {
	dto, err := core.Bind[CreateUserDTO](r)
	if err != nil {
		return nil, err
	}
	return c.svc.Create(dto), nil
}

// PUT /api/v1/users/{id}
// → 200 { "success": true, "data": { ... } }
// → 404 { "code": 404, "message": "user not found" }
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
	return u, nil
}

// DELETE /api/v1/users/{id}
// → 204 No Content    ← automatic when nil is returned
// → 404 { "code": 404, "message": "user not found" }
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
