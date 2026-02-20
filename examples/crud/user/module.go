package user

import (
	"zenqo/core"
	"zenqo/examples/crud/auth"
)

type userModule struct {
	controllers []core.Controller
}

func (m *userModule) Name() string                   { return "UserModule" }
func (m *userModule) Controllers() []core.Controller { return m.controllers }

// NewModule wires the full dependency chain: Repository → Service → Handler.
// The TokenGuard is injected from outside so it can be shared across modules.
func NewModule(guard *auth.TokenGuard) core.Module {
	repo := NewInMemoryRepository()
	svc := NewService(repo)
	handler := NewHandler(svc, guard)
	return &userModule{controllers: []core.Controller{handler}}
}
