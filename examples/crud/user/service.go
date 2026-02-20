package user

import "context"

// Service defines the business logic contract for Users.
type Service interface {
	GetAll(ctx context.Context) ([]User, error)
	GetByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, req CreateUserRequest) (*User, error)
	Update(ctx context.Context, id string, req UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, id string) error
}

type serviceImpl struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return &serviceImpl{repo: repo}
}

func (s *serviceImpl) GetAll(ctx context.Context) ([]User, error) {
	return s.repo.FindAll(ctx)
}

func (s *serviceImpl) GetByID(ctx context.Context, id string) (*User, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *serviceImpl) Create(ctx context.Context, req CreateUserRequest) (*User, error) {
	u := &User{Name: req.Name, Email: req.Email}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *serviceImpl) Update(ctx context.Context, id string, req UpdateUserRequest) (*User, error) {
	if err := s.repo.Update(ctx, id, &User{Name: req.Name, Email: req.Email}); err != nil {
		return nil, err
	}
	return s.repo.FindByID(ctx, id)
}

func (s *serviceImpl) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
