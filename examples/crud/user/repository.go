package user

import (
	"context"
	"fmt"
	"sync"
)

// Repository defines the data access contract for Users.
type Repository interface {
	FindAll(ctx context.Context) ([]User, error)
	FindByID(ctx context.Context, id string) (*User, error)
	Create(ctx context.Context, u *User) error
	Update(ctx context.Context, id string, u *User) error
	Delete(ctx context.Context, id string) error
}

// InMemoryRepository is a thread-safe in-memory implementation of Repository.
// Useful for prototyping and tests — replace with a real DB implementation in production.
type InMemoryRepository struct {
	mu    sync.RWMutex
	store map[string]*User
	seq   int
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{store: make(map[string]*User)}
}

func (r *InMemoryRepository) FindAll(_ context.Context) ([]User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users := make([]User, 0, len(r.store))
	for _, u := range r.store {
		users = append(users, *u)
	}
	return users, nil
}

func (r *InMemoryRepository) FindByID(_ context.Context, id string) (*User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.store[id]
	if !ok {
		return nil, fmt.Errorf("user %q not found", id)
	}
	return u, nil
}

func (r *InMemoryRepository) Create(_ context.Context, u *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.seq++
	u.ID = fmt.Sprintf("%d", r.seq)
	r.store[u.ID] = u
	return nil
}

func (r *InMemoryRepository) Update(_ context.Context, id string, patch *User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.store[id]
	if !ok {
		return fmt.Errorf("user %q not found", id)
	}
	if patch.Name != "" {
		u.Name = patch.Name
	}
	if patch.Email != "" {
		u.Email = patch.Email
	}
	return nil
}

func (r *InMemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.store[id]; !ok {
		return fmt.Errorf("user %q not found", id)
	}
	delete(r.store, id)
	return nil
}
