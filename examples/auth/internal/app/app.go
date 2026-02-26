package app

import (
	"github.com/zenqos/zenqo/core"
	"github.com/zenqos/zenqo/middleware"
	"my-app/internal/auth"
	"my-app/internal/config"
	"my-app/internal/user"
)

// New creates the application and wires all controllers.
func New(cfg config.Config) *core.App {
	app := core.NewApp()
	app.Use(middleware.SecureHeaders())
	app.Use(middleware.CORS(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE"},
		AllowHeaders: []string{"Content-Type", "Authorization"},
	}))

	userSvc := user.NewService()

	return app.
		SetGlobalPrefix("/api/v1").
		UseController(
			auth.NewController(&userStoreAdapter{svc: userSvc}, cfg.JWTSecret),
			user.NewController(userSvc, cfg.JWTSecret),
		)
}

// userStoreAdapter adapts user.Service to the auth.UserStore interface,
// breaking the circular dependency between auth and user packages.
type userStoreAdapter struct {
	svc *user.Service
}

func (a *userStoreAdapter) FindByEmail(email string) *auth.UserInfo {
	u := a.svc.FindByEmail(email)
	if u == nil {
		return nil
	}
	return &auth.UserInfo{ID: u.ID, Email: u.Email, Password: u.Password}
}

func (a *userStoreAdapter) CreateUser(name, email, password string) *auth.UserInfo {
	u := a.svc.Create(user.CreateUserDTO{Name: name, Email: email, Password: password})
	return &auth.UserInfo{ID: u.ID, Email: u.Email, Password: u.Password}
}
