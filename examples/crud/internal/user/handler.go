package user

import (
	"net/http"
	"strconv"

	"github.com/ftery0/zenqo/core"
)

// Controller handles HTTP requests for the /users resource.
type Controller struct {
	core.BaseController
	svc *Service
}

// NewController wires the controller and registers all routes.
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

// GET /users
func (c *Controller) findAll(r *http.Request) (any, error) {
	return c.svc.FindAll(), nil
}

// GET /users/:id
func (c *Controller) findOne(r *http.Request) (any, error) {
	id, err := strconv.ParseInt(core.URLParam(r, "id"), 10, 64)
	if err != nil {
		return nil, core.ErrBadRequest("invalid user id")
	}
	user, err := c.svc.FindOne(id)
	if err != nil {
		return nil, core.ErrNotFound("user not found")
	}
	return user, nil
}

// POST /users  →  201 Created (automatic)
func (c *Controller) create(r *http.Request) (any, error) {
	dto, err := core.Bind[CreateUserDTO](r)
	if err != nil {
		return nil, err
	}
	return c.svc.Create(dto), nil
}

// PUT /users/:id
func (c *Controller) update(r *http.Request) (any, error) {
	id, err := strconv.ParseInt(core.URLParam(r, "id"), 10, 64)
	if err != nil {
		return nil, core.ErrBadRequest("invalid user id")
	}
	dto, err := core.Bind[UpdateUserDTO](r)
	if err != nil {
		return nil, err
	}
	user, err := c.svc.Update(id, dto)
	if err != nil {
		return nil, core.ErrNotFound("user not found")
	}
	return user, nil
}

// DELETE /users/:id  →  204 No Content (automatic)
func (c *Controller) remove(r *http.Request) (any, error) {
	id, err := strconv.ParseInt(core.URLParam(r, "id"), 10, 64)
	if err != nil {
		return nil, core.ErrBadRequest("invalid user id")
	}
	if err := c.svc.Delete(id); err != nil {
		return nil, core.ErrNotFound("user not found")
	}
	return nil, nil
}
