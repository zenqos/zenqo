package user

import (
	"encoding/json"
	"net/http"
	"strings"

	"zenqo/core"
	"zenqo/examples/crud/auth"
)

// UserHandler handles all routes under /users.
// It embeds core.BaseController to get the builder-style route API.
type UserHandler struct {
	core.BaseController
	service Service
}

func NewHandler(service Service, guard *auth.TokenGuard) *UserHandler {
	h := &UserHandler{service: service}
	h.SetBasePath("/users")

	// Public routes — no guard required
	h.GET("/", h.list)
	h.GET("/{id}", h.getOne)

	// Protected routes — require valid Bearer token
	h.POST("/", h.create).UseGuard(guard)
	h.PUT("/{id}", h.update).UseGuard(guard)
	h.DELETE("/{id}", h.delete).UseGuard(guard)

	return h
}

func (h *UserHandler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.GetAll(r.Context())
	if err != nil {
		core.InternalError(w, err.Error())
		return
	}
	core.OK(w, users)
}

func (h *UserHandler) getOne(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	user, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		core.NotFound(w, err.Error())
		return
	}
	core.OK(w, user)
}

func (h *UserHandler) create(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.BadRequest(w, "invalid request body")
		return
	}
	user, err := h.service.Create(r.Context(), req)
	if err != nil {
		core.InternalError(w, err.Error())
		return
	}
	core.Created(w, user)
}

func (h *UserHandler) update(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	var req UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		core.BadRequest(w, "invalid request body")
		return
	}
	user, err := h.service.Update(r.Context(), id, req)
	if err != nil {
		core.NotFound(w, err.Error())
		return
	}
	core.OK(w, user)
}

func (h *UserHandler) delete(w http.ResponseWriter, r *http.Request) {
	id := pathID(r)
	if err := h.service.Delete(r.Context(), id); err != nil {
		core.NotFound(w, err.Error())
		return
	}
	core.JSON(w, http.StatusNoContent, nil)
}

// pathID extracts the {id} segment from the URL path.
// Works without importing chi directly — reads the last path segment.
func pathID(r *http.Request) string {
	parts := strings.Split(strings.TrimSuffix(r.URL.Path, "/"), "/")
	return parts[len(parts)-1]
}
